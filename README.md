# Go L-BFGS-B

A Go package for L-BFGS-B optimization with **zero dependencies**.

This is a fork of [Terminally-Online/go-lbfgsb](https://github.com/Terminally-Online/go-lbfgsb), which itself was forked from [idavydov/go-lbfgsb](https://github.com/idavydov/go-lbfgsb) → [afbarnard/go-lbfgsb](https://github.com/afbarnard/go-lbfgsb). Comes with precompiled Fortran binaries — no GCC, no gfortran, no CGO_LDFLAGS setup required.

## Installation

```bash
go get github.com/Terminally-Online/go-lbfgsb
```

That's it. Works on macOS (arm64/amd64), Linux (amd64/arm64), and Windows (amd64).

## Usage

```go
package main

import (
    "fmt"
    lbfgsb "github.com/Terminally-Online/go-lbfgsb"
)

// Implement the FunctionWithGradient interface
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
```

## What is L-BFGS-B?

L-BFGS-B is a limited-memory quasi-Newton optimization algorithm for bound-constrained problems. It's efficient for large-scale optimization where you can compute gradients but the Hessian is too expensive to store.

## Testing

```bash
# macOS / Linux — just Go
go test -v .

# Windows — needs MinGW-w64 gcc for cgo
# 1. Install MinGW-w64: https://www.mingw-w64.org/
# 2. Make sure gcc is in PATH
gcc --version
# 3. Run tests
go test -v .
```

The test minimizes the Rosenbrock function using the L-BFGS-B solver. If it passes, the precompiled `.syso` links correctly on your platform.

## Building from Source

If you need to rebuild the precompiled binaries (maintainers only):

```bash
# Requires gfortran
brew install gcc  # macOS
# or
sudo apt-get install gfortran  # Linux

make
```

## License

BSD 2-Clause License. See [LICENSE.txt](LICENSE.txt).
