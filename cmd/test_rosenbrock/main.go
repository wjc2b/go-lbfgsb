package main

import (
	"fmt"

	lbfgsb "github.com/Terminally-Online/go-lbfgsb"
)

type Rosenbrock struct{}

func (r Rosenbrock) EvaluateFunction(x []float64) float64 {
	return (1-x[0])*(1-x[0]) + 100*(x[1]-x[0]*x[0])*(x[1]-x[0]*x[0])
}

func (r Rosenbrock) EvaluateGradient(x []float64) []float64 {
	return []float64{
		-2*(1-x[0]) - 400*x[0]*(x[1]-x[0]*x[0]),
		200 * (x[1] - x[0]*x[0]),
	}
}

func main() {
	opt := lbfgsb.NewLbfgsb(2).
		SetFTolerance(1e-10).
		SetGTolerance(1e-10)

	result, status := opt.Minimize(Rosenbrock{}, []float64{-1.2, 1.0})

	fmt.Printf("Status: %v\n", status.Code)
	fmt.Printf("Minimum at: (%.6f, %.6f)\n", result.X[0], result.X[1])
	fmt.Printf("Value: %.6e\n", result.F)
}
