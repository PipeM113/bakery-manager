package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FixedCost struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Name          string    `json:"name"`
	MonthlyAmount float64   `json:"monthly_amount"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type FixedCostRepository struct {
	db *pgxpool.Pool
}

func NewFixedCostRepository(db *pgxpool.Pool) *FixedCostRepository {
	return &FixedCostRepository{db: db}
}

func (r *FixedCostRepository) Create(ctx context.Context, userID, name string, amount float64) (FixedCost, error) {
	var fc FixedCost
	err := r.db.QueryRow(ctx, `
		INSERT INTO fixed_costs (user_id, name, monthly_amount)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, name, monthly_amount, is_active, created_at, updated_at`,
		userID, name, amount,
	).Scan(&fc.ID, &fc.UserID, &fc.Name, &fc.MonthlyAmount, &fc.IsActive, &fc.CreatedAt, &fc.UpdatedAt)
	if err != nil {
		return FixedCost{}, fmt.Errorf("create fixed cost: %w", err)
	}
	return fc, nil
}

func (r *FixedCostRepository) GetAll(ctx context.Context, userID string) ([]FixedCost, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, name, monthly_amount, is_active, created_at, updated_at
		FROM fixed_costs
		WHERE user_id = $1
		ORDER BY name ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("get all fixed costs: %w", err)
	}
	defer rows.Close()

	var costs []FixedCost
	for rows.Next() {
		var fc FixedCost
		if err := rows.Scan(&fc.ID, &fc.UserID, &fc.Name, &fc.MonthlyAmount, &fc.IsActive, &fc.CreatedAt, &fc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan fixed cost: %w", err)
		}
		costs = append(costs, fc)
	}
	return costs, nil
}

func (r *FixedCostRepository) Update(ctx context.Context, id, userID, name string, amount float64, isActive bool) (FixedCost, error) {
	var fc FixedCost
	err := r.db.QueryRow(ctx, `
		UPDATE fixed_costs
		SET name=$1, monthly_amount=$2, is_active=$3, updated_at=NOW()
		WHERE id=$4 AND user_id=$5
		RETURNING id, user_id, name, monthly_amount, is_active, created_at, updated_at`,
		name, amount, isActive, id, userID,
	).Scan(&fc.ID, &fc.UserID, &fc.Name, &fc.MonthlyAmount, &fc.IsActive, &fc.CreatedAt, &fc.UpdatedAt)
	if err != nil {
		return FixedCost{}, fmt.Errorf("update fixed cost: %w", err)
	}
	return fc, nil
}

func (r *FixedCostRepository) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM fixed_costs WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete fixed cost: %w", err)
	}
	return nil
}

func (r *FixedCostRepository) SumActive(ctx context.Context, userID string) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(monthly_amount), 0)
		FROM fixed_costs
		WHERE user_id=$1 AND is_active=TRUE`, userID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum active fixed costs: %w", err)
	}
	return total, nil
}
