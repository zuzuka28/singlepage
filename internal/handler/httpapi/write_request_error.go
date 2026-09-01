package httpapi

import "net/http"

func writeRequestError(w http.ResponseWriter, _ *http.Request, err error) {
	recordRequestError(w, err)
	writeError(w, http.StatusBadRequest, "invalid JSON body")
}
