package lbfgsb

import (
	"fmt"
	"math"
	"testing"
)

// Rosenbrock function: f(x,y) = (1-x)^2 + 100*(y-x^2)^2
// Global minimum at (1,1), f=0
type rosenbrock struct{}

func (r rosenbrock) EvaluateFunction(p []float64) float64 {
	x, y := p[0], p[1]
	return math.Pow(1-x, 2) + 100*math.Pow(y-x*x, 2)
}

func (r rosenbrock) EvaluateGradient(p []float64) []float64 {
	x, y := p[0], p[1]
	return []float64{
		-2*(1-x) - 400*x*(y-x*x),
		200 * (y - x*x),
	}
}

func TestMinimize(t *testing.T) {
	solver := NewLbfgsb(2)
	solver.SetFTolerance(1e-8)
	solver.SetGTolerance(1e-8)

	// Start far from the minimum to stress the optimizer
	minimum, status := solver.Minimize(rosenbrock{}, []float64{-1.2, 1.0})

	if status.Code != SUCCESS {
		t.Fatalf("optimization failed: %v", status)
	}

	fmt.Printf("Minimum: f(%.6f, %.6f) = %.10f\n", minimum.X[0], minimum.X[1], minimum.F)

	if math.Abs(minimum.X[0]-1.0) > 1e-4 || math.Abs(minimum.X[1]-1.0) > 1e-4 {
		t.Errorf("expected (1,1), got (%f,%f)", minimum.X[0], minimum.X[1])
	}
	if math.Abs(minimum.F) > 1e-6 {
		t.Errorf("expected f=0, got f=%f", minimum.F)
	}
}
