package page

import (
	"context"
	"fmt"

	modelpage "singlepage/internal/model/page"
)

func (s *Service) Create(
	ctx context.Context,
	cmd modelpage.CreateServiceCmd,
) (modelpage.MutationResponse, error) {
	err := validateCreateCmd(cmd)
	if err != nil {
		return modelpage.MutationResponse{}, err
	}

	err = s.repo.Create(ctx, modelpage.CreateRepositoryCmd{
		ID:             cmd.ID,
		Salt:           cmd.Salt,
		Ciphertext:     cmd.Ciphertext,
		WriteTokenHash: hashCapability(cmd.WriteToken),
		UpdatedAt:      s.clock.Now(),
		MaxPages:       s.config.MaxPages,
	})
	if err != nil {
		return modelpage.MutationResponse{}, fmt.Errorf("create page: %w", err)
	}

	return modelpage.MutationResponse{Revision: 1}, nil
}

func validateCreateCmd(cmd modelpage.CreateServiceCmd) error {
	err := validateID(cmd.ID)
	if err != nil {
		return err
	}

	err = validateSalt(cmd.Salt)
	if err != nil {
		return err
	}

	err = validateCiphertext(cmd.Ciphertext)
	if err != nil {
		return err
	}

	return validateCapability(cmd.WriteToken)
}
