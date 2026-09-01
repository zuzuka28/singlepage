//go:build wails

package page

import (
	"context"
	"errors"

	modelpage "singlepage/internal/model/page"
)

func (service *Service) RotatePage(
	ctx context.Context,
	id string,
	writeToken string,
	request RotatePageRequest,
	locator string,
) (MutationResponse, error) {
	salt, ciphertext, err := decodePayload(request.Salt, request.Ciphertext)
	if err != nil {
		return MutationResponse{}, err
	}

	previous, err := service.prepareLocator(ctx, locator, request.NewID)
	if err != nil {
		return MutationResponse{}, err
	}

	response, err := service.pages.Rotate(ctx, modelpage.RotateServiceCmd{
		OldID: id, NewID: request.NewID, Salt: salt, Ciphertext: ciphertext,
		WriteToken: writeToken, NewWriteToken: request.NewWriteToken,
	})
	if err != nil {
		return MutationResponse{}, errors.Join(err, service.locators.Write(previous, ""))
	}

	_ = service.locators.Write(locator, "")

	return MutationResponse(response), nil
}
