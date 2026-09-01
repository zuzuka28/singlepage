package page_test

import (
	"bytes"
	"context"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestServiceRotate(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{
		getFn: func(context.Context, string) (modelpage.RepositoryPage, error) {
			return storedPage(), nil
		},
		rotateFn: func(_ context.Context, cmd modelpage.RotateRepositoryCmd) error {
			assertRotateCommand(t, cmd)

			return nil
		},
	}

	response, err := newTestService(repo).Rotate(context.Background(), modelpage.RotateServiceCmd{
		OldID:         testID,
		NewID:         rotatedID,
		Salt:          []byte(testSalt),
		Ciphertext:    []byte("cipher"),
		WriteToken:    testWriteToken,
		NewWriteToken: newWriteToken,
	})
	if err != nil {
		t.Fatal(err)
	}

	if response.Revision != 1 {
		t.Fatalf("revision = %d, want 1", response.Revision)
	}
}

func assertRotateCommand(t *testing.T, cmd modelpage.RotateRepositoryCmd) {
	t.Helper()

	if cmd.OldID != testID || cmd.NewID != rotatedID || cmd.ExpectedRevision != 4 {
		t.Fatalf("unexpected rotate identity: %+v", cmd)
	}

	if string(cmd.Salt) != testSalt || string(cmd.Ciphertext) != "cipher" {
		t.Fatalf("unexpected rotate payload: %+v", cmd)
	}

	if !bytes.Equal(cmd.ExpectedWriteTokenHash, capabilityHash(testWriteToken)) {
		t.Fatal("expected capability hash is incorrect")
	}

	if !bytes.Equal(cmd.NewWriteTokenHash, capabilityHash(newWriteToken)) {
		t.Fatal("new capability hash is incorrect")
	}
}
