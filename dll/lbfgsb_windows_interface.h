// C wrapper for the L-BFGS-B Fortran library, compiled into lbfgsb.dll.
// Unlike lbfgsb_go_interface.c (which pulls callbacks from cgo's
// _cgo_export.h), this version takes callback function pointers as
// normal parameters — the Go side provides them via syscall.NewCallback.

#ifndef __LBFGSB_WINDOWS_INTERFACE_H__
#define __LBFGSB_WINDOWS_INTERFACE_H__

#include "lbfgsb_c.h"

// All parameters packed into a single struct so Go can pass one pointer
// via syscall — avoids float-calling-convention issues.
struct lbfgsb_call {
    lbfgsb_objective_function_type obj_fn;
    lbfgsb_objective_gradient_type grad_fn;
    void *callback_data;
    int dim;
    int *bounds_control;
    double *lower_bounds;
    double *upper_bounds;
    int approximation_size;
    double f_tolerance;
    double g_tolerance;
    double *initial_point;
    double *min_x;
    double *min_f;
    double *min_g;
    int *iters;
    int *evals;
    int fortran_print_control;
    int do_logging;
    lbfgsb_log_function_type log_fn;
    void *log_callback_data;
    char *status_message;
    int status_message_length;
};

// Single entry point — unpacks the struct and calls into Fortran.
// Exported from lbfgsb.dll for Go to call via syscall.
int lbfgsb_minimize_windows(struct lbfgsb_call *call);

#endif
