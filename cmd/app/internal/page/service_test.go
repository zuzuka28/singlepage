//go:build wails

package page_test

import (
	"context"
	"testing"

	nativepage "singlepage/cmd/app/internal/page"
	modelpage "singlepage/internal/model/page"
)

type fakePages struct {
	create func(context.Context, modelpage.CreateServiceCmd) (modelpage.MutationResponse, error)
	get    func(context.Context, modelpage.GetServiceQry) (modelpage.Page, error)
}

func (pages *fakePages) Create(
	ctx context.Context,
	command modelpage.CreateServiceCmd,
) (modelpage.MutationResponse, error) {
	return pages.create(ctx, command)
}

func (pages *fakePages) Get(ctx context.Context, query modelpage.GetServiceQry) (modelpage.Page, error) {
	return pages.get(ctx, query)
}

func (*fakePages) Update(
	context.Context,
	modelpage.UpdateServiceCmd,
) (modelpage.MutationResponse, error) {
	return modelpage.MutationResponse{}, nil
}

func (*fakePages) Rotate(
	context.Context,
	modelpage.RotateServiceCmd,
) (modelpage.MutationResponse, error) {
	return modelpage.MutationResponse{}, nil
}

type memoryLocators struct {
	current  string
	previous string
}

func (store *memoryLocators) Read() (current string, previous string, err error) {
	return store.current, store.previous, nil
}

func (store *memoryLocators) Write(current, previous string) error {
	store.current = current
	store.previous = previous

	return nil
}

func TestCreatePageCommitsRecoverableLocator(t *testing.T) {
	t.Parallel()

	locators := &memoryLocators{current: "/", previous: ""}
	pages := &fakePages{
		create: func(context.Context, modelpage.CreateServiceCmd) (modelpage.MutationResponse, error) {
			if locators.previous != "/" {
				t.Fatal("locator was not made durable before page creation")
			}

			return modelpage.MutationResponse{Revision: 1}, nil
		},
		get: func(context.Context, modelpage.GetServiceQry) (modelpage.Page, error) {
			return modelpage.Page{}, nil
		},
	}
	service := startService(t, pages, locators)
	locator := "/p/0123456789abcdef#0123456789abcdef"

	response, err := service.CreatePage(context.Background(), nativepage.CreatePageRequest{
		ID: "0123456789abcdef", Salt: "AA==", Ciphertext: "AQ==", WriteToken: "token",
	}, locator)
	if err != nil || response.Revision != 1 {
		t.Fatalf("CreatePage() = %#v, %v", response, err)
	}

	if locators.current != locator || locators.previous != "" {
		t.Fatalf("locator state = %q, %q", locators.current, locators.previous)
	}
}

func TestRestoreLocatorRollsBackUncommittedPage(t *testing.T) {
	t.Parallel()

	previous := "/p/fedcba9876543210#fedcba9876543210"
	locators := &memoryLocators{
		current: "/p/0123456789abcdef#0123456789abcdef", previous: previous,
	}
	pages := &fakePages{
		create: func(context.Context, modelpage.CreateServiceCmd) (modelpage.MutationResponse, error) {
			return modelpage.MutationResponse{}, nil
		},
		get: func(context.Context, modelpage.GetServiceQry) (modelpage.Page, error) {
			return modelpage.Page{}, modelpage.ErrNotFound
		},
	}
	service := startService(t, pages, locators)

	got, err := service.RestoreLocator(context.Background())
	if err != nil || got != previous {
		t.Fatalf("RestoreLocator() = %q, %v", got, err)
	}

	if locators.current != previous || locators.previous != "" {
		t.Fatalf("recovered state = %q, %q", locators.current, locators.previous)
	}
}

func TestCreatePagePreservesConfirmedFallbackFromPendingState(t *testing.T) {
	t.Parallel()

	confirmed := "/p/fedcba9876543210#fedcba9876543210"
	locators := &memoryLocators{
		current: "/p/1111111111111111#1111111111111111", previous: confirmed,
	}
	pages := &fakePages{
		create: func(context.Context, modelpage.CreateServiceCmd) (modelpage.MutationResponse, error) {
			if locators.previous != confirmed {
				t.Fatalf("confirmed fallback was replaced with %q", locators.previous)
			}

			return modelpage.MutationResponse{Revision: 1}, nil
		},
		get: func(context.Context, modelpage.GetServiceQry) (modelpage.Page, error) {
			return modelpage.Page{}, modelpage.ErrNotFound
		},
	}
	service := startService(t, pages, locators)
	locator := "/p/2222222222222222#2222222222222222"

	_, err := service.CreatePage(context.Background(), nativepage.CreatePageRequest{
		ID: "2222222222222222", Salt: "AA==", Ciphertext: "AQ==", WriteToken: "token",
	}, locator)
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
}

func startService(
	t *testing.T,
	pages nativepage.Pages,
	locators nativepage.LocatorStore,
) *nativepage.Service {
	t.Helper()

	return nativepage.NewService(pages, locators)
}
