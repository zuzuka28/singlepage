package frontend_test

import (
	"testing"

	"singlepage/internal/handler/frontend"
)

func TestNewReturnsEmbeddedFrontendHandler(t *testing.T) {
	t.Parallel()

	if frontend.New() == nil {
		t.Fatal("New() returned nil")
	}
}
