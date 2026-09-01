//go:build wails

package page

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	modelpage "singlepage/internal/model/page"
)

var (
	locatorPattern         = regexp.MustCompile(`^/p/[A-Za-z0-9_-]{16,128}#[A-Za-z0-9_-]{16,128}$`)
	errInvalidPageLocator  = errors.New("native page locator is invalid")
	errLocatorPageMismatch = errors.New("native locator does not match the page")
)

func (service *Service) RestoreLocator(ctx context.Context) (string, error) {
	current, previous, err := service.locators.Read()
	if err != nil {
		return "", fmt.Errorf("read native locator: %w", err)
	}

	if current == "/" {
		return current, nil
	}

	id, err := locatorID(current)
	if err != nil {
		return "", err
	}

	_, err = service.pages.Get(ctx, modelpage.GetServiceQry{ID: id})
	if err == nil {
		_ = service.locators.Write(current, "")

		return current, nil
	}

	if !errors.Is(err, modelpage.ErrNotFound) {
		return "", fmt.Errorf("recover native page locator: %w", err)
	}

	fallback := previous
	if fallback == "" {
		fallback = "/"
	}

	writeErr := service.locators.Write(fallback, "")
	if writeErr != nil {
		return "", fmt.Errorf("persist recovered native locator: %w", writeErr)
	}

	return fallback, nil
}

func (service *Service) RememberLocator(locator string) error {
	err := service.locators.Write(locator, "")
	if err != nil {
		return fmt.Errorf("remember native locator: %w", err)
	}

	return nil
}

func (service *Service) prepareLocator(ctx context.Context, locator, pageID string) (string, error) {
	locatorPageID, err := locatorID(locator)
	if err != nil {
		return "", err
	}

	if locatorPageID != pageID {
		return "", errLocatorPageMismatch
	}

	previous, err := service.RestoreLocator(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve previous native locator: %w", err)
	}

	err = service.locators.Write(locator, previous)
	if err != nil {
		return "", fmt.Errorf("persist pending native locator: %w", err)
	}

	return previous, nil
}

func locatorID(locator string) (string, error) {
	if !locatorPattern.MatchString(locator) {
		return "", errInvalidPageLocator
	}

	withoutPrefix := strings.TrimPrefix(locator, "/p/")

	id, _, found := strings.Cut(withoutPrefix, "#")
	if !found {
		return "", errInvalidPageLocator
	}

	return id, nil
}
