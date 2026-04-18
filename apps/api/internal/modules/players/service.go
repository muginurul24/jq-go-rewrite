package players

import "context"

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) FindByUsername(ctx context.Context, tokoID int64, username string) (*Player, error) {
	return s.repository.FindByUsername(ctx, tokoID, username)
}

func (s *Service) UsernameMapForToko(ctx context.Context, tokoID int64) (map[string]string, error) {
	return s.repository.UsernameMapForToko(ctx, tokoID)
}
