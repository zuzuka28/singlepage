package page_test

import (
	"errors"
	"testing"
	"time"

	modelpage "singlepage/internal/model/page"
)

func TestUpdateUsesRevisionAndCapabilityCAS(t *testing.T) {
	t.Parallel()

	repository := openTestRepository(t)
	createTestPage(t, repository)

	err := repository.Update(t.Context(), modelpage.UpdateRepositoryCmd{
		ID:                     testID,
		ExpectedRevision:       1,
		Salt:                   []byte("salt-two"),
		Ciphertext:             []byte("cipher-two"),
		ExpectedWriteTokenHash: []byte(testHash),
		WriteTokenHash:         []byte(testNewHash),
		UpdatedAt:              testTime().Add(time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}

	err = repository.Update(t.Context(), modelpage.UpdateRepositoryCmd{
		ID:                     testID,
		ExpectedRevision:       1,
		Salt:                   []byte("stale"),
		Ciphertext:             []byte("stale"),
		ExpectedWriteTokenHash: []byte(testHash),
		WriteTokenHash:         []byte(testHash),
		UpdatedAt:              testTime(),
	})
	if !errors.Is(err, modelpage.ErrConflict) {
		t.Fatalf("stale Update() error = %v, want %v", err, modelpage.ErrConflict)
	}
}
