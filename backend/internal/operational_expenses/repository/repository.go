package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OperationalExpense struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Description string    `json:"description"`
	Amount      int64     `json:"amount"`
	CategoryID  string    `json:"category_id"`
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

// ErrUnknownCategory means the expense points at a category that does not exist.
var ErrUnknownCategory = errors.New("la categoría no existe")

const expenseColumns = `e.id, e.user_id, e.description, e.amount, e.category_id, c.name, e.expense_date, e.created_at, e.updated_at`

func scanExpense(row pgx.Row) (OperationalExpense, error) {
	var e OperationalExpense
	var rawDate time.Time
	if err := row.Scan(&e.ID, &e.UserID, &e.Description, &e.Amount, &e.CategoryID, &e.Category, &rawDate, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return OperationalExpense{}, err
	}
	e.ExpenseDate = rawDate.Format("2006-01-02")
	return e, nil
}

func mapCategoryError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrUnknownCategory
	}
	return err
}

func (r *ExpenseRepository) Create(ctx context.Context, userID, description, categoryID, expenseDate string, amount int64) (OperationalExpense, error) {
	e, err := scanExpense(r.db.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO operational_expenses (user_id, description, amount, category_id, expense_date)
			VALUES ($1, $2, $3, $4, $5) RETURNING *
		)
		SELECT `+expenseColumns+` FROM ins e JOIN expense_categories c ON c.id = e.category_id`,
		userID, description, amount, categoryID, expenseDate))
	if err != nil {
		return OperationalExpense{}, fmt.Errorf("create operational expense: %w", mapCategoryError(err))
	}
	return e, nil
}

func (r *ExpenseRepository) GetByMonth(ctx context.Context, userID string, month, year int) ([]OperationalExpense, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+expenseColumns+`
		FROM operational_expenses e JOIN expense_categories c ON c.id = e.category_id
		WHERE e.user_id = $1
		  AND EXTRACT(MONTH FROM e.expense_date) = $2
		  AND EXTRACT(YEAR  FROM e.expense_date) = $3
		ORDER BY e.expense_date DESC, e.created_at DESC`,
		userID, month, year,
	)
	if err != nil {
		return nil, fmt.Errorf("get expenses by month: %w", err)
	}
	defer rows.Close()

	var expenses []OperationalExpense
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, fmt.Errorf("scan operational expense: %w", err)
		}
		expenses = append(expenses, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read operational expenses: %w", err)
	}
	return expenses, nil
}

func (r *ExpenseRepository) Update(ctx context.Context, id, userID, description, categoryID, expenseDate string, amount int64) (OperationalExpense, error) {
	e, err := scanExpense(r.db.QueryRow(ctx, `
		WITH upd AS (
			UPDATE operational_expenses
			SET description=$1, amount=$2, category_id=$3, expense_date=$4, updated_at=NOW()
			WHERE id=$5 AND user_id=$6 RETURNING *
		)
		SELECT `+expenseColumns+` FROM upd e JOIN expense_categories c ON c.id = e.category_id`,
		description, amount, categoryID, expenseDate, id, userID))
	if err != nil {
		return OperationalExpense{}, fmt.Errorf("update operational expense: %w", mapCategoryError(err))
	}
	return e, nil
}

func (r *ExpenseRepository) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM operational_expenses WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete operational expense: %w", err)
	}
	return nil
}
