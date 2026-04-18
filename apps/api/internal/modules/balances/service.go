package balances

import "context"

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetOrCreateForToko(ctx context.Context, tokoID int64) (*Balance, error) {
	return s.repository.FindOrCreateByTokoID(ctx, tokoID)
}
