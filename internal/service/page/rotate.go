package page

import (
	"context"
	"fmt"

	modelpage "singlepage/internal/model/page"
)

func (s *Service) Rotate(
	ctx context.Context,
	cmd modelpage.RotateServiceCmd,
) (modelpage.MutationResponse, error) {
	err := validateRotateCmd(cmd)
	if err != nil {
		return modelpage.MutationResponse{}, err
	}

	if cmd.WriteToken == "" {
		return modelpage.MutationResponse{}, modelpage.ErrUnauthorized
	}

	stored, err := s.repo.Get(ctx, cmd.OldID)
	if err != nil {
		return modelpage.MutationResponse{}, fmt.Errorf("get page for rotation: %w", err)
	}

	if !capabilityMatches(stored.WriteTokenHash, cmd.WriteToken) {
		return modelpage.MutationResponse{}, modelpage.ErrForbidden
	}

	err = s.repo.Rotate(ctx, modelpage.RotateRepositoryCmd{
		OldID:                  cmd.OldID,
		NewID:                  cmd.NewID,
		ExpectedRevision:       stored.Revision,
		Salt:                   cmd.Salt,
		Ciphertext:             cmd.Ciphertext,
		ExpectedWriteTokenHash: stored.WriteTokenHash,
		NewWriteTokenHash:      hashCapability(cmd.NewWriteToken),
		UpdatedAt:              s.clock.Now(),
	})
	if err != nil {
		return modelpage.MutationResponse{}, fmt.Errorf("rotate page: %w", err)
	}

	return modelpage.MutationResponse{Revision: 1}, nil
}

func validateRotateCmd(cmd modelpage.RotateServiceCmd) error {
	err := validateID(cmd.OldID)
	if err != nil {
		return err
	}

	err = validateID(cmd.NewID)
	if err != nil || cmd.NewID == cmd.OldID {
		return fmt.Errorf("%w: new id", modelpage.ErrInvalid)
	}

	err = validateSalt(cmd.Salt)
	if err != nil {
		return err
	}

	err = validateCiphertext(cmd.Ciphertext)
	if err != nil {
		return err
	}

	return validateCapability(cmd.NewWriteToken)
}
