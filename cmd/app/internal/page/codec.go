//go:build wails

package page

import (
	"encoding/base64"
	"fmt"

	modelpage "singlepage/internal/model/page"
)

func decodePayload(encodedSalt, encodedCiphertext string) (salt, ciphertext []byte, returnErr error) {
	salt, err := decodeBase64("salt", encodedSalt)
	if err != nil {
		return nil, nil, err
	}

	ciphertext, err = decodeBase64("ciphertext", encodedCiphertext)
	if err != nil {
		return nil, nil, err
	}

	return salt, ciphertext, nil
}

func decodeBase64(name, encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid %s encoding: %w", name, modelpage.ErrInvalid)
	}

	return decoded, nil
}
