package httpapi

import (
	"context"
	"net/http"

	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

func writeValidationError(
	_ context.Context,
	err error,
	w http.ResponseWriter,
	_ *http.Request,
	opts nethttpmiddleware.ErrorHandlerOpts,
) {
	recordRequestError(w, err)
	writeError(w, opts.StatusCode, http.StatusText(opts.StatusCode))
}
