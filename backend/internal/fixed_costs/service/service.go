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

func (s *FixedCostService) Create(ctx context.Context, userID, name string, amount float64) (repository.FixedCost, error) {
	return s.repo.Create(ctx, userID, name, amount)
}

func (s *FixedCostService) List(ctx context.Context, userID string) ([]repository.FixedCost, error) {
	return s.repo.GetAll(ctx, userID)
}

func (s *FixedCostService) Update(ctx context.Context, id, userID, name string, amount float64, isActive bool) (repository.FixedCost, error) {
	return s.repo.Update(ctx, id, userID, name, amount, isActive)
}

func (s *FixedCostService) Delete(ctx context.Context, id, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *FixedCostService) TotalActiveMonthly(ctx context.Context, userID string) (float64, error) {
	return s.repo.SumActive(ctx, userID)
}
