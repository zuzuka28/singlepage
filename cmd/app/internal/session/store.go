//go:build wails

// Package session persists the native application's recoverable page locator.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const locatorFileName = "session-locator"

var (
	locatorPattern    = regexp.MustCompile(`^/p/[A-Za-z0-9_-]{16,128}#[A-Za-z0-9_-]{16,128}$`)
	errInvalidLocator = errors.New("native session locator is invalid")
)

type locatorState struct {
	Current  string `json:"current"`
	Previous string `json:"previous,omitempty"`
}

// Store atomically persists the current locator and an optional recovery fallback.
type Store struct {
	directory string
}

// New creates a locator store rooted in the native application data directory.
func New(directory string) *Store {
	return &Store{directory: directory}
}

// Read returns the current and recovery locators.
func (store *Store) Read() (current string, previous string, err error) {
	state, err := store.readState()
	if err != nil {
		return "", "", err
	}

	return state.Current, state.Previous, nil
}

// Write durably replaces the locator state.
func (store *Store) Write(current, previous string) error {
	return store.writeState(locatorState{Current: current, Previous: previous})
}

func (store *Store) readState() (locatorState, error) {
	raw, err := os.ReadFile(filepath.Join(store.directory, locatorFileName))
	if errors.Is(err, os.ErrNotExist) {
		return locatorState{Current: "/", Previous: ""}, nil
	}

	if err != nil {
		return locatorState{}, fmt.Errorf("read native session locator: %w", err)
	}

	var state locatorState
	if json.Unmarshal(raw, &state) == nil && validState(state) {
		return state, nil
	}

	legacyLocator := string(raw)
	if validLocator(legacyLocator) {
		return locatorState{Current: legacyLocator, Previous: ""}, nil
	}

	return locatorState{}, errInvalidLocator
}

func (store *Store) writeState(state locatorState) error {
	if !validState(state) {
		return errInvalidLocator
	}

	err := os.MkdirAll(store.directory, 0o700)
	if err != nil {
		return fmt.Errorf("create native data directory: %w", err)
	}

	encoded, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode native session locator: %w", err)
	}

	temporary, err := os.CreateTemp(store.directory, ".session-locator-*")
	if err != nil {
		return fmt.Errorf("create native session locator: %w", err)
	}

	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	err = temporary.Chmod(0o600)
	if err == nil {
		_, err = temporary.Write(encoded)
	}

	if err == nil {
		err = temporary.Sync()
	}

	err = errors.Join(err, temporary.Close())
	if err != nil {
		return fmt.Errorf("write native session locator: %w", err)
	}

	err = os.Rename(temporaryName, filepath.Join(store.directory, locatorFileName))
	if err != nil {
		return fmt.Errorf("replace native session locator: %w", err)
	}

	err = syncDataDirectory(store.directory)
	if err != nil {
		return fmt.Errorf("sync native data directory: %w", err)
	}

	return nil
}

func validState(state locatorState) bool {
	return validLocator(state.Current) && (state.Previous == "" || validLocator(state.Previous))
}

func validLocator(locator string) bool {
	return locator == "/" || locatorPattern.MatchString(locator)
}
