package page

import "time"

type UpdateServiceCmd struct {
	ID               string
	ExpectedRevision int64
	Ciphertext       []byte
	Salt             *[]byte
	WriteToken       string
	NewWriteToken    string
}

type UpdateRepositoryCmd struct {
	ID                     string
	ExpectedRevision       int64
	Salt                   []byte
	Ciphertext             []byte
	ExpectedWriteTokenHash []byte
	WriteTokenHash         []byte
	UpdatedAt              time.Time
}
