//go:build windows && amd64

package lbfgsb

import (
	_ "embed"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// DLL embedded at compile time — extracted to a temp file on first load.
//
//go:embed lbfgsb.dll
var dllData []byte

var (
	dllPath      string
	dllLbfgsb    *syscall.DLL
	procMinimize *syscall.Proc
)

func init() {
	// Extract the embedded DLL to a temp file so LoadLibrary can find it.
	// Use a stable name under TEMP — only write once per binary version.
	tmpDir := os.TempDir()
	dllPath = filepath.Join(tmpDir, "go_lbfgsb.dll")
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		if err := os.WriteFile(dllPath, dllData, 0644); err != nil {
			panic(fmt.Sprintf("lbfgsb: failed to write embedded DLL to %s: %v", dllPath, err))
		}
	}
	dllLbfgsb = syscall.MustLoadDLL(dllPath)
	procMinimize = dllLbfgsb.MustFindProc("lbfgsb_minimize_windows")
}

// lbfgsbCall matches the C struct lbfgsb_call layout exactly.
type lbfgsbCall struct {
	objFn             uintptr // function pointer
	gradFn            uintptr
	callbackData      uintptr
	dim               int32
	_pad0             int32 // align bounds_control to 8
	boundsControl     uintptr
	lowerBounds       uintptr
	upperBounds       uintptr
	approximationSize int32
	_pad1             int32 // align f_tolerance to 8
	fTolerance        float64
	gTolerance        float64
	initialPoint      uintptr
	minX              uintptr
	minF              uintptr
	minG              uintptr
	iters             uintptr
	evals             uintptr
	fortranPrintCtrl  int32
	doLogging         int32
	logFn             uintptr
	logCallbackData   uintptr
	statusMessage     uintptr
	statusMsgLen      int32
	_pad2             int32 // final padding
}

// Minimize optimizes the given objective using the L-BFGS-B algorithm.
// On Windows this calls into the embedded lbfgsb.dll via syscall.
func (lbfgsb *Lbfgsb) Minimize(
	objective FunctionWithGradient,
	initialPoint []float64) (
	minimum PointValueGradient,
	exitStatus ExitStatus) {

	// Make sure object has been initialized
	lbfgsb.Init(len(initialPoint))

	dim := len(initialPoint)
	if lbfgsb.dimensionality != dim {
		exitStatus.Code = USAGE_ERROR
		exitStatus.Message = fmt.Sprintf("Lbfgsb: Dimensionality of the initial point (%d) does not match the dimensionality of the solver (%d).", dim, lbfgsb.dimensionality)
		return
	}

	// Register callbacks and build function pointers via syscall.NewCallback.
	cId := registerCallback(objective)
	defer unregisterCallback(cId)
	objCB := syscall.NewCallback(objCallback)
	gradCB := syscall.NewCallback(gradCallback)

	// Bounds control: int32 slice.
	boundsControl := make([]int32, dim)
	if lbfgsb.lowerBounds != nil {
		for i, b := range lbfgsb.lowerBounds {
			if !math.IsNaN(b) && !math.IsInf(b, -1) {
				boundsControl[i] = 1
			}
		}
	}
	if lbfgsb.upperBounds != nil {
		for i, b := range lbfgsb.upperBounds {
			if !math.IsNaN(b) && !math.IsInf(b, -1) {
				boundsControl[i] = int32(3 - boundsControl[i])
			}
		}
	}

	// Lower / upper bounds — copy to C-compatible arrays.
	lowerBounds := make([]float64, dim)
	upperBounds := make([]float64, dim)
	if lbfgsb.lowerBounds != nil {
		copy(lowerBounds, lbfgsb.lowerBounds)
	}
	if lbfgsb.upperBounds != nil {
		copy(upperBounds, lbfgsb.upperBounds)
	}

	// Output arrays
	minimum.X = make([]float64, dim)
	minimum.G = make([]float64, dim)
	var iters, evals int32

	// Status message buffer
	statusBuf := make([]byte, bufferSize)

	// Pack everything into the C-compatible struct.
	call := lbfgsbCall{
		objFn:             objCB,
		gradFn:            gradCB,
		callbackData:      uintptr(cId),
		dim:               int32(dim),
		_pad0:             0,
		boundsControl:     uintptr(unsafe.Pointer(&boundsControl[0])),
		lowerBounds:       uintptr(unsafe.Pointer(&lowerBounds[0])),
		upperBounds:       uintptr(unsafe.Pointer(&upperBounds[0])),
		approximationSize: int32(lbfgsb.approximationSize),
		_pad1:             0,
		fTolerance:        lbfgsb.fTolerance,
		gTolerance:        lbfgsb.gTolerance,
		initialPoint:      uintptr(unsafe.Pointer(&initialPoint[0])),
		minX:              uintptr(unsafe.Pointer(&minimum.X[0])),
		minF:              uintptr(unsafe.Pointer(&minimum.F)),
		minG:              uintptr(unsafe.Pointer(&minimum.G[0])),
		iters:             uintptr(unsafe.Pointer(&iters)),
		evals:             uintptr(unsafe.Pointer(&evals)),
		fortranPrintCtrl:  int32(lbfgsb.printControl),
		doLogging:         0, // logging disabled on Windows (float args in callback not supported by NewCallback)
		logFn:             0,
		logCallbackData:   0,
		statusMessage:     uintptr(unsafe.Pointer(&statusBuf[0])),
		statusMsgLen:      int32(bufferSize),
		_pad2:             0,
	}

	// Call the DLL.
	statusCode, _, _ := procMinimize.Call(uintptr(unsafe.Pointer(&call)))

	exitStatus.Code = ExitStatusCode(statusCode)
	if statusBuf[0] != 0 {
		// C string in statusBuf, find null terminator
		n := 0
		for n < len(statusBuf) && statusBuf[n] != 0 {
			n++
		}
		exitStatus.Message = string(statusBuf[:n])
	}

	// Save statistics
	lbfgsb.statistics.Iterations = int(iters)
	lbfgsb.statistics.FunctionEvaluations = int(evals)
	lbfgsb.statistics.GradientEvaluations = lbfgsb.statistics.FunctionEvaluations

	return
}

// objCallback is the Windows callback for objective function evaluation.
// Signature matches lbfgsb_objective_function_type:
//
//	int (*)(int dim, double *point, double *value, void *data, char *msg, int len)
func objCallback(dim uintptr, pointPtr uintptr, valuePtr uintptr, cbData uintptr, msgPtr uintptr, msgLen uintptr) uintptr {
	n := int(dim)
	point := unsafe.Slice((*float64)(unsafe.Pointer(pointPtr)), n)
	objective := lookupCallback(cbData).(FunctionWithGradient)

	value := objective.EvaluateFunction(point)
	*(*float64)(unsafe.Pointer(valuePtr)) = value
	return 0
}

// gradCallback is the Windows callback for gradient evaluation.
// Signature matches lbfgsb_objective_gradient_type:
//
//	int (*)(int dim, double *point, double *gradient, void *data, char *msg, int len)
func gradCallback(dim uintptr, pointPtr uintptr, gradPtr uintptr, cbData uintptr, msgPtr uintptr, msgLen uintptr) uintptr {
	n := int(dim)
	point := unsafe.Slice((*float64)(unsafe.Pointer(pointPtr)), n)
	gradient := unsafe.Slice((*float64)(unsafe.Pointer(gradPtr)), n)
	objective := lookupCallback(cbData).(FunctionWithGradient)

	gradRet := objective.EvaluateGradient(point)
	copy(gradient, gradRet)
	return 0
}
