package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OperationalExpense struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Category    string    `json:"category"`
	ExpenseDate string    `json:"expense_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ExpenseRepository struct {
	db *pgxpool.Pool
}

func NewExpenseRepository(db *pgxpool.Pool) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) Create(ctx context.Context, userID, description, category, expenseDate string, amount float64) (OperationalExpense, error) {
	var e OperationalExpense
	var rawDate time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO operational_expenses (user_id, description, amount, category, expense_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, description, amount, category, expense_date, created_at, updated_at`,
		userID, description, amount, category, expenseDate,
	).Scan(&e.ID, &e.UserID, &e.Description, &e.Amount, &e.Category, &rawDate, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return OperationalExpense{}, fmt.Errorf("create operational expense: %w", err)
	}
	e.ExpenseDate = rawDate.Format("2006-01-02")
	return e, nil
}

func (r *ExpenseRepository) GetByMonth(ctx context.Context, userID string, month, year int) ([]OperationalExpense, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, description, amount, category, expense_date, created_at, updated_at
		FROM operational_expenses
		WHERE user_id = $1
		  AND EXTRACT(MONTH FROM expense_date) = $2
		  AND EXTRACT(YEAR  FROM expense_date) = $3
		ORDER BY expense_date DESC, created_at DESC`,
		userID, month, year,
	)
	if err != nil {
		return nil, fmt.Errorf("get expenses by month: %w", err)
	}
	defer rows.Close()

	var expenses []OperationalExpense
	for rows.Next() {
		var e OperationalExpense
		var rawDate time.Time
		if err := rows.Scan(&e.ID, &e.UserID, &e.Description, &e.Amount, &e.Category, &rawDate, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan operational expense: %w", err)
		}
		e.ExpenseDate = rawDate.Format("2006-01-02")
		expenses = append(expenses, e)
	}
	return expenses, nil
}

func (r *ExpenseRepository) Update(ctx context.Context, id, userID, description, category, expenseDate string, amount float64) (OperationalExpense, error) {
	var e OperationalExpense
	var rawDate time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE operational_expenses
		SET description=$1, amount=$2, category=$3, expense_date=$4, updated_at=NOW()
		WHERE id=$5 AND user_id=$6
		RETURNING id, user_id, description, amount, category, expense_date, created_at, updated_at`,
		description, amount, category, expenseDate, id, userID,
	).Scan(&e.ID, &e.UserID, &e.Description, &e.Amount, &e.Category, &rawDate, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return OperationalExpense{}, fmt.Errorf("update operational expense: %w", err)
	}
	e.ExpenseDate = rawDate.Format("2006-01-02")
	return e, nil
}

func (r *ExpenseRepository) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM operational_expenses WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete operational expense: %w", err)
	}
	return nil
}
