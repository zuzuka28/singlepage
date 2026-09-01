package frontend

import (
	"embed"
	"io/fs"
)

// Embedding a placeholder keeps development builds valid before Vite creates the production assets.

//go:embed all:dist
var assets embed.FS

const fallbackHTML = "<!doctype html><html><body><main><h1>Frontend is not built</h1>" +
	"<p>Run npm run build and restart the server.</p></main></body></html>"

func embeddedFiles() fs.FS {
	files, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}

	return files
}
