package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestPageRotateCmdUsesBearerTokenAndReturnsCreatedRevision(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()

	var captured modelpage.RotateServiceCmd

	pages.rotate = func(
		_ context.Context,
		cmd modelpage.RotateServiceCmd,
	) (modelpage.MutationResponse, error) {
		captured = cmd

		return modelpage.MutationResponse{Revision: 1}, nil
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPost, "/api/pages/"+testPageID+"/rotate", `{
		"newId":"fedcba9876543210",
		"salt":"bmV3IHNhbHQ=",
		"ciphertext":"bmV3IGNpcGhlcg==",
		"newWriteToken":"new-secret"
	}`, "old-secret")

	assertAPIResponse(t, response, http.StatusCreated, "{\"revision\":1}\n")

	if captured.OldID != testPageID || captured.NewID != rotatedPageID ||
		string(captured.Salt) != "new salt" || string(captured.Ciphertext) != "new cipher" ||
		captured.WriteToken != "old-secret" || captured.NewWriteToken != "new-secret" {
		t.Fatalf("rotate command = %+v", captured)
	}
}

func TestPageRotateCmdMapsConcurrentChangeToConflict(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	pages.rotate = func(
		context.Context,
		modelpage.RotateServiceCmd,
	) (modelpage.MutationResponse, error) {
		return modelpage.MutationResponse{}, modelpage.ErrConcurrentChange
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPost, "/api/pages/"+testPageID+"/rotate", `{
		"newId":"fedcba9876543210",
		"salt":"bmV3IHNhbHQ=",
		"ciphertext":"bmV3IGNpcGhlcg==",
		"newWriteToken":"new-secret"
	}`, "old-secret")

	assertAPIResponse(t, response, http.StatusConflict, "{\"error\":\"Conflict\"}\n")
}

func TestPageRotateCmdMapsDuplicateNewIDToConflict(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	pages.rotate = func(
		context.Context,
		modelpage.RotateServiceCmd,
	) (modelpage.MutationResponse, error) {
		return modelpage.MutationResponse{}, modelpage.ErrConflict
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPost, "/api/pages/"+testPageID+"/rotate", `{
		"newId":"fedcba9876543210",
		"salt":"bmV3IHNhbHQ=",
		"ciphertext":"bmV3IGNpcGhlcg==",
		"newWriteToken":"new-secret"
	}`, "old-secret")

	assertAPIResponse(t, response, http.StatusConflict, "{\"error\":\"Conflict\"}\n")
}
