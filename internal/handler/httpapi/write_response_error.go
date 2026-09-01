package httpapi

import "net/http"

func writeResponseError(w http.ResponseWriter, _ *http.Request, err error) {
	recordRequestError(w, err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}
