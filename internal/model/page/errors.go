package page

import "errors"

var (
	ErrInvalid          = errors.New("invalid page input")
	ErrNotFound         = errors.New("page not found")
	ErrConflict         = errors.New("page conflict")
	ErrUnauthorized     = errors.New("authorization required")
	ErrForbidden        = errors.New("authorization forbidden")
	ErrQuotaExceeded    = errors.New("page storage quota exceeded")
	ErrConcurrentChange = errors.New("page changed concurrently")

	ErrInvalidID         = errors.New("invalid page id")
	ErrInvalidSalt       = errors.New("invalid page salt")
	ErrInvalidCiphertext = errors.New("invalid page ciphertext")
	ErrInvalidCapability = errors.New("invalid page capability")
	ErrInvalidRevision   = errors.New("invalid page revision")
)
