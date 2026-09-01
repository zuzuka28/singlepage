//go:build wails && windows

package session

func syncDataDirectory(string) error {
	// Windows has no portable directory fsync. os.Rename still atomically
	// replaces the locator on the same volume.
	return nil
}
