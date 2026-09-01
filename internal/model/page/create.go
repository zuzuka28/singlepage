package page

import "time"

type CreateServiceCmd struct {
	ID         string
	Salt       []byte
	Ciphertext []byte
	WriteToken string
}

type CreateRepositoryCmd struct {
	ID             string
	Salt           []byte
	Ciphertext     []byte
	WriteTokenHash []byte
	UpdatedAt      time.Time
	MaxPages       int64
}
