package httpapi

import (
	"net/http"

	"singlepage/internal/handler/httpapi/gen"
)

func badRequestResponse() gen.BadRequestJSONResponse {
	return gen.BadRequestJSONResponse{Error: http.StatusText(http.StatusBadRequest)}
}

func conflictResponse() gen.ConflictJSONResponse {
	return gen.ConflictJSONResponse{Error: http.StatusText(http.StatusConflict)}
}

func forbiddenResponse() gen.ForbiddenJSONResponse {
	return gen.ForbiddenJSONResponse{Error: http.StatusText(http.StatusForbidden)}
}

func notFoundResponse() gen.NotFoundJSONResponse {
	return gen.NotFoundJSONResponse{Error: http.StatusText(http.StatusNotFound)}
}

func unauthorizedResponse() gen.UnauthorizedJSONResponse {
	return gen.UnauthorizedJSONResponse{Error: http.StatusText(http.StatusUnauthorized)}
}

func insufficientStorageResponse() gen.InsufficientStorageJSONResponse {
	return gen.InsufficientStorageJSONResponse{Error: http.StatusText(http.StatusInsufficientStorage)}
}
