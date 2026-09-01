//go:build wails

package page

import (
	"context"

	modelpage "singlepage/internal/model/page"
)

func (service *Service) UpdatePage(
	ctx context.Context,
	id string,
	writeToken string,
	request UpdatePageRequest,
) (MutationResponse, error) {
	ciphertext, err := decodeBase64("ciphertext", request.Ciphertext)
	if err != nil {
		return MutationResponse{}, err
	}

	var salt *[]byte

	if request.Salt != nil {
		decoded, decodeErr := decodeBase64("salt", *request.Salt)
		if decodeErr != nil {
			return MutationResponse{}, decodeErr
		}

		salt = &decoded
	}

	response, err := service.pages.Update(ctx, modelpage.UpdateServiceCmd{
		ID: id, ExpectedRevision: request.ExpectedRevision, Ciphertext: ciphertext,
		Salt: salt, WriteToken: writeToken, NewWriteToken: request.NewWriteToken,
	})

	return MutationResponse(response), err
}
