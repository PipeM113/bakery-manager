package service

import (
	"context"

	"github.com/PipeM113/bakery-manager/internal/operational_expenses/repository"
)

type ExpenseService struct {
	repo *repository.ExpenseRepository
}

func NewExpenseService(repo *repository.ExpenseRepository) *ExpenseService {
	return &ExpenseService{repo: repo}
}

func (s *ExpenseService) Create(ctx context.Context, userID, description, categoryID, expenseDate string, amount int64) (repository.OperationalExpense, error) {
	return s.repo.Create(ctx, userID, description, categoryID, expenseDate, amount)
}

func (s *ExpenseService) ListByMonth(ctx context.Context, month, year int) ([]repository.OperationalExpense, error) {
	return s.repo.GetByMonth(ctx, month, year)
}

func (s *ExpenseService) Update(ctx context.Context, id, description, categoryID, expenseDate string, amount int64) (repository.OperationalExpense, error) {
	return s.repo.Update(ctx, id, description, categoryID, expenseDate, amount)
}

func (s *ExpenseService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
