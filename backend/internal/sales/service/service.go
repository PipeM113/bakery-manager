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

func (s *SaleService) GetAll(ctx context.Context, from, to string) ([]repository.Sale, error) {
	return s.repo.GetAll(ctx, from, to)
}

func (s *SaleService) DeleteSale(ctx context.Context, id string) error {
	return s.repo.DeleteSale(ctx, id)
}

func (s *SaleService) CreateOrder(ctx context.Context, in repository.OrderInput) (repository.Order, error) {
	return s.repo.CreateOrder(ctx, in)
}

func (s *SaleService) ListOrders(ctx context.Context, month string) ([]repository.Order, error) {
	return s.repo.ListOrders(ctx, month)
}

func (s *SaleService) GetOrder(ctx context.Context, id string) (repository.Order, error) {
	return s.repo.GetOrder(ctx, id)
}

func (s *SaleService) DeleteOrder(ctx context.Context, id string) error {
	return s.repo.DeleteOrder(ctx, id)
}
