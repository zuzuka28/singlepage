package metrics

import (
	"fmt"
	"net/http"
)

type responseWriter struct {
	http.ResponseWriter

	status      int
	wroteHeader bool
	route       string
}

func (writer *responseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}

	writer.status = status
	writer.wroteHeader = true
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *responseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}

	written, err := writer.ResponseWriter.Write(body)
	if err != nil {
		return written, fmt.Errorf("write response: %w", err)
	}

	return written, nil
}

func (writer *responseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

func (writer *responseWriter) RecordRoute(pattern string) {
	writer.route = pattern
}
