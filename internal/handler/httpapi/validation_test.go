package httpapi_test

import (
	"net/http"
	"testing"
)

func TestOpenAPIValidationRejectsUnknownJSONField(t *testing.T) {
	t.Parallel()

	handler := newAPIHandler(t, newFakePageService())
	response := apiRequest(t, handler, http.MethodPost, "/api/pages", `{
		"id":"0123456789abcdef",
		"salt":"c2FsdA==",
		"ciphertext":"Y2lwaGVy",
		"writeToken":"secret",
		"unknown":true
	}`, "")

	assertAPIResponse(t, response, http.StatusBadRequest, "{\"error\":\"Bad Request\"}\n")
}

func TestGeneratedRouterRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	handler := newAPIHandler(t, newFakePageService())
	response := apiRequest(t, handler, http.MethodPost, "/api/pages", "{", "")

	assertAPIResponse(t, response, http.StatusBadRequest, "{\"error\":\"Bad Request\"}\n")
}

func TestOpenAPIValidationRejectsMissingRequestBody(t *testing.T) {
	t.Parallel()

	handler := newAPIHandler(t, newFakePageService())
	response := apiRequest(t, handler, http.MethodPost, "/api/pages", "", "")

	assertAPIResponse(t, response, http.StatusBadRequest, "{\"error\":\"Bad Request\"}\n")
}
