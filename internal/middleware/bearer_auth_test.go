package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"singlepage/internal/middleware"
)

func TestBearerAuthReturnsNotFoundWhenFeatureIsDisabled(t *testing.T) {
	t.Parallel()

	response := serveBearerAuthRequest(t, "", "", nil)

	assertJSONResponse(t, response, http.StatusNotFound, "{\"error\":\"Not Found\"}\n")
}

func TestBearerAuthRejectsMissingToken(t *testing.T) {
	t.Parallel()

	response := serveBearerAuthRequest(t, "secret", "", nil)

	assertJSONResponse(t, response, http.StatusUnauthorized, "{\"error\":\"Unauthorized\"}\n")
}

func TestBearerAuthRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	response := serveBearerAuthRequest(t, "secret", "wrong", nil)

	assertJSONResponse(t, response, http.StatusForbidden, "{\"error\":\"Forbidden\"}\n")
}

func TestBearerAuthAllowsMatchingToken(t *testing.T) {
	t.Parallel()

	called := false
	response := serveBearerAuthRequest(t, "secret", "secret", func() { called = true })

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}

	if !called {
		t.Fatal("next handler was not called")
	}
}

func serveBearerAuthRequest(
	t *testing.T,
	wantToken string,
	providedToken string,
	onCall func(),
) *httptest.ResponseRecorder {
	t.Helper()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if onCall != nil {
			onCall()
		}

		w.WriteHeader(http.StatusNoContent)
	})
	handler := middleware.BearerAuth(wantToken)(next)

	request := newRequest(http.MethodDelete, "/api/admin/pages/id", nil)

	if providedToken != "" {
		request.Header.Set("Authorization", "Bearer "+providedToken)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	return response
}
