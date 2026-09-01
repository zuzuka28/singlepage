package page_test

import (
	"context"
	"crypto/sha256"
	"time"

	modelpage "singlepage/internal/model/page"
	servicepage "singlepage/internal/service/page"
)

const (
	testID         = "01234567-89ab-cdef-0123-456789abcdef"
	rotatedID      = "fedcba98-7654-3210-fedc-ba9876543210"
	testWriteToken = "secret-token"
	testSalt       = "salt"
	testCiphertext = "ciphertext"
	newWriteToken  = "new-token"
)

type fakeRepository struct {
	createFn func(context.Context, modelpage.CreateRepositoryCmd) error
	getFn    func(context.Context, string) (modelpage.RepositoryPage, error)
	updateFn func(context.Context, modelpage.UpdateRepositoryCmd) error
	rotateFn func(context.Context, modelpage.RotateRepositoryCmd) error
	deleteFn func(context.Context, string) error
}

func (f *fakeRepository) Create(ctx context.Context, cmd modelpage.CreateRepositoryCmd) error {
	if f.createFn == nil {
		panic("unexpected Create call")
	}

	return f.createFn(ctx, cmd)
}

func (f *fakeRepository) Get(ctx context.Context, id string) (modelpage.RepositoryPage, error) {
	if f.getFn == nil {
		panic("unexpected Get call")
	}

	return f.getFn(ctx, id)
}

func (f *fakeRepository) Update(ctx context.Context, cmd modelpage.UpdateRepositoryCmd) error {
	if f.updateFn == nil {
		panic("unexpected Update call")
	}

	return f.updateFn(ctx, cmd)
}

func (f *fakeRepository) Rotate(ctx context.Context, cmd modelpage.RotateRepositoryCmd) error {
	if f.rotateFn == nil {
		panic("unexpected Rotate call")
	}

	return f.rotateFn(ctx, cmd)
}

func (f *fakeRepository) Delete(ctx context.Context, id string) error {
	if f.deleteFn == nil {
		panic("unexpected Delete call")
	}

	return f.deleteFn(ctx, id)
}

type fixedClock struct {
	now time.Time
}

func (clock fixedClock) Now() time.Time {
	return clock.now
}

func newTestService(repo servicepage.Repository) *servicepage.Service {
	config := servicepage.Config{MaxPages: 12}

	return servicepage.NewWithClock(repo, config, fixedClock{now: testTime()})
}

func storedPage() modelpage.RepositoryPage {
	return modelpage.RepositoryPage{
		Page: modelpage.Page{
			ID:         testID,
			Revision:   4,
			Salt:       []byte("old-salt"),
			Ciphertext: []byte("old-ciphertext"),
		},
		WriteTokenHash: capabilityHash(testWriteToken),
	}
}

func capabilityHash(token string) []byte {
	hash := sha256.Sum256([]byte(token))

	return hash[:]
}

func testTime() time.Time {
	return time.Unix(1_725_000_000, 0)
}
