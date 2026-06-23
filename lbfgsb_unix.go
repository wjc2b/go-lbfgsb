//go:build !windows

// Copyright (c) 2014 Aubrey Barnard.  This is free software.  See
// LICENSE.txt for details.

package lbfgsb

// #cgo LDFLAGS: -lm
// #include "lbfgsb_go_interface.h"
import "C"

import (
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

// Minimize optimizes the given objective using the L-BFGS-B algorithm.
// Implements OptimizationFunctionMinimizer.Minimize.
func (lbfgsb *Lbfgsb) Minimize(
	objective FunctionWithGradient,
	initialPoint []float64) (
	minimum PointValueGradient,
	exitStatus ExitStatus) {

	// Make sure object has been initialized
	lbfgsb.Init(len(initialPoint))

	// Check dimensionality
	dim := len(initialPoint)
	dim_c := C.int(dim)
	if lbfgsb.dimensionality != dim {
		exitStatus.Code = USAGE_ERROR
		exitStatus.Message = fmt.Sprintf("Lbfgsb: Dimensionality of the initial point (%d) does not match the dimensionality of the solver (%d).", dim, lbfgsb.dimensionality)
		return
	}

	// Set up bounds control.  Use a C-compatible type.
	boundsControl := make([]C.int, dim)
	if lbfgsb.lowerBounds != nil {
		for index, bound := range lbfgsb.lowerBounds {
			if !math.IsNaN(bound) && !math.IsInf(bound, -1) {
				boundsControl[index] = C.int(1)
			}
		}
	}
	if lbfgsb.upperBounds != nil {
		for index, bound := range lbfgsb.upperBounds {
			if !math.IsNaN(bound) && !math.IsInf(bound, -1) {
				// Map 0 -> 3, 1 -> 2
				boundsControl[index] = C.int(3 - boundsControl[index])
			}
		}
	}

	// Set up lower and upper bounds.  These must be different slices
	// than the ones in the Lbfgsb object because those must remain
	// unallocated if no bounds are specified.
	lowerBounds := makeCCopySlice_Float(lbfgsb.lowerBounds, dim)
	upperBounds := makeCCopySlice_Float(lbfgsb.upperBounds, dim)

	// Set up callbacks for function, gradient, and logging
	cId := registerCallback(objective)
	defer unregisterCallback(cId)
	callbackData_c := unsafe.Pointer(cId)
	var doLogging_c C.int                        // false
	var logFunctionCallbackData_c unsafe.Pointer // null
	var loggerId uintptr
	if lbfgsb.logger != nil {
		doLogging_c = C.int(1) // true
		loggerId = registerCallback(lbfgsb.logger)
		defer unregisterCallback(loggerId)
		logFunctionCallbackData_c = unsafe.Pointer(loggerId)
	}

	// Allocate arrays for return value
	minimum.X = make([]float64, dim)
	minimum.G = make([]float64, dim)

	// Convert parameters for C
	approximationSize_c := C.int(lbfgsb.approximationSize)
	fTolerance_c := C.double(lbfgsb.fTolerance)
	gTolerance_c := C.double(lbfgsb.gTolerance)
	printControl_c := C.int(lbfgsb.printControl)

	// Prepare buffers and arrays for C.  Avoid allocation in C land by
	// allocating compatible things in Go and passing their addresses.
	// The following arrays may not be interoperably type-safe but this
	// is how they did it on the Cgo page: http://golang.org/cmd/cgo/.
	// (One could always allocate slices of C types, pass those, and
	// then copy out and convert the contents on return.)
	var boundsControl_c *C.int = &boundsControl[0]
	var lowerBounds_c *C.double = &lowerBounds[0]
	var upperBounds_c *C.double = &upperBounds[0]
	var x0_c *C.double = (*C.double)(&initialPoint[0])
	var minX_c *C.double = (*C.double)(&minimum.X[0])
	var minF_c *C.double = (*C.double)(&minimum.F)
	var minG_c *C.double = (*C.double)(&minimum.G[0])
	var iters_c, evals_c C.int
	// Status message
	statusMessageLength_c := C.int(bufferSize)
	var statusMessageBuffer [bufferSize]C.char
	statusMessage_c := (*C.char)(&statusMessageBuffer[0])

	// Call the actual L-BFGS-B procedure
	statusCode_c := C.lbfgsb_minimize_c(
		callbackData_c, dim_c,
		boundsControl_c, lowerBounds_c, upperBounds_c,
		approximationSize_c, fTolerance_c, gTolerance_c,
		x0_c, minX_c, minF_c, minG_c, &iters_c, &evals_c,
		printControl_c, doLogging_c, logFunctionCallbackData_c,
		statusMessage_c, statusMessageLength_c,
	)

	// Convert outputs
	// Exit status codes match between ExitStatusCode and the C enum
	exitStatus.Code = ExitStatusCode(statusCode_c)
	exitStatus.Message = C.GoString(statusMessage_c)
	// Minimum already populated because pointers to its members were
	// passed into C/Fortran

	// Save statistics
	lbfgsb.statistics.Iterations = int(iters_c)
	lbfgsb.statistics.FunctionEvaluations = int(evals_c)
	// Number of function and gradient evaluations is always the same
	lbfgsb.statistics.GradientEvaluations = lbfgsb.statistics.FunctionEvaluations

	return
}

// makeCCopySlice_Float creates a C copy of a Go slice.  If the Go slice
// is nil, then a slice of the given length is created.
func makeCCopySlice_Float(slice []float64, sliceLen int) (
	slice_c []C.double) {

	slice_c = make([]C.double, sliceLen)
	// Copy the Go slice to the C slice, converting elements
	if slice != nil {
		for i := 0; i < sliceLen; i++ {
			slice_c[i] = C.double(slice[i])
		}
	}
	return
}

// go_objective_function_callback is an adapter between the C callback
// and the Go callback for evaluating the objective function.  Exported
// to C for use as a function pointer.  Must match the signature of
// objective_function_type in lbfgsb_c.h.
//
//export go_objective_function_callback
func go_objective_function_callback(
	dim_c C.int, point_c, value_c *C.double,
	callbackData_c unsafe.Pointer,
	statusMessage_c *C.char, statusMessageLength_c C.int) (
	statusCode_c C.int) {

	var point []float64

	// Convert inputs
	dim := int(dim_c)
	wrapCArrayAsGoSlice_Float64(point_c, dim, &point)
	objective := lookupCallback(uintptr(callbackData_c)).(FunctionWithGradient)

	// Evaluate the objective function.  Let panics propagate through
	// C/Fortran.
	value := objective.EvaluateFunction(point)

	// Convert outputs
	*value_c = C.double(value)

	//fmt.Printf("go_objective_function_callback: %v; %v;\n", point, value)

	return
}

// go_objective_gradient_callback is an adapter between the C callback
// and the Go callback for evaluating the objective gradient.  Exported
// to C for use as a function pointer.  Must match the signature of
// objective_gradient_type in lbfgsb_c.h.
//
//export go_objective_gradient_callback
func go_objective_gradient_callback(
	dim_c C.int, point_c, gradient_c *C.double,
	callbackData_c unsafe.Pointer,
	statusMessage_c *C.char, statusMessageLength_c C.int) (
	statusCode_c C.int) {

	var point, gradient, gradRet []float64

	// Convert inputs
	dim := int(dim_c)
	wrapCArrayAsGoSlice_Float64(point_c, dim, &point)
	objective := lookupCallback(uintptr(callbackData_c)).(FunctionWithGradient)

	// Evaluate the gradient of the objective function.  Let panics
	// propagate through C/Fortran.
	gradRet = objective.EvaluateGradient(point)

	// Convert outputs
	wrapCArrayAsGoSlice_Float64(gradient_c, dim, &gradient)
	copy(gradient, gradRet)

	//fmt.Printf("go_objective_gradient_callback: %v; %v;\n", point, gradient)

	return
}

// go_log_function_callback is an adapter between the C callback and the
// Go callback for logging information about each iteration.  Exported
// to C for use as a function pointer.  Must match the signature of
// lbfgsb_log_function_type in lbfgsb_c.h.
//
//export go_log_function_callback
func go_log_function_callback(
	logCallbackData_c unsafe.Pointer,
	iteration_c, fgEvals_c, fgEvalsTotal_c C.int, stepLength_c C.double,
	dim_c C.int, x_c *C.double, f_c C.double, g_c *C.double,
	fDelta_c, fDeltaBound_c, gNorm_c, gNormBound_c C.double) (
	statusCode_c C.int) {

	var x, g []float64

	// Convert inputs
	dim := int(dim_c)
	wrapCArrayAsGoSlice_Float64(x_c, dim, &x)
	wrapCArrayAsGoSlice_Float64(g_c, dim, &g)

	// Get the logging function from the callback data
	logger := lookupCallback(uintptr(logCallbackData_c)).(OptimizationIterationLogger)

	// Call the logging function.  Let panics propagate through
	// C/Fortran.
	logger(
		&OptimizationIterationInformation{
			Iteration:   int(iteration_c),
			FEvals:      int(fgEvals_c),
			GEvals:      int(fgEvals_c),
			FEvalsTotal: int(fgEvalsTotal_c),
			GEvalsTotal: int(fgEvalsTotal_c),
			StepLength:  float64(stepLength_c),
			X:           x,
			F:           float64(f_c),
			G:           g,
			FDelta:      float64(fDelta_c),
			FDeltaBound: float64(fDeltaBound_c),
			GNorm:       float64(gNorm_c),
			GNormBound:  float64(gNormBound_c),
		})

	return
}

// wrapCArrayAsGoSlice_Float64 allows a C array to be treated as a Go
// slice.  Based on https://code.google.com/p/go-wiki/wiki/cgo.  This
// only works if the Go and C types happen to be interoperable (binary
// compatible), but that seems to be the case so far.
func wrapCArrayAsGoSlice_Float64(array *C.double, length int,
	slice *[]float64) {

	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(slice))
	sliceHeader.Cap = length
	sliceHeader.Len = length
	sliceHeader.Data = uintptr(unsafe.Pointer(array))
}
