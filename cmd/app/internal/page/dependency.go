//go:build wails

package page

import (
	"context"

	modelpage "singlepage/internal/model/page"
)

// Pages is the domain service surface used by the native binding adapter.
type Pages interface {
	Create(ctx context.Context, command modelpage.CreateServiceCmd) (modelpage.MutationResponse, error)
	Get(ctx context.Context, query modelpage.GetServiceQry) (modelpage.Page, error)
	Update(ctx context.Context, command modelpage.UpdateServiceCmd) (modelpage.MutationResponse, error)
	Rotate(ctx context.Context, command modelpage.RotateServiceCmd) (modelpage.MutationResponse, error)
}

// LocatorStore persists the native route and a crash-recovery fallback.
type LocatorStore interface {
	Read() (current string, previous string, err error)
	Write(current string, previous string) error
	WriteRemembered(current string, previous string) error
	List() ([]string, error)
}
