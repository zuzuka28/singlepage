package httpapi

import (
	"context"
	"errors"
	"fmt"

	"singlepage/internal/handler/httpapi/gen"
	modelpage "singlepage/internal/model/page"
)

// PageCreateCmd creates an encrypted page.
func (h *handler) PageCreateCmd(
	ctx context.Context,
	request gen.PageCreateCmdRequestObject,
) (gen.PageCreateCmdResponseObject, error) {
	if request.Body == nil {
		return gen.PageCreateCmd400JSONResponse{BadRequestJSONResponse: badRequestResponse()}, nil
	}

	response, err := h.pages.Create(ctx, modelpage.CreateServiceCmd{
		ID:         request.Body.Id,
		Salt:       request.Body.Salt,
		Ciphertext: request.Body.Ciphertext,
		WriteToken: request.Body.WriteToken,
	})
	if err != nil {
		switch {
		case errors.Is(err, modelpage.ErrInvalid):
			return gen.PageCreateCmd400JSONResponse{BadRequestJSONResponse: badRequestResponse()}, nil

		case errors.Is(err, modelpage.ErrConflict):
			return gen.PageCreateCmd409JSONResponse{ConflictJSONResponse: conflictResponse()}, nil

		case errors.Is(err, modelpage.ErrQuotaExceeded):
			return gen.PageCreateCmd507JSONResponse{
				InsufficientStorageJSONResponse: insufficientStorageResponse(),
			}, nil

		default:
			return nil, fmt.Errorf("create page: %w", err)
		}
	}

	return gen.PageCreateCmd201JSONResponse{Revision: response.Revision}, nil
}
