//go:build wails

package app

import (
	"encoding/json"
	"errors"

	modelpage "singlepage/internal/model/page"
)

type marshaledError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func marshalError(err error) []byte {
	status := 500

	switch {
	case errors.Is(err, modelpage.ErrInvalid):
		status = 400

	case errors.Is(err, modelpage.ErrUnauthorized):
		status = 401

	case errors.Is(err, modelpage.ErrForbidden):
		status = 403

	case errors.Is(err, modelpage.ErrNotFound):
		status = 404

	case errors.Is(err, modelpage.ErrConflict), errors.Is(err, modelpage.ErrConcurrentChange):
		status = 409

	case errors.Is(err, modelpage.ErrQuotaExceeded):
		status = 507

	default:
	}

	encoded, marshalErr := json.Marshal(marshaledError{Status: status, Message: err.Error()})
	if marshalErr != nil {
		return nil
	}

	return encoded
}
