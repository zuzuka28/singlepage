package httpapi

import (
	"context"
	"errors"
	"fmt"

	"singlepage/internal/handler/httpapi/gen"
	"singlepage/internal/middleware"
	modelpage "singlepage/internal/model/page"
)

// PageUpdateCmd updates an encrypted page using revision CAS.
func (h *handler) PageUpdateCmd(
	ctx context.Context,
	request gen.PageUpdateCmdRequestObject,
) (gen.PageUpdateCmdResponseObject, error) {
	if request.Body == nil {
		return gen.PageUpdateCmd400JSONResponse{BadRequestJSONResponse: badRequestResponse()}, nil
	}

	token := middleware.BearerTokenFromContext(ctx)
	if token == "" && request.Body.WriteToken != nil {
		token = *request.Body.WriteToken
	}

	newWriteToken := ""

	if request.Body.NewWriteToken != nil {
		newWriteToken = *request.Body.NewWriteToken
	}

	response, err := h.pages.Update(ctx, modelpage.UpdateServiceCmd{
		ID:               request.Id,
		ExpectedRevision: request.Body.ExpectedRevision,
		Ciphertext:       request.Body.Ciphertext,
		Salt:             request.Body.Salt,
		WriteToken:       token,
		NewWriteToken:    newWriteToken,
	})
	if err != nil {
		return pageUpdateErrorResponse(err)
	}

	return gen.PageUpdateCmd200JSONResponse{Revision: response.Revision}, nil
}

func pageUpdateErrorResponse(err error) (gen.PageUpdateCmdResponseObject, error) {
	switch {
	case errors.Is(err, modelpage.ErrInvalid):
		return gen.PageUpdateCmd400JSONResponse{BadRequestJSONResponse: badRequestResponse()}, nil

	case errors.Is(err, modelpage.ErrUnauthorized):
		return gen.PageUpdateCmd401JSONResponse{UnauthorizedJSONResponse: unauthorizedResponse()}, nil

	case errors.Is(err, modelpage.ErrForbidden):
		return gen.PageUpdateCmd403JSONResponse{ForbiddenJSONResponse: forbiddenResponse()}, nil

	case errors.Is(err, modelpage.ErrNotFound):
		return gen.PageUpdateCmd404JSONResponse{NotFoundJSONResponse: notFoundResponse()}, nil

	case errors.Is(err, modelpage.ErrConflict):
		return gen.PageUpdateCmd409JSONResponse{ConflictJSONResponse: conflictResponse()}, nil

	case errors.Is(err, modelpage.ErrQuotaExceeded):
		return gen.PageUpdateCmd507JSONResponse{
			InsufficientStorageJSONResponse: insufficientStorageResponse(),
		}, nil

	default:
		return nil, fmt.Errorf("update page: %w", err)
	}
}
