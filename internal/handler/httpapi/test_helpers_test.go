package httpapi_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"singlepage/internal/config"
	"singlepage/internal/handler/httpapi"
	"singlepage/internal/metrics"
	modelpage "singlepage/internal/model/page"
)

const (
	testPageID    = "0123456789abcdef"
	rotatedPageID = "fedcba9876543210"
)

type fakePageService struct {
	create func(context.Context, modelpage.CreateServiceCmd) (modelpage.MutationResponse, error)
	get    func(context.Context, modelpage.GetServiceQry) (modelpage.Page, error)
	update func(context.Context, modelpage.UpdateServiceCmd) (modelpage.MutationResponse, error)
	rotate func(context.Context, modelpage.RotateServiceCmd) (modelpage.MutationResponse, error)
	delete func(context.Context, modelpage.DeleteServiceCmd) error
}

func (f *fakePageService) Create(
	ctx context.Context,
	cmd modelpage.CreateServiceCmd,
) (modelpage.MutationResponse, error) {
	return f.create(ctx, cmd)
}

func (f *fakePageService) Get(
	ctx context.Context,
	qry modelpage.GetServiceQry,
) (modelpage.Page, error) {
	return f.get(ctx, qry)
}

func (f *fakePageService) Update(
	ctx context.Context,
	cmd modelpage.UpdateServiceCmd,
) (modelpage.MutationResponse, error) {
	return f.update(ctx, cmd)
}

func (f *fakePageService) Rotate(
	ctx context.Context,
	cmd modelpage.RotateServiceCmd,
) (modelpage.MutationResponse, error) {
	return f.rotate(ctx, cmd)
}

func (f *fakePageService) Delete(ctx context.Context, cmd modelpage.DeleteServiceCmd) error {
	return f.delete(ctx, cmd)
}

func newFakePageService() *fakePageService {
	return &fakePageService{
		create: func(context.Context, modelpage.CreateServiceCmd) (modelpage.MutationResponse, error) {
			return modelpage.MutationResponse{}, nil
		},
		get: func(context.Context, modelpage.GetServiceQry) (modelpage.Page, error) {
			return modelpage.Page{}, nil
		},
		update: func(context.Context, modelpage.UpdateServiceCmd) (modelpage.MutationResponse, error) {
			return modelpage.MutationResponse{}, nil
		},
		rotate: func(context.Context, modelpage.RotateServiceCmd) (modelpage.MutationResponse, error) {
			return modelpage.MutationResponse{}, nil
		},
		delete: func(context.Context, modelpage.DeleteServiceCmd) error { return nil },
	}
}

func newAPIHandler(t *testing.T, pages *fakePageService) http.Handler {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)

	return httpapi.New(
		pages,
		config.Config{Protection: config.Protection{
			MaxRequestBodyBytes: 1 << 20,
			AdminToken:          "admin",
		}},
		metrics.New(),
		logger,
	)
}

func apiRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	body string,
	token string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(
		context.Background(),
		method,
		path,
		bytes.NewBufferString(body),
	)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	return response
}

func assertAPIResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantBody string,
) {
	t.Helper()

	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, wantStatus, response.Body.String())
	}

	if response.Body.String() != wantBody {
		t.Fatalf("body = %q, want %q", response.Body.String(), wantBody)
	}

	if wantBody == "" {
		return
	}

	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
}
