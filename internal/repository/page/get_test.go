package page_test

import (
	"errors"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestGet(t *testing.T) {
	t.Parallel()

	repository := openTestRepository(t)
	createTestPage(t, repository)

	page, err := repository.Get(t.Context(), testID)
	if err != nil {
		t.Fatal(err)
	}

	if page.ID != testID || page.Revision != 1 || !page.UpdatedAt.Equal(testTime()) {
		t.Fatalf("Get() = %+v", page)
	}

	if string(page.WriteTokenHash) != testHash {
		t.Fatalf("Get().WriteTokenHash = %q", page.WriteTokenHash)
	}
}

func TestGetReturnsNotFound(t *testing.T) {
	t.Parallel()

	repository := openTestRepository(t)

	_, err := repository.Get(t.Context(), testID)
	if !errors.Is(err, modelpage.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, modelpage.ErrNotFound)
	}
}
