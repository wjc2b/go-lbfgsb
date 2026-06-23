// C wrapper compiled into lbfgsb.dll.  Accepts callback function pointers
// as normal parameters (no cgo dependency).  See the header for details.

#include <stddef.h>
#include "lbfgsb_windows_interface.h"

int lbfgsb_minimize_windows(struct lbfgsb_call *call) {
    // Only pass the logging function if requested
    lbfgsb_log_function_type log_fn = NULL;
    if (call->do_logging) {
        log_fn = call->log_fn;
    }

    return lbfgsb_minimize(
        call->obj_fn,
        call->grad_fn,
        call->callback_data,
        call->dim,
        call->bounds_control,
        call->lower_bounds,
        call->upper_bounds,
        call->approximation_size,
        call->f_tolerance,
        call->g_tolerance,
        call->initial_point,
        call->min_x,
        call->min_f,
        call->min_g,
        call->iters,
        call->evals,
        call->fortran_print_control,
        log_fn,
        call->log_callback_data,
        call->status_message,
        call->status_message_length);
}
