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
	rotate func(context.Context, modelpage.RotateServiceCmd) (modelpage.MutationResponse, error)
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

func (pages *fakePages) Rotate(
	ctx context.Context,
	command modelpage.RotateServiceCmd,
) (modelpage.MutationResponse, error) {
	if pages.rotate != nil {
		return pages.rotate(ctx, command)
	}

	return modelpage.MutationResponse{}, nil
}

type memoryLocators struct {
	current  string
	previous string
	history  []string
}

func (store *memoryLocators) Read() (current string, previous string, err error) {
	return store.current, store.previous, nil
}

func (store *memoryLocators) Write(current, previous string) error {
	store.current = current
	store.previous = previous

	return nil
}

func (store *memoryLocators) WriteRemembered(current, previous string) error {
	store.current = current

	store.previous = previous

	if current != "/" {
		store.history = append([]string{current}, store.history...)
	}

	return nil
}

func (store *memoryLocators) List() ([]string, error) {
	return append([]string(nil), store.history...), nil
}

func TestCreatePageCommitsRecoverableLocator(t *testing.T) {
	t.Parallel()

	locators := &memoryLocators{current: "/", previous: ""}
	locator := "/p/0123456789abcdef#0123456789abcdef"
	pages := &fakePages{
		create: func(context.Context, modelpage.CreateServiceCmd) (modelpage.MutationResponse, error) {
			if locators.previous != "/" {
				t.Fatal("locator was not made durable before page creation")
			}

			if len(locators.history) != 1 || locators.history[0] != locator {
				t.Fatalf("locator history was not durable before page creation: %#v", locators.history)
			}

			return modelpage.MutationResponse{Revision: 1}, nil
		},
		get: func(context.Context, modelpage.GetServiceQry) (modelpage.Page, error) {
			return modelpage.Page{}, nil
		},
	}
	service := startService(t, pages, locators)

	response, err := service.CreatePage(context.Background(), nativepage.CreatePageRequest{
		ID: "0123456789abcdef", Salt: "AA==", Ciphertext: "AQ==", WriteToken: "token",
	}, locator)
	if err != nil || response.Revision != 1 {
		t.Fatalf("CreatePage() = %#v, %v", response, err)
	}

	if locators.current != locator || locators.previous != "" {
		t.Fatalf("locator state = %q, %q", locators.current, locators.previous)
	}

	if len(locators.history) != 1 || locators.history[0] != locator {
		t.Fatalf("locator history = %#v", locators.history)
	}
}

func TestRotatePageCommitsRecoverableLocatorWithMutation(t *testing.T) {
	t.Parallel()

	oldLocator := "/p/0123456789abcdef#0123456789abcdef"
	newLocator := "/p/fedcba9876543210#fedcba9876543210"
	locators := &memoryLocators{current: oldLocator, history: []string{oldLocator}}
	pages := &fakePages{
		create: func(context.Context, modelpage.CreateServiceCmd) (modelpage.MutationResponse, error) {
			return modelpage.MutationResponse{}, nil
		},
		get: func(context.Context, modelpage.GetServiceQry) (modelpage.Page, error) {
			return modelpage.Page{}, nil
		},
		rotate: func(_ context.Context, command modelpage.RotateServiceCmd) (modelpage.MutationResponse, error) {
			if locators.current != newLocator || locators.previous != oldLocator {
				t.Fatalf("locator was not made recoverable before rotation: %q, %q", locators.current, locators.previous)
			}

			if command.OldID != "0123456789abcdef" || command.NewID != "fedcba9876543210" {
				t.Fatalf("rotation ids = %q -> %q", command.OldID, command.NewID)
			}

			return modelpage.MutationResponse{Revision: 2}, nil
		},
	}
	service := startService(t, pages, locators)

	response, err := service.RotatePage(
		context.Background(),
		"0123456789abcdef",
		"old-token",
		nativepage.RotatePageRequest{
			NewID: "fedcba9876543210", Salt: "AA==", Ciphertext: "AQ==", NewWriteToken: "new-token",
		},
		newLocator,
	)
	if err != nil || response.Revision != 2 {
		t.Fatalf("RotatePage() = %#v, %v", response, err)
	}

	if locators.current != newLocator || locators.previous != "" {
		t.Fatalf("committed locator state = %q, %q", locators.current, locators.previous)
	}
}

func TestListLocatorsOmitsPagesThatNoLongerExist(t *testing.T) {
	t.Parallel()

	existing := "/p/1111111111111111#1111111111111111"
	missing := "/p/2222222222222222#2222222222222222"
	locators := &memoryLocators{current: existing, history: []string{missing, existing}}
	pages := &fakePages{
		create: func(context.Context, modelpage.CreateServiceCmd) (modelpage.MutationResponse, error) {
			return modelpage.MutationResponse{}, nil
		},
		get: func(_ context.Context, query modelpage.GetServiceQry) (modelpage.Page, error) {
			if query.ID == "2222222222222222" {
				return modelpage.Page{}, modelpage.ErrNotFound
			}

			return modelpage.Page{}, nil
		},
	}
	service := startService(t, pages, locators)

	got, err := service.ListLocators(context.Background())
	if err != nil {
		t.Fatalf("ListLocators: %v", err)
	}

	if len(got) != 1 || got[0] != existing {
		t.Fatalf("ListLocators() = %#v, want [%q]", got, existing)
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
