package frontend

import "io/fs"

// NewHandlerForTest creates a frontend handler with injected assets.
func NewHandlerForTest(files fs.FS, fallback []byte) *Handler {
	return newHandler(files, fallback)
}
