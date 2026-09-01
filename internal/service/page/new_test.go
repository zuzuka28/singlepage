package page_test

import (
	"testing"

	servicepage "singlepage/internal/service/page"
)

func TestNew(t *testing.T) {
	t.Parallel()

	service := servicepage.New(&fakeRepository{}, servicepage.Config{})
	if service == nil {
		t.Fatal("New returned nil")
	}
}
