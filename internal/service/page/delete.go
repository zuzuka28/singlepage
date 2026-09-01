package page

import (
	"context"
	"fmt"

	modelpage "singlepage/internal/model/page"
)

func (s *Service) Delete(ctx context.Context, cmd modelpage.DeleteServiceCmd) error {
	err := validateID(cmd.ID)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ctx, cmd.ID)
	if err != nil {
		return fmt.Errorf("delete page: %w", err)
	}

	return nil
}
