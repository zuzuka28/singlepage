package page_test

import (
	"context"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestServiceGet(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{getFn: func(_ context.Context, id string) (modelpage.RepositoryPage, error) {
		if id != testID {
			t.Fatalf("id = %q", id)
		}

		return storedPage(), nil
	}}

	page, err := newTestService(repo).Get(context.Background(), modelpage.GetServiceQry{ID: testID})
	if err != nil {
		t.Fatal(err)
	}

	if page.ID != testID || page.Revision != 4 || string(page.Ciphertext) != "old-ciphertext" {
		t.Fatalf("unexpected page: %+v", page)
	}
}
