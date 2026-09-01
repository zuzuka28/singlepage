package page

import (
	"context"
	"fmt"

	modelpage "singlepage/internal/model/page"
)

func (s *Service) Get(ctx context.Context, qry modelpage.GetServiceQry) (modelpage.Page, error) {
	err := validateID(qry.ID)
	if err != nil {
		return modelpage.Page{}, err
	}

	stored, err := s.repo.Get(ctx, qry.ID)
	if err != nil {
		return modelpage.Page{}, fmt.Errorf("get page: %w", err)
	}

	return stored.Page, nil
}
