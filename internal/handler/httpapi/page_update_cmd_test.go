package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestPageUpdateCmdUsesBearerTokenAndReturnsRevision(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()

	var captured modelpage.UpdateServiceCmd

	pages.update = func(
		_ context.Context,
		cmd modelpage.UpdateServiceCmd,
	) (modelpage.MutationResponse, error) {
		captured = cmd

		return modelpage.MutationResponse{Revision: 2}, nil
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPut, "/api/pages/"+testPageID, `{
		"expectedRevision":1,
		"ciphertext":"bmV3IGNpcGhlcg==",
		"salt":"bmV3IHNhbHQ=",
		"newWriteToken":"new-secret",
		"writeToken":"legacy-secret"
	}`, "bearer-secret")

	assertAPIResponse(t, response, http.StatusOK, "{\"revision\":2}\n")

	if captured.ID != testPageID || captured.ExpectedRevision != 1 ||
		string(captured.Ciphertext) != "new cipher" || captured.Salt == nil ||
		string(*captured.Salt) != "new salt" || captured.WriteToken != "bearer-secret" ||
		captured.NewWriteToken != "new-secret" {
		t.Fatalf("update command = %+v", captured)
	}
}

func TestPageUpdateCmdUsesLegacyBodyTokenWhenBearerIsMissing(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()

	var capturedToken string

	pages.update = func(
		_ context.Context,
		cmd modelpage.UpdateServiceCmd,
	) (modelpage.MutationResponse, error) {
		capturedToken = cmd.WriteToken

		return modelpage.MutationResponse{Revision: 2}, nil
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPut, "/api/pages/"+testPageID, `{
		"expectedRevision":1,
		"ciphertext":"bmV3IGNpcGhlcg==",
		"writeToken":"legacy-secret"
	}`, "")

	assertAPIResponse(t, response, http.StatusOK, "{\"revision\":2}\n")

	if capturedToken != "legacy-secret" {
		t.Fatalf("write token = %q, want legacy-secret", capturedToken)
	}
}

func TestPageUpdateCmdMapsMissingCapabilityToUnauthorized(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	pages.update = func(
		context.Context,
		modelpage.UpdateServiceCmd,
	) (modelpage.MutationResponse, error) {
		return modelpage.MutationResponse{}, modelpage.ErrUnauthorized
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPut, "/api/pages/"+testPageID, `{
		"expectedRevision":1,
		"ciphertext":"bmV3IGNpcGhlcg=="
	}`, "")

	assertAPIResponse(t, response, http.StatusUnauthorized, "{\"error\":\"Unauthorized\"}\n")
}

func TestPageUpdateCmdMapsStaleRevisionToConflict(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	pages.update = func(
		context.Context,
		modelpage.UpdateServiceCmd,
	) (modelpage.MutationResponse, error) {
		return modelpage.MutationResponse{}, modelpage.ErrConflict
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodPut, "/api/pages/"+testPageID, `{
		"expectedRevision":1,
		"ciphertext":"bmV3IGNpcGhlcg=="
	}`, "secret")

	assertAPIResponse(t, response, http.StatusConflict, "{\"error\":\"Conflict\"}\n")
}
