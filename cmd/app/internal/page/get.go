//go:build wails

package page

import (
	"context"
	"encoding/base64"
	"fmt"

	modelpage "singlepage/internal/model/page"
)

func (service *Service) GetPage(ctx context.Context, id string) (Response, error) {
	page, err := service.pages.Get(ctx, modelpage.GetServiceQry{ID: id})
	if err != nil {
		return Response{}, fmt.Errorf("get native page: %w", err)
	}

	return Response{
		Revision:   page.Revision,
		Salt:       base64.StdEncoding.EncodeToString(page.Salt),
		Ciphertext: base64.StdEncoding.EncodeToString(page.Ciphertext),
	}, nil
}
