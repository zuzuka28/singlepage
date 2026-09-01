package httpapi

import (
	"encoding/json"
	"net/http"

	"singlepage/internal/handler/httpapi/gen"
)

func writeError(w http.ResponseWriter, status int, message string) {
	payload, err := json.Marshal(gen.ErrorResponse{Error: message})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	payload = append(payload, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(payload)
	if err != nil {
		return
	}
}

func recordRequestError(w http.ResponseWriter, err error) {
	for {
		recorder, ok := w.(interface{ RecordError(err error) })
		if ok {
			recorder.RecordError(err)

			return
		}

		unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return
		}

		w = unwrapper.Unwrap()
	}
}
