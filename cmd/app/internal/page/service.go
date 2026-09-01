//go:build wails

// Package page exposes the domain page service through Wails bindings.
package page

// Service adapts domain page operations to JSON-safe Wails methods.
type Service struct {
	pages    Pages
	locators LocatorStore
}

// NewService creates a native page binding service.
func NewService(pages Pages, locators LocatorStore) *Service {
	return &Service{pages: pages, locators: locators}
}
