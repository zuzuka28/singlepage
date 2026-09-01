package page_test

import (
	"errors"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestDelete(t *testing.T) {
	t.Parallel()

	repository := openTestRepository(t)
	createTestPage(t, repository)

	err := repository.Delete(t.Context(), testID)
	if err != nil {
		t.Fatal(err)
	}

	err = repository.Delete(t.Context(), testID)
	if !errors.Is(err, modelpage.ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want %v", err, modelpage.ErrNotFound)
	}
}
