//go:build wails && !windows

package session

import (
	"errors"
	"fmt"
	"os"
)

func syncDataDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory: %w", err)
	}

	return errors.Join(directory.Sync(), directory.Close())
}
