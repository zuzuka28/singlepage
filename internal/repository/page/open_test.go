package page_test

import (
	"errors"
	"fmt"
	"testing"

	modelpage "singlepage/internal/model/page"
	repositorypage "singlepage/internal/repository/page"
)

func TestOpenAppliesDatabaseSizeLimit(t *testing.T) {
	t.Parallel()

	const databaseLimit = 1 << 20

	repository, err := repositorypage.Open(t.Context(), testDatabasePath(t), databaseLimit)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { closeTestRepository(t, repository) })

	payload := make([]byte, databaseLimit/2)
	for index := range 10 {
		err = repository.Create(t.Context(), modelpage.CreateRepositoryCmd{
			ID:             fmt.Sprintf("page-identifier-%02d", index),
			Salt:           []byte("salt"),
			Ciphertext:     payload,
			WriteTokenHash: []byte(testHash),
			UpdatedAt:      testTime(),
		})
		if errors.Is(err, modelpage.ErrQuotaExceeded) {
			return
		}

		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	t.Fatal("database accepted writes beyond its configured size limit")
}

func TestOpenRejectsNegativeDatabaseSizeLimit(t *testing.T) {
	t.Parallel()

	_, err := repositorypage.Open(t.Context(), testDatabasePath(t), -1)
	if err == nil {
		t.Fatal("Open() error = nil, want non-nil")
	}
}
