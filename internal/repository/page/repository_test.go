package page_test

import (
	"testing"
	"time"

	modelpage "singlepage/internal/model/page"
	repositorypage "singlepage/internal/repository/page"
)

const (
	testID       = "01234567-89ab-cdef-0123-456789abcdef"
	testSecondID = "fedcba98-7654-3210-fedc-ba9876543210"
	testDBLimit  = 8 << 20
	testHash     = "01234567890123456789012345678901"
	testNewHash  = "abcdefghijklmnopqrstuvwxyzABCDEF"
)

func openTestRepository(t *testing.T) *repositorypage.Repository {
	t.Helper()

	repository, err := repositorypage.Open(t.Context(), testDatabasePath(t), testDBLimit)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { closeTestRepository(t, repository) })

	return repository
}

func testDatabasePath(t *testing.T) string {
	t.Helper()

	return t.TempDir() + "/data.db"
}

func createTestPage(t *testing.T, repository *repositorypage.Repository) {
	t.Helper()

	err := repository.Create(t.Context(), modelpage.CreateRepositoryCmd{
		ID:             testID,
		Salt:           []byte("salt"),
		Ciphertext:     []byte("cipher"),
		WriteTokenHash: []byte(testHash),
		UpdatedAt:      testTime(),
		MaxPages:       10,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func testTime() time.Time {
	return time.Unix(123, 0)
}

func closeTestRepository(t *testing.T, repository *repositorypage.Repository) {
	t.Helper()

	err := repository.Close()
	if err != nil {
		t.Error(err)
	}
}
