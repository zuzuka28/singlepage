//go:build wails

// Package app composes the native Wails desktop application.
package app

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"singlepage/cmd/app/internal/page"
	"singlepage/internal/middleware"
)

//go:embed all:frontend/dist
var assets embed.FS

// Run starts the native desktop application without opening a TCP listener.
func Run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false, Level: slog.LevelInfo, ReplaceAttr: nil,
	}))

	dependencies, err := openPageDependencies(context.Background(), logger)
	if err != nil {
		return err
	}
	defer dependencies.cleanup()

	pageHandler := page.NewService(dependencies.pages, dependencies.locators)

	//nolint:exhaustruct // Platform defaults are intentional.
	nativeApp := application.New(application.Options{
		Name: "Singlepage", Description: "Local-first encrypted outline", Logger: logger,
		Assets:       application.AssetOptions{Handler: nativeAssetHandler()},
		Services:     []application.Service{application.NewService(pageHandler)},
		MarshalError: marshalError,
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyRegular,
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	//nolint:exhaustruct // Wails defaults cover optional window capabilities.
	nativeApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Singlepage", Width: 1200, Height: 800, MinWidth: 360, MinHeight: 600,
		URL: "/", BackgroundColour: application.NewRGB(248, 250, 252),
	})

	err = nativeApp.Run()
	if err != nil {
		return fmt.Errorf("run native application: %w", err)
	}

	return nil
}

func nativeAssetHandler() http.Handler {
	return nativeAssetHandlerWith(application.BundledAssetFileServer(assets))
}

func nativeAssetHandlerWith(assetsHandler http.Handler) http.Handler {
	securedHandler := middleware.SecurityHeaders(assetsHandler)

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// History state uses /p/<id>, while the native app is a single embedded
		// document. Re-enter through index.html on reload without opening a TCP server.
		if strings.HasPrefix(request.URL.Path, "/p/") {
			request = request.Clone(request.Context())
			request.URL.Path = "/"
			request.URL.RawPath = ""
		}

		securedHandler.ServeHTTP(response, request)
	})
}
