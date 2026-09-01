package page_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestServiceCreate(t *testing.T) {
	t.Parallel()

	t.Run("hashes capability and passes limits", testCreateSuccess)
	t.Run("does not apply a separate ciphertext size limit", testCreateWithoutCiphertextSizeLimit)
	t.Run("validates input before repository call", testCreateValidation)
	t.Run("preserves repository error", testCreateRepositoryError)
}

func testCreateWithoutCiphertextSizeLimit(t *testing.T) {
	t.Parallel()

	ciphertext := bytes.Repeat([]byte("x"), 2048)
	repo := &fakeRepository{createFn: func(_ context.Context, cmd modelpage.CreateRepositoryCmd) error {
		if !bytes.Equal(cmd.Ciphertext, ciphertext) {
			t.Fatal("ciphertext was changed")
		}

		return nil
	}}
	command := validCreateCmd()
	command.Ciphertext = ciphertext

	_, err := newTestService(repo).Create(context.Background(), command)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func testCreateSuccess(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{createFn: func(_ context.Context, cmd modelpage.CreateRepositoryCmd) error {
		assertCreateCommand(t, cmd)

		return nil
	}}

	response, err := newTestService(repo).Create(context.Background(), validCreateCmd())
	if err != nil {
		t.Fatal(err)
	}

	if response.Revision != 1 {
		t.Fatalf("revision = %d, want 1", response.Revision)
	}
}

func testCreateValidation(t *testing.T) {
	t.Parallel()

	commands := []modelpage.CreateServiceCmd{
		{ID: "short", Salt: []byte(testSalt), Ciphertext: []byte(testCiphertext), WriteToken: testWriteToken},
		{ID: testID, Ciphertext: []byte(testCiphertext), WriteToken: testWriteToken},
		{ID: testID, Salt: []byte(testSalt), WriteToken: testWriteToken},
		{ID: testID, Salt: []byte(testSalt), Ciphertext: []byte(testCiphertext)},
	}

	for _, command := range commands {
		_, err := newTestService(&fakeRepository{}).Create(context.Background(), command)
		if !errors.Is(err, modelpage.ErrInvalid) {
			t.Fatalf("error = %v, want ErrInvalid", err)
		}
	}
}

func testCreateRepositoryError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{createFn: func(context.Context, modelpage.CreateRepositoryCmd) error {
		return modelpage.ErrQuotaExceeded
	}}

	_, err := newTestService(repo).Create(context.Background(), validCreateCmd())
	if !errors.Is(err, modelpage.ErrQuotaExceeded) {
		t.Fatalf("error = %v, want ErrQuotaExceeded", err)
	}
}

func validCreateCmd() modelpage.CreateServiceCmd {
	return modelpage.CreateServiceCmd{
		ID:         testID,
		Salt:       []byte(testSalt),
		Ciphertext: []byte(testCiphertext),
		WriteToken: testWriteToken,
	}
}

func assertCreateCommand(t *testing.T, cmd modelpage.CreateRepositoryCmd) {
	t.Helper()

	if cmd.ID != testID || string(cmd.Salt) != testSalt || string(cmd.Ciphertext) != testCiphertext {
		t.Fatalf("unexpected create command: %+v", cmd)
	}

	if cmd.MaxPages != 12 || !cmd.UpdatedAt.Equal(testTime()) {
		t.Fatalf("limits/time not forwarded: max=%d time=%v", cmd.MaxPages, cmd.UpdatedAt)
	}

	wantHash := capabilityHash(testWriteToken)
	if !bytes.Equal(cmd.WriteTokenHash, wantHash) {
		t.Fatalf("capability was not SHA-256 hashed: %x", cmd.WriteTokenHash)
	}

	if bytes.Equal(cmd.WriteTokenHash, []byte(testWriteToken)) {
		t.Fatal("plaintext capability was passed to repository")
	}
}
