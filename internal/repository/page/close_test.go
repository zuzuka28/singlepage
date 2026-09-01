package page_test

import (
	"testing"

	repositorypage "singlepage/internal/repository/page"
)

func TestClose(t *testing.T) {
	t.Parallel()

	repository, err := repositorypage.Open(t.Context(), testDatabasePath(t), testDBLimit)
	if err != nil {
		t.Fatal(err)
	}

	err = repository.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
