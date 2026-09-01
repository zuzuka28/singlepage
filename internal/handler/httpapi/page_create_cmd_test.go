package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	modelpage "singlepage/internal/model/page"
)

var errServiceFailed = errors.New("service failed")

func TestPageCreateCmdReturnsCreatedRevision(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()

	var captured modelpage.CreateServiceCmd

	pages.create = func(
		_ context.Context,
		cmd modelpage.CreateServiceCmd,
	) (modelpage.MutationResponse, error) {
		captured = cmd

		return modelpage.MutationResponse{Revision: 1}, nil
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPost, "/api/pages", `{
		"id":"0123456789abcdef",
		"salt":"c2FsdA==",
		"ciphertext":"Y2lwaGVy",
		"writeToken":"secret"
	}`, "")

	assertAPIResponse(t, response, http.StatusCreated, "{\"revision\":1}\n")

	if captured.ID != testPageID || string(captured.Salt) != "salt" ||
		string(captured.Ciphertext) != "cipher" || captured.WriteToken != "secret" {
		t.Fatalf("create command = %+v", captured)
	}
}

func TestPageCreateCmdMapsDuplicateIDToConflict(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	pages.create = func(
		context.Context,
		modelpage.CreateServiceCmd,
	) (modelpage.MutationResponse, error) {
		return modelpage.MutationResponse{}, modelpage.ErrConflict
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPost, "/api/pages", `{
		"id":"0123456789abcdef",
		"salt":"c2FsdA==",
		"ciphertext":"Y2lwaGVy",
		"writeToken":"secret"
	}`, "")

	assertAPIResponse(t, response, http.StatusConflict, "{\"error\":\"Conflict\"}\n")
}

func TestPageCreateCmdMapsUnknownFailureToInternalError(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	pages.create = func(
		context.Context,
		modelpage.CreateServiceCmd,
	) (modelpage.MutationResponse, error) {
		return modelpage.MutationResponse{}, errServiceFailed
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPost, "/api/pages", `{
		"id":"0123456789abcdef",
		"salt":"c2FsdA==",
		"ciphertext":"Y2lwaGVy",
		"writeToken":"secret"
	}`, "")

	assertAPIResponse(t, response, http.StatusInternalServerError, "{\"error\":\"internal server error\"}\n")
}
