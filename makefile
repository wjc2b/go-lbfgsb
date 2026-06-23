.PHONY: all clean build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-linux-arm64 build-windows-amd64 build-windows-dll

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

all:
ifeq ($(UNAME_S),Darwin)
ifeq ($(UNAME_M),arm64)
	$(MAKE) build-darwin-arm64
else
	$(MAKE) build-darwin-amd64
endif
else
ifeq ($(UNAME_M),aarch64)
	$(MAKE) build-linux-arm64
else
	$(MAKE) build-linux-amd64
endif
endif

build-darwin-arm64:
	@echo "Building for darwin/arm64..."
	cd src && gfortran -c -O2 -fPIC lbfgsb.f blas.f linpack.f timer.f
	cd src && gfortran -c -O2 -fPIC lbfgsb__entry.f90
	cd src && gfortran -c -O2 -fPIC lbfgsb_c.f90
	ld -r -o lbfgsb_darwin_arm64.syso \
		src/lbfgsb.o src/blas.o src/linpack.o src/timer.o \
		src/lbfgsb__entry.o src/lbfgsb_c.o \
		$$(brew --prefix gcc)/lib/gcc/current/libgfortran.a \
		$$(brew --prefix gcc)/lib/gcc/current/libquadmath.a \
		$$(brew --prefix gcc)/lib/gcc/current/gcc/aarch64-apple-darwin*/*/libgcc.a
	rm -f src/*.o src/*.mod
	@echo "Built lbfgsb_darwin_arm64.syso"

build-darwin-amd64:
	@echo "Building for darwin/amd64..."
	cd src && gfortran -c -O2 -fPIC lbfgsb.f blas.f linpack.f timer.f
	cd src && gfortran -c -O2 -fPIC lbfgsb__entry.f90
	cd src && gfortran -c -O2 -fPIC lbfgsb_c.f90
	ld -r -o lbfgsb_darwin_amd64.syso \
		src/lbfgsb.o src/blas.o src/linpack.o src/timer.o \
		src/lbfgsb__entry.o src/lbfgsb_c.o \
		$$(brew --prefix gcc)/lib/gcc/current/libgfortran.a \
		$$(brew --prefix gcc)/lib/gcc/current/libquadmath.a \
		$$(brew --prefix gcc)/lib/gcc/current/gcc/x86_64-apple-darwin*/*/libgcc.a
	rm -f src/*.o src/*.mod
	@echo "Built lbfgsb_darwin_amd64.syso"

build-linux-amd64:
	@echo "Building for linux/amd64..."
	cd src && gfortran -c -O2 -fPIC lbfgsb.f blas.f linpack.f timer.f
	cd src && gfortran -c -O2 -fPIC lbfgsb__entry.f90
	cd src && gfortran -c -O2 -fPIC lbfgsb_c.f90
	ld -r -o lbfgsb_linux_amd64.syso \
		src/lbfgsb.o src/blas.o src/linpack.o src/timer.o \
		src/lbfgsb__entry.o src/lbfgsb_c.o \
		$$(find /usr/lib/gcc -name 'libgfortran.a' | head -1) \
		$$(find /usr/lib/gcc -name 'libquadmath.a' | head -1) \
		$$(find /usr/lib/gcc -name 'libgcc.a' | head -1)
	rm -f src/*.o src/*.mod
	@echo "Built lbfgsb_linux_amd64.syso"

# NOTE: no libquadmath on linux/arm64 — __float128 is an x86-only GCC
# extension and Ubuntu doesn't ship libquadmath for aarch64. lbfgsb
# itself is double-precision, so dropping the lib is correct.
build-linux-arm64:
	@echo "Building for linux/arm64..."
	cd src && gfortran -c -O2 -fPIC lbfgsb.f blas.f linpack.f timer.f
	cd src && gfortran -c -O2 -fPIC lbfgsb__entry.f90
	cd src && gfortran -c -O2 -fPIC lbfgsb_c.f90
	ld -r -o lbfgsb_linux_arm64.syso \
		src/lbfgsb.o src/blas.o src/linpack.o src/timer.o \
		src/lbfgsb__entry.o src/lbfgsb_c.o \
		$$(find /usr/lib/gcc -name 'libgfortran.a' | head -1) \
		$$(find /usr/lib/gcc -name 'libgcc.a' | head -1)
	rm -f src/*.o src/*.mod
	@echo "Built lbfgsb_linux_arm64.syso"

# Cross-compile for Windows x64 using MinGW-w64.
# Prerequisites:
#   macOS:  brew install mingw-w64
#   Linux:  apt install gfortran-mingw-w64-x86-64
build-windows-amd64:
	@echo "Building for windows/amd64..."
	cd src && x86_64-w64-mingw32-gfortran -c -O2 -fPIC lbfgsb.f blas.f linpack.f timer.f
	cd src && x86_64-w64-mingw32-gfortran -c -O2 -fPIC lbfgsb__entry.f90
	cd src && x86_64-w64-mingw32-gfortran -c -O2 -fPIC lbfgsb_c.f90
ifeq ($(UNAME_S),Darwin)
	x86_64-w64-mingw32-ld -r -o lbfgsb_windows_amd64.syso \
		src/lbfgsb.o src/blas.o src/linpack.o src/timer.o \
		src/lbfgsb__entry.o src/lbfgsb_c.o \
		$$(brew --prefix mingw-w64)/toolchain-x86_64/x86_64-w64-mingw32/lib/libgfortran.a \
		$$(brew --prefix mingw-w64)/toolchain-x86_64/x86_64-w64-mingw32/lib/libquadmath.a \
		$$(brew --prefix mingw-w64)/toolchain-x86_64/lib/gcc/x86_64-w64-mingw32/*/libgcc.a
else
	x86_64-w64-mingw32-ld -r -o lbfgsb_windows_amd64.syso \
		src/lbfgsb.o src/blas.o src/linpack.o src/timer.o \
		src/lbfgsb__entry.o src/lbfgsb_c.o \
		$$(find /usr/lib/gcc/x86_64-w64-mingw32 -name 'libgfortran.a' | head -1) \
		$$(find /usr/lib/gcc/x86_64-w64-mingw32 -name 'libquadmath.a' | head -1) \
		$$(find /usr/lib/gcc/x86_64-w64-mingw32 -name 'libgcc.a' | head -1)
endif
	rm -f src/*.o src/*.mod
	@echo "Built lbfgsb_windows_amd64.syso"

# Build Windows DLL (for zero-dependency windows/amd64).
# Contains Fortran + C glue + static libgfortran/libquadmath/libgcc.
# Required by lbfgsb_windows.go (pure Go, no cgo).
# Prerequisites:
#   macOS:  brew install mingw-w64
#   Linux:  apt install gfortran-mingw-w64-x86-64
build-windows-dll:
	@echo "Building lbfgsb.dll for windows/amd64..."
	cd src && x86_64-w64-mingw32-gfortran -c -O2 -fPIC lbfgsb.f blas.f linpack.f timer.f
	cd src && x86_64-w64-mingw32-gfortran -c -O2 -fPIC lbfgsb__entry.f90
	cd src && x86_64-w64-mingw32-gfortran -c -O2 -fPIC lbfgsb_c.f90
	x86_64-w64-mingw32-gcc -c -O2 -fPIC -I. -o dll/lbfgsb_windows_interface.o dll/lbfgsb_windows_interface.c
ifeq ($(UNAME_S),Darwin)
	x86_64-w64-mingw32-gcc -shared -o lbfgsb.dll \
		src/lbfgsb.o src/blas.o src/linpack.o src/timer.o \
		src/lbfgsb__entry.o src/lbfgsb_c.o \
		dll/lbfgsb_windows_interface.o \
		$$(brew --prefix mingw-w64)/toolchain-x86_64/x86_64-w64-mingw32/lib/libgfortran.a \
		$$(brew --prefix mingw-w64)/toolchain-x86_64/x86_64-w64-mingw32/lib/libquadmath.a \
		$$(brew --prefix mingw-w64)/toolchain-x86_64/lib/gcc/x86_64-w64-mingw32/*/libgcc.a \
		-Wl,--export-all-symbols
else
	x86_64-w64-mingw32-gcc -shared -o lbfgsb.dll \
		src/lbfgsb.o src/blas.o src/linpack.o src/timer.o \
		src/lbfgsb__entry.o src/lbfgsb_c.o \
		dll/lbfgsb_windows_interface.o \
		$$(find /usr/lib/gcc/x86_64-w64-mingw32 -name 'libgfortran.a' | head -1) \
		$$(find /usr/lib/gcc/x86_64-w64-mingw32 -name 'libquadmath.a' | head -1) \
		$$(find /usr/lib/gcc/x86_64-w64-mingw32 -name 'libgcc.a' | head -1) \
		-Wl,--export-all-symbols
endif
	rm -f src/*.o src/*.mod dll/*.o
	@echo "Built lbfgsb.dll"

clean:
	rm -f src/*.o src/*.mod dll/*.o *.syso *.dll
