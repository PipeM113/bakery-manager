package service

import (
	"context"

	"github.com/PipeM113/bakery-manager/internal/fixed_costs/repository"
)

type FixedCostService struct {
	repo *repository.FixedCostRepository
}

func NewFixedCostService(repo *repository.FixedCostRepository) *FixedCostService {
	return &FixedCostService{repo: repo}
}

// Months are "YYYY-MM-01" dates.

func (s *FixedCostService) Create(ctx context.Context, userID, name string, amount int64, validFrom string) (repository.FixedCost, error) {
	return s.repo.Create(ctx, userID, name, amount, validFrom)
}

func (s *FixedCostService) List(ctx context.Context, month string) ([]repository.FixedCost, error) {
	return s.repo.List(ctx, month)
}

func (s *FixedCostService) Update(ctx context.Context, id, name string, amount int64, active bool, month string) (repository.FixedCost, error) {
	return s.repo.Update(ctx, id, name, amount, active, month)
}

func (s *FixedCostService) End(ctx context.Context, id, month string) error {
	return s.repo.End(ctx, id, month)
}
