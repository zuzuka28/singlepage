package page_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestServiceUpdate(t *testing.T) {
	t.Parallel()

	t.Run("verifies capability and builds CAS", testUpdateSuccess)
	t.Run("rejects missing token before lookup", testUpdateMissingToken)
	t.Run("rejects wrong token", testUpdateWrongToken)
	t.Run("rejects stale revision", testUpdateStaleRevision)
	t.Run("preserves repository conflict", testUpdateRepositoryConflict)
}

func testUpdateSuccess(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		getFn: func(context.Context, string) (modelpage.RepositoryPage, error) {
			return storedPage(), nil
		},
		updateFn: func(_ context.Context, cmd modelpage.UpdateRepositoryCmd) error {
			assertUpdateCommand(t, cmd)

			return nil
		},
	}

	newSalt := []byte("new-salt")

	response, err := newTestService(repo).Update(context.Background(), modelpage.UpdateServiceCmd{
		ID:               testID,
		ExpectedRevision: 4,
		Ciphertext:       []byte("new-cipher"),
		Salt:             &newSalt,
		WriteToken:       testWriteToken,
		NewWriteToken:    newWriteToken,
	})
	if err != nil {
		t.Fatal(err)
	}

	if response.Revision != 5 {
		t.Fatalf("revision = %d, want 5", response.Revision)
	}
}

func testUpdateMissingToken(t *testing.T) {
	t.Parallel()

	_, err := newTestService(&fakeRepository{}).Update(context.Background(), modelpage.UpdateServiceCmd{
		ID:               testID,
		ExpectedRevision: 4,
		Ciphertext:       []byte("cipher"),
	})
	if !errors.Is(err, modelpage.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
}

func testUpdateWrongToken(t *testing.T) {
	t.Parallel()

	repo := repositoryReturningStoredPage()

	_, err := newTestService(repo).Update(context.Background(), modelpage.UpdateServiceCmd{
		ID:               testID,
		ExpectedRevision: 4,
		Ciphertext:       []byte("cipher"),
		WriteToken:       "wrong",
	})

	if !errors.Is(err, modelpage.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func testUpdateStaleRevision(t *testing.T) {
	t.Parallel()

	repo := repositoryReturningStoredPage()

	_, err := newTestService(repo).Update(context.Background(), modelpage.UpdateServiceCmd{
		ID:               testID,
		ExpectedRevision: 3,
		Ciphertext:       []byte("cipher"),
		WriteToken:       testWriteToken,
	})

	if !errors.Is(err, modelpage.ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
}

func testUpdateRepositoryConflict(t *testing.T) {
	t.Parallel()

	repo := repositoryReturningStoredPage()
	repo.updateFn = func(context.Context, modelpage.UpdateRepositoryCmd) error {
		return modelpage.ErrConflict
	}

	_, err := newTestService(repo).Update(context.Background(), modelpage.UpdateServiceCmd{
		ID:               testID,
		ExpectedRevision: 4,
		Ciphertext:       []byte("cipher"),
		WriteToken:       testWriteToken,
	})
	if !errors.Is(err, modelpage.ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
}

func repositoryReturningStoredPage() *fakeRepository {
	return &fakeRepository{getFn: func(context.Context, string) (modelpage.RepositoryPage, error) {
		return storedPage(), nil
	}}
}

func assertUpdateCommand(t *testing.T, cmd modelpage.UpdateRepositoryCmd) {
	t.Helper()

	if cmd.ID != testID || cmd.ExpectedRevision != 4 {
		t.Fatalf("unexpected update identity: %+v", cmd)
	}

	if string(cmd.Salt) != "new-salt" || string(cmd.Ciphertext) != "new-cipher" {
		t.Fatalf("unexpected update payload: %+v", cmd)
	}

	if !cmd.UpdatedAt.Equal(testTime()) {
		t.Fatalf("updated time = %v", cmd.UpdatedAt)
	}

	if !bytes.Equal(cmd.ExpectedWriteTokenHash, capabilityHash(testWriteToken)) {
		t.Fatal("expected hash was not used for repository CAS")
	}

	if !bytes.Equal(cmd.WriteTokenHash, capabilityHash(newWriteToken)) {
		t.Fatal("new capability was not hashed")
	}
}
