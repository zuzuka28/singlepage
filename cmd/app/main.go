//go:build wails

package main

import (
	"log/slog"
	"os"

	nativeapp "singlepage/cmd/app/internal/app"
)

func main() {
	err := nativeapp.Run()
	if err != nil {
		slog.Error("native application stopped", "error", err)
		os.Exit(1)
	}
}
