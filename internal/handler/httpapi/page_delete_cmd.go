package httpapi

import (
	"context"
	"errors"
	"fmt"

	"singlepage/internal/handler/httpapi/gen"
	modelpage "singlepage/internal/model/page"
)

// PageDeleteCmd deletes a page after administrative middleware authorization.
func (h *handler) PageDeleteCmd(
	ctx context.Context,
	request gen.PageDeleteCmdRequestObject,
) (gen.PageDeleteCmdResponseObject, error) {
	err := h.pages.Delete(ctx, modelpage.DeleteServiceCmd{ID: request.Id})
	if err != nil {
		if errors.Is(err, modelpage.ErrNotFound) {
			return gen.PageDeleteCmd404JSONResponse{NotFoundJSONResponse: notFoundResponse()}, nil
		}

		return nil, fmt.Errorf("delete page: %w", err)
	}

	return gen.PageDeleteCmd204Response{}, nil
}
