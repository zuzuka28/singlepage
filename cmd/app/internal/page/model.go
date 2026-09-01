//go:build wails

package page

type CreatePageRequest struct {
	ID         string `json:"id"`
	Salt       string `json:"salt"`
	Ciphertext string `json:"ciphertext"`
	WriteToken string `json:"writeToken"`
}

type UpdatePageRequest struct {
	ExpectedRevision int64   `json:"expectedRevision"`
	Ciphertext       string  `json:"ciphertext"`
	Salt             *string `json:"salt,omitempty"`
	NewWriteToken    string  `json:"newWriteToken,omitempty"`
}

type RotatePageRequest struct {
	NewID         string `json:"newId"`
	Salt          string `json:"salt"`
	Ciphertext    string `json:"ciphertext"`
	NewWriteToken string `json:"newWriteToken"`
}

type MutationResponse struct {
	Revision int64 `json:"revision"`
}

type Response struct {
	Revision   int64  `json:"revision"`
	Salt       string `json:"salt"`
	Ciphertext string `json:"ciphertext"`
}
