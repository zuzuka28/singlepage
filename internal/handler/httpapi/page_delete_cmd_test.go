package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestPageDeleteCmdReturnsNoContent(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()

	var captured modelpage.DeleteServiceCmd

	pages.delete = func(_ context.Context, cmd modelpage.DeleteServiceCmd) error {
		captured = cmd

		return nil
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodDelete, "/api/admin/pages/"+testPageID, "", "admin")

	assertAPIResponse(t, response, http.StatusNoContent, "")

	if captured.ID != testPageID {
		t.Fatalf("delete ID = %q, want %q", captured.ID, testPageID)
	}
}

func TestPageDeleteCmdMapsMissingPageToNotFound(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	pages.delete = func(context.Context, modelpage.DeleteServiceCmd) error {
		return modelpage.ErrNotFound
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodDelete, "/api/admin/pages/"+testPageID, "", "admin")

	assertAPIResponse(t, response, http.StatusNotFound, "{\"error\":\"Not Found\"}\n")
}

func TestPageDeleteCmdRejectsInvalidID(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	called := false
	pages.delete = func(context.Context, modelpage.DeleteServiceCmd) error {
		called = true

		return nil
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodDelete, "/api/admin/pages/too-short", "", "admin")

	assertAPIResponse(t, response, http.StatusBadRequest, "{\"error\":\"Bad Request\"}\n")

	if called {
		t.Fatal("service was called for an invalid ID")
	}
}
