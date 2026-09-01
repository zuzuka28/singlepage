package page_test

import (
	"testing"

	servicepage "singlepage/internal/service/page"
)

func TestNewWithClock(t *testing.T) {
	t.Parallel()

	service := servicepage.NewWithClock(&fakeRepository{}, servicepage.Config{}, nil)
	if service == nil {
		t.Fatal("NewWithClock returned nil")
	}
}
