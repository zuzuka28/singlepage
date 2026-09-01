package httpapi

import (
	"context"
	"errors"
	"fmt"

	"singlepage/internal/handler/httpapi/gen"
	modelpage "singlepage/internal/model/page"
)

// PageFetchQry fetches an encrypted page.
func (h *handler) PageFetchQry(
	ctx context.Context,
	request gen.PageFetchQryRequestObject,
) (gen.PageFetchQryResponseObject, error) {
	page, err := h.pages.Get(ctx, modelpage.GetServiceQry{ID: request.Id})
	if err != nil {
		if errors.Is(err, modelpage.ErrNotFound) {
			return gen.PageFetchQry404JSONResponse{NotFoundJSONResponse: notFoundResponse()}, nil
		}

		return nil, fmt.Errorf("fetch page: %w", err)
	}

	return gen.PageFetchQry200JSONResponse{
		Revision:   page.Revision,
		Salt:       page.Salt,
		Ciphertext: page.Ciphertext,
	}, nil
}
