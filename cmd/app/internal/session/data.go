//go:build wails

package session

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	dataDirectoryName = "Singlepage"
	goosLinux         = "linux"
)

// DataDir returns the platform-owned native application data directory.
func DataDir() (string, error) {
	return resolveDataDir(runtime.GOOS, os.Getenv, os.UserHomeDir, os.UserConfigDir)
}

func resolveDataDir(
	goos string,
	getenv func(string) string,
	userHomeDir func() (string, error),
	userConfigDir func() (string, error),
) (string, error) {
	if goos == goosLinux {
		if base := getenv("XDG_DATA_HOME"); base != "" {
			return filepath.Join(base, "singlepage"), nil
		}

		home, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}

		return filepath.Join(home, ".local", "share", "singlepage"), nil
	}

	base, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user application data directory: %w", err)
	}

	return filepath.Join(base, dataDirectoryName), nil
}
