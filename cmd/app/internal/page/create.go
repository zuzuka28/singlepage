//go:build wails

package page

import (
	"context"
	"errors"

	modelpage "singlepage/internal/model/page"
)

func (service *Service) CreatePage(
	ctx context.Context,
	request CreatePageRequest,
	locator string,
) (MutationResponse, error) {
	salt, ciphertext, err := decodePayload(request.Salt, request.Ciphertext)
	if err != nil {
		return MutationResponse{}, err
	}

	previous, err := service.prepareLocator(ctx, locator, request.ID)
	if err != nil {
		return MutationResponse{}, err
	}

	response, err := service.pages.Create(ctx, modelpage.CreateServiceCmd{
		ID: request.ID, Salt: salt, Ciphertext: ciphertext, WriteToken: request.WriteToken,
	})
	if err != nil {
		return MutationResponse{}, errors.Join(err, service.locators.Write(previous, ""))
	}

	// The pending state is already durable; cleanup is best effort. RestoreLocator
	// resolves a leftover pending state against the committed page after a crash.
	_ = service.locators.Write(locator, "")

	return MutationResponse(response), nil
}
