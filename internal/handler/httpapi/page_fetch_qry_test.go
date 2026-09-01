package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	modelpage "singlepage/internal/model/page"
)

func TestPageFetchQryReturnsOpaquePage(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	pages.get = func(
		_ context.Context,
		qry modelpage.GetServiceQry,
	) (modelpage.Page, error) {
		if qry.ID != testPageID {
			t.Fatalf("page ID = %q, want %q", qry.ID, testPageID)
		}

		return modelpage.Page{
			ID:         testPageID,
			Revision:   2,
			Salt:       []byte("salt"),
			Ciphertext: []byte("cipher"),
			UpdatedAt:  time.Time{},
		}, nil
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodGet, "/api/pages/"+testPageID, "", "")

	assertAPIResponse(
		t,
		response,
		http.StatusOK,
		"{\"ciphertext\":\"Y2lwaGVy\",\"revision\":2,\"salt\":\"c2FsdA==\"}\n",
	)
}

func TestPageFetchQryRejectsInvalidID(t *testing.T) {
	t.Parallel()

	pages := newFakePageService()
	called := false
	pages.get = func(
		context.Context,
		modelpage.GetServiceQry,
	) (modelpage.Page, error) {
		called = true

		return modelpage.Page{}, nil
	}
	handler := newAPIHandler(t, pages)

	response := apiRequest(t, handler, http.MethodGet, "/api/pages/too-short", "", "")

	assertAPIResponse(t, response, http.StatusBadRequest, "{\"error\":\"Bad Request\"}\n")

	if called {
		t.Fatal("service was called for an invalid ID")
	}
}
