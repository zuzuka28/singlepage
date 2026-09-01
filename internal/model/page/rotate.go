package page

import "time"

type RotateServiceCmd struct {
	OldID         string
	NewID         string
	Salt          []byte
	Ciphertext    []byte
	WriteToken    string
	NewWriteToken string
}

type RotateRepositoryCmd struct {
	OldID                  string
	NewID                  string
	ExpectedRevision       int64
	Salt                   []byte
	Ciphertext             []byte
	ExpectedWriteTokenHash []byte
	NewWriteTokenHash      []byte
	UpdatedAt              time.Time
}
