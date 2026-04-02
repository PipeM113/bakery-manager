package service

import (
	"context"

	"github.com/PipeM113/bakery-manager/internal/sales/repository"
)

type SaleService struct {
	repo *repository.SaleRepository
}

func NewSaleService(repo *repository.SaleRepository) *SaleService {
	return &SaleService{repo: repo}
}

func (s *SaleService) RegisterSale(ctx context.Context, input repository.RegisterSaleInput) (repository.Sale, error) {
	return s.repo.RegisterSale(ctx, input)
}

func (s *SaleService) GetAll(ctx context.Context, userID, from, to string) ([]repository.Sale, error) {
	return s.repo.GetAll(ctx, userID, from, to)
}

func (s *SaleService) DeleteSale(ctx context.Context, id, userID string) error {
	return s.repo.DeleteSale(ctx, id, userID)
}
