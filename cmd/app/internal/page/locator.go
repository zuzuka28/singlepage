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
		err = service.locators.WriteRemembered(current, "")
		if err != nil {
			return "", fmt.Errorf("confirm recovered native locator: %w", err)
		}

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
	err := service.locators.WriteRemembered(locator, "")
	if err != nil {
		return fmt.Errorf("remember native locator: %w", err)
	}

	return nil
}

// ListLocators returns usable native page locators in most-recently-opened order.
func (service *Service) ListLocators(ctx context.Context) ([]string, error) {
	locators, err := service.locators.List()
	if err != nil {
		return nil, fmt.Errorf("list native locators: %w", err)
	}

	available := make([]string, 0, len(locators))
	for _, locator := range locators {
		id, locatorErr := locatorID(locator)
		if locatorErr != nil {
			return nil, locatorErr
		}

		_, pageErr := service.pages.Get(ctx, modelpage.GetServiceQry{ID: id})
		if pageErr == nil {
			available = append(available, locator)
			continue
		}

		if !errors.Is(pageErr, modelpage.ErrNotFound) {
			return nil, fmt.Errorf("check native locator %q: %w", locator, pageErr)
		}
	}

	return available, nil
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

	err = service.locators.WriteRemembered(locator, previous)
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
