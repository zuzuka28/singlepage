package page

import (
	"context"
	"time"

	modelpage "singlepage/internal/model/page"
)

type Repository interface {
	Create(ctx context.Context, cmd modelpage.CreateRepositoryCmd) error
	Get(ctx context.Context, id string) (modelpage.RepositoryPage, error)
	Update(ctx context.Context, cmd modelpage.UpdateRepositoryCmd) error
	Rotate(ctx context.Context, cmd modelpage.RotateRepositoryCmd) error
	Delete(ctx context.Context, id string) error
}

type Clock interface {
	Now() time.Time
}
