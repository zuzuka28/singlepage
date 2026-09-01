package page

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"

	modelpage "singlepage/internal/model/page"
)

func validateID(id string) error {
	if !modelpage.ValidID(id) {
		return errors.Join(modelpage.ErrInvalid, modelpage.ErrInvalidID)
	}

	return nil
}

func validateSalt(salt []byte) error {
	if len(salt) == 0 || len(salt) > maxSaltBytes {
		return errors.Join(modelpage.ErrInvalid, modelpage.ErrInvalidSalt)
	}

	return nil
}

func validateCiphertext(ciphertext []byte) error {
	if len(ciphertext) == 0 {
		return errors.Join(modelpage.ErrInvalid, modelpage.ErrInvalidCiphertext)
	}

	return nil
}

func validateCapability(token string) error {
	if len(token) == 0 || len(token) > maxCapabilityBytes {
		return errors.Join(modelpage.ErrInvalid, modelpage.ErrInvalidCapability)
	}

	return nil
}

func hashCapability(token string) []byte {
	hash := sha256.Sum256([]byte(token))

	return hash[:]
}

func capabilityMatches(storedHash []byte, token string) bool {
	providedHash := sha256.Sum256([]byte(token))

	return len(storedHash) == sha256.Size && subtle.ConstantTimeCompare(storedHash, providedHash[:]) == 1
}
