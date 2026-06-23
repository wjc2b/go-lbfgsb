# Go L-BFGS-B

A Go package for L-BFGS-B optimization. Fortran code is precompiled — **no gfortran needed**.

This is a fork of [Terminally-Online/go-lbfgsb](https://github.com/Terminally-Online/go-lbfgsb), which itself was forked from [idavydov/go-lbfgsb](https://github.com/idavydov/go-lbfgsb) → [afbarnard/go-lbfgsb](https://github.com/afbarnard/go-lbfgsb).

## Platform Support

| Platform | Approach | Prerequisites |
|----------|----------|---------------|
| macOS (arm64/amd64) | cgo + precompiled `.syso` | clang (built-in) |
| Linux (amd64/arm64) | cgo + precompiled `.syso` | gcc (usually built-in) |
| **Windows (amd64)** | pure Go + precompiled `.dll` | **none — zero dependency** |

On Windows the package uses `syscall.LoadDLL` instead of cgo, so no C compiler is needed at all. Just Go.

## Installation

```bash
go get github.com/Terminally-Online/go-lbfgsb
```

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
# macOS / Linux
go test -v .

# Windows — no setup needed
go test -v .
```

The test minimizes the Rosenbrock function. If it passes, the precompiled binaries link correctly on your platform.

## Building from Source

If you need to rebuild the precompiled binaries (maintainers only):

### macOS / Linux native (.syso)

```bash
# Prerequisites
brew install gcc           # macOS
sudo apt install gfortran  # Linux

make
```

### Windows DLL

```bash
# Prerequisites
brew install mingw-w64              # macOS
sudo apt install gfortran-mingw-w64-x86-64  # Linux

make build-windows-dll
# → produces lbfgsb.dll
```

### Windows (.syso, deprecated)

The old cgo-based approach for Windows is still available but requires MinGW-w64 at build time and on the target machine:

```bash
make build-windows-amd64
# → produces lbfgsb_windows_amd64.syso
```

## License

BSD 2-Clause License. See [LICENSE.txt](LICENSE.txt).
