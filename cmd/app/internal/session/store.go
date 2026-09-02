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
	"sync"
)

const locatorFileName = "session-locator"

var (
	locatorPattern    = regexp.MustCompile(`^/p/[A-Za-z0-9_-]{16,128}#[A-Za-z0-9_-]{16,128}$`)
	errInvalidLocator = errors.New("native session locator is invalid")
)

type locatorState struct {
	Current  string   `json:"current"`
	Previous string   `json:"previous,omitempty"`
	History  []string `json:"history,omitempty"`
}

// Store atomically persists the current locator and an optional recovery fallback.
type Store struct {
	directory string
	mutex     sync.Mutex
}

// New creates a locator store rooted in the native application data directory.
func New(directory string) *Store {
	return &Store{directory: directory, mutex: sync.Mutex{}}
}

// Read returns the current and recovery locators.
func (store *Store) Read() (current string, previous string, err error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	state, err := store.readState()
	if err != nil {
		return "", "", err
	}

	return state.Current, state.Previous, nil
}

// Write durably replaces the locator state.
func (store *Store) Write(current, previous string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	state, err := store.readState()
	if err != nil && !errors.Is(err, errInvalidLocator) {
		return err
	}

	state.Current = current

	state.Previous = previous

	return store.writeState(state)
}

// List returns previously opened page locators, newest first.
func (store *Store) List() ([]string, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	state, err := store.readState()
	if err != nil {
		return nil, err
	}

	return append([]string(nil), state.History...), nil
}

// WriteRemembered atomically replaces the locator state and moves the current page to the history front.
func (store *Store) WriteRemembered(current, previous string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if !validLocator(current) || (previous != "" && !validLocator(previous)) {
		return errInvalidLocator
	}

	state, err := store.readState()
	if err != nil {
		return err
	}

	state.Current = current

	state.Previous = previous
	if current != "/" {
		history := make([]string, 0, len(state.History)+1)

		history = append(history, current)
		for _, remembered := range state.History {
			if remembered != current {
				history = append(history, remembered)
			}
		}

		state.History = history
	}

	return store.writeState(state)
}

func (store *Store) readState() (locatorState, error) {
	raw, err := os.ReadFile(filepath.Join(store.directory, locatorFileName))
	if errors.Is(err, os.ErrNotExist) {
		return locatorState{Current: "/", Previous: "", History: nil}, nil
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
		return locatorState{Current: legacyLocator, Previous: "", History: nil}, nil
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
	if !validLocator(state.Current) || (state.Previous != "" && !validLocator(state.Previous)) {
		return false
	}

	for _, locator := range state.History {
		if locator == "/" || !validLocator(locator) {
			return false
		}
	}

	return true
}

func validLocator(locator string) bool {
	return locator == "/" || locatorPattern.MatchString(locator)
}
