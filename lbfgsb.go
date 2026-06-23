// Copyright (c) 2014 Aubrey Barnard.  This is free software.  See
// LICENSE.txt for details.

// Go package that provides an interface to the Fortran implementation
// of the L-BFGS-B optimization algorithm.  The Fortran code is provided
// as a C-compatible library and this is the Go API for that library.
//
// Platform support:
//
//	darwin/amd64, darwin/arm64, linux/amd64, linux/arm64 —
//	  cgo-based, precompiled .syso included.
//	windows/amd64 —
//	  zero-dependency, loads a precompiled lbfgsb.dll via syscall.

package lbfgsb

import (
	"fmt"
	"math"
	"sync"
)

// Private constants
const (
	// 3 or so 80-character lines
	bufferSize = 250
)

// NewLbfgsb creates, initializes, and returns a new Lbfgsb solver
// object.  Equivalent to 'new(Lbfgsb).Init(dimensionality)'.  A
// zero-value Lbfgsb object is valid and needs no explicit construction.
// However, this constructor is convenient and explicit.
func NewLbfgsb(dimensionality int) *Lbfgsb {
	return new(Lbfgsb).Init(dimensionality)
}

// Lbfgsb provides the functionality of the Fortran L-BFGS-B optimizer
// as a Go object.  A Lbfgsb solver object contains the setup for an
// optimization problem of a particular dimensionality.  It stores
// bounds, parameters, and results so it is relatively lightweight
// (especially if no bounds are specified).  It can be re-used for other
// problems with the same dimensionality, but using a different solver
// object for each problem is probably better organization.  A
// zero-value Lbfgsb object is valid and needs no explicit construction.
// A solver object will perform unconstrained optimization unless bounds
// are set.
type Lbfgsb struct {
	// Dimensionality of the problem.  Zero is an invalid
	// dimensionality, so this also serves as an indicator of whether
	// this object has been initialized for computation.  Once the
	// dimensionality has been set it cannot be changed.  To be ready
	// for computation the following must be greater than zero:
	// dimensionality, approximationSize, fTolerance, gTolerance.
	dimensionality int

	// Problem specification.  Bounds may be nil or allocated fully.
	// Individual bounds may be omitted by placing NaNs or Infs.
	lowerBounds []float64
	upperBounds []float64

	// Parameters
	approximationSize int
	fTolerance        float64
	gTolerance        float64
	printControl      int

	// Logging
	logger OptimizationIterationLogger

	// Statistics (do not embed or members will be public)
	statistics OptimizationStatistics
}

// Init initializes this Lbfgsb solver for problems of the given
// dimensionality.  Also sets default parameters that are not zero
// values.  Returns this for method chaining.  Ignores calls subsequent
// to the first (because a solver is intended for only a particular
// dimensionality).
func (lbfgsb *Lbfgsb) Init(dimensionality int) *Lbfgsb {
	// Only initialize if not previously initialized
	if lbfgsb.dimensionality == 0 {
		// Check for a valid dimensionality
		if dimensionality <= 0 {
			panic(fmt.Errorf("Lbfgsb: Optimization problem dimensionality %d <= 0.  Expected > 0.", dimensionality))
		}
		// Set up the solver.  Protect previous values so Init can be
		// called after other methods.
		lbfgsb.dimensionality = dimensionality
		if lbfgsb.approximationSize == 0 {
			lbfgsb.approximationSize = 5
		}
		if lbfgsb.fTolerance == 0.0 {
			lbfgsb.fTolerance = 1e-6
		}
		if lbfgsb.gTolerance == 0.0 {
			lbfgsb.gTolerance = 1e-6
		}
	}
	return lbfgsb
}

// SetBounds sets the upper and lower bounds on the individual
// dimensions to the given intervals resulting in a constrained
// optimization problem.  Individual bounds may be (+/-)Inf.
func (lbfgsb *Lbfgsb) SetBounds(bounds [][2]float64) *Lbfgsb {
	// Ensure object is initialized
	lbfgsb.Init(len(bounds))
	// Check dimensionality
	if lbfgsb.dimensionality != len(bounds) {
		panic(fmt.Errorf("Lbfgsb: Dimensionality of the bounds (%d) does not match the dimensionality of the solver (%d).", len(bounds), lbfgsb.dimensionality))
	}

	lbfgsb.lowerBounds = make([]float64, lbfgsb.dimensionality)
	lbfgsb.upperBounds = make([]float64, lbfgsb.dimensionality)
	for i, interval := range bounds {
		lbfgsb.lowerBounds[i] = interval[0]
		lbfgsb.upperBounds[i] = interval[1]
	}
	return lbfgsb
}

// SetBoundsAll sets the bounds of all the dimensions to [lower,upper].
// Init must be called first to set the dimensionality.
func (lbfgsb *Lbfgsb) SetBoundsAll(lower, upper float64) *Lbfgsb {
	// Check object has been initialized
	if lbfgsb.dimensionality == 0 {
		panic(fmt.Errorf("Lbfgsb: Init() must be called before SetAllBounds()."))
	}

	lbfgsb.lowerBounds = make([]float64, lbfgsb.dimensionality)
	lbfgsb.upperBounds = make([]float64, lbfgsb.dimensionality)
	for i := 0; i < lbfgsb.dimensionality; i++ {
		lbfgsb.lowerBounds[i] = lower
		lbfgsb.upperBounds[i] = upper
	}
	return lbfgsb
}

// SetBoundsSparse sets the bounds to only those in the given map;
// others are unbounded.  Each entry in the map is a (zero-based)
// dimension index mapped to a slice representing an interval.
// Individual bounds may be (+/-)Inf.  Init must be called first to set
// the dimensionality.
//
// The slice is interpreted as an interval as follows:
//
//	nil | []: [-Inf, +Inf]
//	[x]: [-|x|, |x|]
//	[l, u, ...]: [l, u]
func (lbfgsb *Lbfgsb) SetBoundsSparse(sparseBounds map[int][]float64) *Lbfgsb {
	// Check object has been initialized
	if lbfgsb.dimensionality == 0 {
		panic(fmt.Errorf("Lbfgsb: Init() must be called before SetAllBounds()."))
	}

	// If no bounds are given, clear the bounds
	if sparseBounds == nil || len(sparseBounds) == 0 {
		return lbfgsb.ClearBounds()
	}

	lbfgsb.lowerBounds = make([]float64, lbfgsb.dimensionality)
	lbfgsb.upperBounds = make([]float64, lbfgsb.dimensionality)
	nInf := math.Inf(-1)
	pInf := math.Inf(+1)
	for i := 0; i < lbfgsb.dimensionality; i++ {
		interval, exists := sparseBounds[i]
		if exists {
			if interval == nil || len(interval) == 0 {
				lbfgsb.lowerBounds[i] = nInf
				lbfgsb.upperBounds[i] = pInf
			} else if len(interval) == 1 {
				lbfgsb.upperBounds[i] = math.Abs(interval[0])
				lbfgsb.lowerBounds[i] = -lbfgsb.upperBounds[i]
			} else {
				lbfgsb.lowerBounds[i] = interval[0]
				lbfgsb.upperBounds[i] = interval[1]
			}
		} else {
			lbfgsb.lowerBounds[i] = nInf
			lbfgsb.upperBounds[i] = pInf
		}
	}
	return lbfgsb
}

// ClearBounds clears all bounds resulting in an unconstrained
// optimization problem.
func (lbfgsb *Lbfgsb) ClearBounds() *Lbfgsb {
	lbfgsb.lowerBounds = nil
	lbfgsb.upperBounds = nil
	return lbfgsb
}

// SetApproximationSize sets the amount of history (points and
// gradients) stored and used to approximate the inverse Hessian matrix.
// More history allows better approximation at the cost of more memory.
// The recommended range is [3,20].  Defaults to 5.
func (lbfgsb *Lbfgsb) SetApproximationSize(size int) *Lbfgsb {
	if size <= 0 {
		panic(fmt.Errorf("Lbfgsb: Approximation size %d <= 0.  Expected > 0.", size))
	}
	lbfgsb.approximationSize = size
	return lbfgsb
}

// SetFTolerance sets the tolerance of the precision of the objective
// function required for convergence.  Defaults to 1e-6.
func (lbfgsb *Lbfgsb) SetFTolerance(fTolerance float64) *Lbfgsb {
	if fTolerance <= 0.0 {
		panic(fmt.Errorf("Lbfgsb: F tolerance %g <= 0.  Expected > 0.", fTolerance))
	}
	lbfgsb.fTolerance = fTolerance
	return lbfgsb
}

// SetGTolerance sets the tolerance of the precision of the objective
// gradient required for convergence.  Defaults to 1e-6.
func (lbfgsb *Lbfgsb) SetGTolerance(gTolerance float64) *Lbfgsb {
	if gTolerance <= 0.0 {
		panic(fmt.Errorf("Lbfgsb: G tolerance %g <= 0.  Expected > 0.", gTolerance))
	}
	lbfgsb.gTolerance = gTolerance
	return lbfgsb
}

// SetFortranPrintControl sets the level of output verbosity from the
// Fortran L-BFGS-B code.  Defaults to 0, no output.  Ranges from 0 to
// 102: 1 displays a summary, 100 displays details of each iteration,
// 102 adds vectors (X and G) to the output.
func (lbfgsb *Lbfgsb) SetFortranPrintControl(verbosity int) *Lbfgsb {
	if verbosity < 0 {
		panic(fmt.Errorf("Lbfgsb: Print control %d < 0.  Expected >= 0.", verbosity))
	}
	lbfgsb.printControl = verbosity
	return lbfgsb
}

// SetLogger sets a logging function for the optimization that will be
// called after each iteration.  May be nil, which disables logging.
// Defaults to nil.
func (lbfgsb *Lbfgsb) SetLogger(
	logger OptimizationIterationLogger) *Lbfgsb {

	lbfgsb.logger = logger
	return lbfgsb
}

// OptimizationStatistics returns some statistics about the most recent
// minimization: the total number of iterations and the total numbers of
// function and gradient evaluations.
func (lbfgsb *Lbfgsb) OptimizationStatistics() OptimizationStatistics {
	return lbfgsb.statistics
}

// callbackFunctions is a container for the actual objective functions and
// related data.
var callbackFunctions = make(map[uintptr]interface{})

// callbackIndex stores an index to use for new callback function.
var callbackIndex uintptr

// callbackMutex is a mutex preventing simultanious access to callback
// and callbackIds.
var callbackMutex sync.Mutex

// registerCallback registers a new callback and returns its' index
// (>=1).
func registerCallback(f interface{}) uintptr {
	callbackMutex.Lock()
	defer callbackMutex.Unlock()
	// We always increment callbackIndex to have more or less
	// unique ids. This way it is easier to debug problems with
	// reusing unregistered ids.
	callbackIndex++
	startIndex := callbackIndex
	for callbackIndex == 0 || callbackFunctions[callbackIndex] != nil {
		// Find the first free non-zero index.
		callbackIndex++
		// If the map is full, i.e. all non-zero uintptrs were
		// used, we do not want to loop infinitely. We check
		// if we already encountered the starting index. If
		// so, we panic. In practice this is very unlikely to
		// have this kind of problem since all the objects are
		// unregistered at the end of the function call.
		if callbackIndex == startIndex {
			panic("no more space in the map to store a callback function")
		}
	}
	callbackFunctions[callbackIndex] = f
	return callbackIndex
}

// lookupCallback returns a callback function given an index.
func lookupCallback(i uintptr) interface{} {
	callbackMutex.Lock()
	defer callbackMutex.Unlock()
	return callbackFunctions[i]
}

// unregisterCallback unregisters a callback by removing it from the
// callbackFunctions map.
func unregisterCallback(i uintptr) {
	callbackMutex.Lock()
	defer callbackMutex.Unlock()
	delete(callbackFunctions, i)
}
