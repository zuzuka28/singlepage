package frontend

import "io/fs"

type Handler struct {
	files    fs.FS
	fallback []byte
}

func newHandler(files fs.FS, fallback []byte) *Handler {
	return &Handler{files: files, fallback: fallback}
}
