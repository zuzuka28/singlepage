package page_test

import (
	"errors"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestCreateEnforcesPageQuota(t *testing.T) {
	t.Parallel()

	repository := openTestRepository(t)
	createTestPage(t, repository)

	err := repository.Create(t.Context(), modelpage.CreateRepositoryCmd{
		ID:             testSecondID,
		Salt:           []byte("salt"),
		Ciphertext:     []byte("cipher"),
		WriteTokenHash: []byte(testHash),
		UpdatedAt:      testTime(),
		MaxPages:       1,
	})
	if !errors.Is(err, modelpage.ErrQuotaExceeded) {
		t.Fatalf("Create() error = %v, want %v", err, modelpage.ErrQuotaExceeded)
	}
}

func TestCreateRejectsDuplicateID(t *testing.T) {
	t.Parallel()

	repository := openTestRepository(t)
	createTestPage(t, repository)

	err := repository.Create(t.Context(), modelpage.CreateRepositoryCmd{
		ID:             testID,
		Salt:           []byte("salt"),
		Ciphertext:     []byte("cipher"),
		WriteTokenHash: []byte(testHash),
		UpdatedAt:      testTime(),
	})
	if !errors.Is(err, modelpage.ErrConflict) {
		t.Fatalf("Create() error = %v, want %v", err, modelpage.ErrConflict)
	}
}
