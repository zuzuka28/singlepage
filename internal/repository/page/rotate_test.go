package page_test

import (
	"errors"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestRotateUsesRevisionAndCapabilityCAS(t *testing.T) {
	t.Parallel()

	repository := openTestRepository(t)
	createTestPage(t, repository)

	command := modelpage.RotateRepositoryCmd{
		OldID:                  testID,
		NewID:                  testSecondID,
		ExpectedRevision:       2,
		Salt:                   []byte("rotated"),
		Ciphertext:             []byte("rotated"),
		ExpectedWriteTokenHash: []byte(testHash),
		NewWriteTokenHash:      []byte(testNewHash),
		UpdatedAt:              testTime(),
	}

	err := repository.Rotate(t.Context(), command)
	if !errors.Is(err, modelpage.ErrConcurrentChange) {
		t.Fatalf("stale Rotate() error = %v, want %v", err, modelpage.ErrConcurrentChange)
	}

	command.ExpectedRevision = 1
	command.ExpectedWriteTokenHash = []byte(testNewHash)

	err = repository.Rotate(t.Context(), command)
	if !errors.Is(err, modelpage.ErrConcurrentChange) {
		t.Fatalf("unauthorized Rotate() error = %v, want %v", err, modelpage.ErrConcurrentChange)
	}

	command.ExpectedWriteTokenHash = []byte(testHash)

	err = repository.Rotate(t.Context(), command)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repository.Get(t.Context(), testID)
	if !errors.Is(err, modelpage.ErrNotFound) {
		t.Fatalf("Get(old ID) error = %v, want %v", err, modelpage.ErrNotFound)
	}
}
