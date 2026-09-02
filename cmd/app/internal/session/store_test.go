//go:build wails

package session_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"singlepage/cmd/app/internal/session"
)

func TestStoreRoundTrip(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	store := session.New(directory)
	current := "/p/0123456789abcdef#0123456789abcdef"
	previous := "/p/fedcba9876543210#fedcba9876543210"

	err := store.Write(current, previous)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	gotCurrent, gotPrevious, err := store.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if gotCurrent != current || gotPrevious != previous {
		t.Fatalf("Read() = %q, %q", gotCurrent, gotPrevious)
	}

	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("ReadDir() = %v, %v", entries, err)
	}

	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("locator mode = %o, want 600", info.Mode().Perm())
	}
}

func TestStoreRemembersLocatorsNewestFirst(t *testing.T) {
	t.Parallel()

	store := session.New(t.TempDir())
	first := "/p/1111111111111111#1111111111111111"
	second := "/p/2222222222222222#2222222222222222"

	err := store.WriteRemembered(first, "")
	if err != nil {
		t.Fatalf("WriteRemembered(first): %v", err)
	}

	err = store.WriteRemembered(second, "")
	if err != nil {
		t.Fatalf("WriteRemembered(second): %v", err)
	}

	err = store.WriteRemembered(first, "")
	if err != nil {
		t.Fatalf("WriteRemembered(first again): %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	want := []string{first, second}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestStoreWritePreservesHistory(t *testing.T) {
	t.Parallel()

	store := session.New(t.TempDir())
	remembered := "/p/1111111111111111#1111111111111111"
	current := "/p/2222222222222222#2222222222222222"

	err := store.WriteRemembered(remembered, "")
	if err != nil {
		t.Fatalf("WriteRemembered: %v", err)
	}

	err = store.Write(current, remembered)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(got) != 1 || got[0] != remembered {
		t.Fatalf("List() = %#v, want [%q]", got, remembered)
	}
}

func TestStoreReadsStateWithoutHistory(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	current := "/p/0123456789abcdef#0123456789abcdef"

	raw, err := json.Marshal(map[string]string{"current": current})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	err = os.WriteFile(filepath.Join(directory, "session-locator"), raw, 0o600)
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := session.New(directory)

	gotCurrent, gotPrevious, err := store.Read()
	if err != nil || gotCurrent != current || gotPrevious != "" {
		t.Fatalf("Read() = %q, %q, %v", gotCurrent, gotPrevious, err)
	}

	history, err := store.List()
	if err != nil || len(history) != 0 {
		t.Fatalf("List() = %#v, %v", history, err)
	}
}

func TestStoreDefaultsToRoot(t *testing.T) {
	t.Parallel()

	current, previous, err := session.New(t.TempDir()).Read()
	if err != nil || current != "/" || previous != "" {
		t.Fatalf("Read() = %q, %q, %v", current, previous, err)
	}
}
