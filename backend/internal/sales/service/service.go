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

func (s *SaleService) CreateOrder(ctx context.Context, in repository.OrderInput) (repository.Order, error) {
	return s.repo.CreateOrder(ctx, in)
}

func (s *SaleService) ListOrders(ctx context.Context, userID, month string) ([]repository.Order, error) {
	return s.repo.ListOrders(ctx, userID, month)
}

func (s *SaleService) GetOrder(ctx context.Context, userID, id string) (repository.Order, error) {
	return s.repo.GetOrder(ctx, userID, id)
}

func (s *SaleService) DeleteOrder(ctx context.Context, userID, id string) error {
	return s.repo.DeleteOrder(ctx, userID, id)
}
