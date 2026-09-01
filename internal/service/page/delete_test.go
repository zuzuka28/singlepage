package page_test

import (
	"context"
	"errors"
	"testing"

	modelpage "singlepage/internal/model/page"
)

func TestServiceDelete(t *testing.T) {
	t.Parallel()

	repo := &fakeRepository{deleteFn: func(context.Context, string) error {
		return modelpage.ErrNotFound
	}}

	err := newTestService(repo).Delete(context.Background(), modelpage.DeleteServiceCmd{ID: testID})
	if !errors.Is(err, modelpage.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}
