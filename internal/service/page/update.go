package page

import (
	"context"
	"errors"
	"fmt"

	modelpage "singlepage/internal/model/page"
)

func (s *Service) Update(
	ctx context.Context,
	cmd modelpage.UpdateServiceCmd,
) (modelpage.MutationResponse, error) {
	err := validateUpdateCmd(cmd)
	if err != nil {
		return modelpage.MutationResponse{}, err
	}

	if cmd.WriteToken == "" {
		return modelpage.MutationResponse{}, modelpage.ErrUnauthorized
	}

	stored, err := s.repo.Get(ctx, cmd.ID)
	if err != nil {
		return modelpage.MutationResponse{}, fmt.Errorf("get page for update: %w", err)
	}

	if !capabilityMatches(stored.WriteTokenHash, cmd.WriteToken) {
		return modelpage.MutationResponse{}, modelpage.ErrForbidden
	}

	if stored.Revision != cmd.ExpectedRevision {
		return modelpage.MutationResponse{}, modelpage.ErrConflict
	}

	salt, err := updateSalt(stored.Salt, cmd.Salt)
	if err != nil {
		return modelpage.MutationResponse{}, err
	}

	writeTokenHash, err := updateWriteTokenHash(stored.WriteTokenHash, cmd.NewWriteToken)
	if err != nil {
		return modelpage.MutationResponse{}, err
	}

	err = s.repo.Update(ctx, modelpage.UpdateRepositoryCmd{
		ID:                     cmd.ID,
		ExpectedRevision:       cmd.ExpectedRevision,
		Salt:                   salt,
		Ciphertext:             cmd.Ciphertext,
		ExpectedWriteTokenHash: stored.WriteTokenHash,
		WriteTokenHash:         writeTokenHash,
		UpdatedAt:              s.clock.Now(),
	})
	if err != nil {
		return modelpage.MutationResponse{}, fmt.Errorf("update page: %w", err)
	}

	return modelpage.MutationResponse{Revision: cmd.ExpectedRevision + 1}, nil
}

func validateUpdateCmd(cmd modelpage.UpdateServiceCmd) error {
	err := validateID(cmd.ID)
	if err != nil {
		return err
	}

	if cmd.ExpectedRevision < 1 {
		return errors.Join(modelpage.ErrInvalid, modelpage.ErrInvalidRevision)
	}

	return validateCiphertext(cmd.Ciphertext)
}

func updateSalt(storedSalt []byte, requestedSalt *[]byte) ([]byte, error) {
	if requestedSalt == nil {
		return storedSalt, nil
	}

	err := validateSalt(*requestedSalt)
	if err != nil {
		return nil, err
	}

	return *requestedSalt, nil
}

func updateWriteTokenHash(storedHash []byte, newToken string) ([]byte, error) {
	if newToken == "" {
		return storedHash, nil
	}

	err := validateCapability(newToken)
	if err != nil {
		return nil, err
	}

	return hashCapability(newToken), nil
}
