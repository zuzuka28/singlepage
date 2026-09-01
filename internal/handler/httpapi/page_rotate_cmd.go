package httpapi

import (
	"context"
	"errors"
	"fmt"

	"singlepage/internal/handler/httpapi/gen"
	"singlepage/internal/middleware"
	modelpage "singlepage/internal/model/page"
)

// PageRotateCmd atomically rotates the page identifier and capability.
func (h *handler) PageRotateCmd(
	ctx context.Context,
	request gen.PageRotateCmdRequestObject,
) (gen.PageRotateCmdResponseObject, error) {
	if request.Body == nil {
		return gen.PageRotateCmd400JSONResponse{BadRequestJSONResponse: badRequestResponse()}, nil
	}

	response, err := h.pages.Rotate(ctx, modelpage.RotateServiceCmd{
		OldID:         request.Id,
		NewID:         request.Body.NewId,
		Salt:          request.Body.Salt,
		Ciphertext:    request.Body.Ciphertext,
		WriteToken:    middleware.BearerTokenFromContext(ctx),
		NewWriteToken: request.Body.NewWriteToken,
	})
	if err != nil {
		switch {
		case errors.Is(err, modelpage.ErrInvalid):
			return gen.PageRotateCmd400JSONResponse{BadRequestJSONResponse: badRequestResponse()}, nil

		case errors.Is(err, modelpage.ErrUnauthorized):
			return gen.PageRotateCmd401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil

		case errors.Is(err, modelpage.ErrForbidden):
			return gen.PageRotateCmd403JSONResponse{ForbiddenJSONResponse: forbiddenResponse()}, nil

		case errors.Is(err, modelpage.ErrNotFound):
			return gen.PageRotateCmd404JSONResponse{NotFoundJSONResponse: notFoundResponse()}, nil

		case errors.Is(err, modelpage.ErrConflict), errors.Is(err, modelpage.ErrConcurrentChange):
			return gen.PageRotateCmd409JSONResponse{ConflictJSONResponse: conflictResponse()}, nil

		case errors.Is(err, modelpage.ErrQuotaExceeded):
			return gen.PageRotateCmd507JSONResponse{
				InsufficientStorageJSONResponse: insufficientStorageResponse(),
			}, nil

		default:
			return nil, fmt.Errorf("rotate page: %w", err)
		}
	}

	return gen.PageRotateCmd201JSONResponse{Revision: response.Revision}, nil
}
