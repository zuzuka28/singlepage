package httpapi

import (
	"context"

	modelpage "singlepage/internal/model/page"
)

type pageService interface {
	Create(
		ctx context.Context,
		cmd modelpage.CreateServiceCmd,
	) (modelpage.MutationResponse, error)
	Get(ctx context.Context, qry modelpage.GetServiceQry) (modelpage.Page, error)
	Update(
		ctx context.Context,
		cmd modelpage.UpdateServiceCmd,
	) (modelpage.MutationResponse, error)
	Rotate(
		ctx context.Context,
		cmd modelpage.RotateServiceCmd,
	) (modelpage.MutationResponse, error)
	Delete(ctx context.Context, cmd modelpage.DeleteServiceCmd) error
}
