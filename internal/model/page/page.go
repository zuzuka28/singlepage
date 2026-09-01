package page

import "time"

type Page struct {
	ID         string
	Revision   int64
	Salt       []byte
	Ciphertext []byte
	UpdatedAt  time.Time
}

type RepositoryPage struct {
	Page

	WriteTokenHash []byte
}

type MutationResponse struct {
	Revision int64
}
