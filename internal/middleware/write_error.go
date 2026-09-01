package middleware

import (
	"encoding/json"
	"net/http"
)

func writeError(w http.ResponseWriter, status int, message string) {
	payload, err := json.Marshal(map[string]string{"error": message})
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
