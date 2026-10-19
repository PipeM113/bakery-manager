package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when the cost does not exist or belongs to another user.
var ErrNotFound = errors.New("fixed cost not found")

// FixedCost is a cost as it stands in one month: the amount in force and whether it still counts.
// Months are "YYYY-MM-01" dates.
type FixedCost struct {
	ID            string    `json:"id"`
	CreatedBy     *string   `json:"created_by"`
	Name          string    `json:"name"`
	MonthlyAmount int64     `json:"monthly_amount"`
	IsActive      bool      `json:"is_active"`
	ValidFrom     string    `json:"valid_from"`
	EndedFrom     *string   `json:"ended_from"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type FixedCostRepository struct {
	db *pgxpool.Pool
}

func NewFixedCostRepository(db *pgxpool.Pool) *FixedCostRepository {
	return &FixedCostRepository{db: db}
}

// inMonth reads the cost with the version in force on the month $1 (a cost that has no version
// by then is not there). It is the one definition of "the cost in a month".
const inMonth = `
	SELECT fc.id, fc.created_by, fc.name, v.monthly_amount,
	       (fc.ended_from IS NULL OR $1::date < fc.ended_from) AS is_active,
	       to_char(v.valid_from, 'YYYY-MM-DD'), to_char(fc.ended_from, 'YYYY-MM-DD'),
	       fc.created_at, fc.updated_at
	FROM fixed_costs fc
	JOIN LATERAL (
		SELECT valid_from, monthly_amount FROM fixed_cost_versions
		WHERE fixed_cost_id = fc.id AND valid_from <= $1::date
		ORDER BY valid_from DESC LIMIT 1) v ON TRUE
	WHERE TRUE`

func scan(row pgx.Row) (FixedCost, error) {
	var fc FixedCost
	err := row.Scan(&fc.ID, &fc.CreatedBy, &fc.Name, &fc.MonthlyAmount, &fc.IsActive, &fc.ValidFrom, &fc.EndedFrom, &fc.CreatedAt, &fc.UpdatedAt)
	return fc, err
}

func (r *FixedCostRepository) get(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id, month string) (FixedCost, error) {
	fc, err := scan(q.QueryRow(ctx, inMonth+` AND fc.id = $2`, month, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return FixedCost{}, ErrNotFound
	}
	return fc, err
}

// Create starts a cost with its first version on validFrom ("YYYY-MM-01").
func (r *FixedCostRepository) Create(ctx context.Context, userID, name string, amount int64, validFrom string) (FixedCost, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return FixedCost{}, fmt.Errorf("create fixed cost: %w", err)
	}
	defer tx.Rollback(ctx)
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO fixed_costs (created_by, name) VALUES ($1, $2) RETURNING id`, userID, name).Scan(&id); err != nil {
		return FixedCost{}, fmt.Errorf("create fixed cost: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO fixed_cost_versions (fixed_cost_id, valid_from, monthly_amount) VALUES ($1, $2::date, $3)`, id, validFrom, amount); err != nil {
		return FixedCost{}, fmt.Errorf("create fixed cost version: %w", err)
	}
	fc, err := r.get(ctx, tx, id, validFrom)
	if err != nil {
		return FixedCost{}, fmt.Errorf("create fixed cost: %w", err)
	}
	return fc, tx.Commit(ctx)
}

// List gives the costs that exist in the month, each with the amount in force then.
func (r *FixedCostRepository) List(ctx context.Context, month string) ([]FixedCost, error) {
	rows, err := r.db.Query(ctx, inMonth+` ORDER BY fc.name ASC`, month)
	if err != nil {
		return nil, fmt.Errorf("list fixed costs: %w", err)
	}
	defer rows.Close()
	var costs []FixedCost
	for rows.Next() {
		fc, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan fixed cost: %w", err)
		}
		costs = append(costs, fc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read fixed costs: %w", err)
	}
	return costs, nil
}

// Update renames the cost, puts a version of the amount on `month` when the amount in force
// there differs (or that month already has its own version), and ends the cost from `month` or
// clears its end. It never touches the versions of other months.
func (r *FixedCostRepository) Update(ctx context.Context, id, name string, amount int64, active bool, month string) (FixedCost, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return FixedCost{}, fmt.Errorf("update fixed cost: %w", err)
	}
	defer tx.Rollback(ctx)

	var locked string
	if err := tx.QueryRow(ctx, `SELECT id FROM fixed_costs WHERE id = $1 FOR UPDATE`, id).Scan(&locked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FixedCost{}, ErrNotFound
		}
		return FixedCost{}, fmt.Errorf("update fixed cost: %w", err)
	}

	var inForce *int64
	if err := tx.QueryRow(ctx, `
		SELECT monthly_amount FROM fixed_cost_versions
		WHERE fixed_cost_id = $1 AND valid_from <= $2::date ORDER BY valid_from DESC LIMIT 1`, id, month).Scan(&inForce); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return FixedCost{}, fmt.Errorf("read amount in force: %w", err)
	}
	if inForce == nil || *inForce != amount {
		if _, err := tx.Exec(ctx, `
			INSERT INTO fixed_cost_versions (fixed_cost_id, valid_from, monthly_amount) VALUES ($1, $2::date, $3)
			ON CONFLICT (fixed_cost_id, valid_from) DO UPDATE SET monthly_amount = EXCLUDED.monthly_amount`, id, month, amount); err != nil {
			return FixedCost{}, fmt.Errorf("write version: %w", err)
		}
	}

	if active {
		_, err = tx.Exec(ctx, `UPDATE fixed_costs SET name = $1, ended_from = NULL, updated_at = NOW() WHERE id = $2`, name, id)
	} else {
		_, err = tx.Exec(ctx, `UPDATE fixed_costs SET name = $1, ended_from = $3::date, updated_at = NOW() WHERE id = $2`, name, id, month)
	}
	if err != nil {
		return FixedCost{}, fmt.Errorf("update fixed cost: %w", err)
	}
	fc, err := r.get(ctx, tx, id, month)
	if err != nil {
		return FixedCost{}, fmt.Errorf("update fixed cost: %w", err)
	}
	return fc, tx.Commit(ctx)
}

// End stops the cost from `month` on. The row and its versions stay, so the months before keep
// their figures. A cost that already ended earlier keeps the earlier end.
func (r *FixedCostRepository) End(ctx context.Context, id, month string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE fixed_costs SET ended_from = LEAST(COALESCE(ended_from, $2::date), $2::date), updated_at = NOW()
		WHERE id = $1`, id, month)
	if err != nil {
		return fmt.Errorf("end fixed cost: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
