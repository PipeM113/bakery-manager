package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Quotation struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	RecipeID   string    `json:"recipe_id"`
	RecipeName string    `json:"recipe_name"`
	ClientName string    `json:"client_name"`
	MarginPct  float64   `json:"margin_pct"`
	FinalPrice float64   `json:"final_price"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SaveInput struct {
	UserID     string
	RecipeID   string
	ClientName string
	MarginPct  float64
	FinalPrice float64
}

type QuotationRepository struct {
	db *pgxpool.Pool
}

func NewQuotationRepository(db *pgxpool.Pool) *QuotationRepository {
	return &QuotationRepository{db: db}
}

func (r *QuotationRepository) Save(ctx context.Context, in SaveInput) (Quotation, error) {
	var q Quotation
	err := r.db.QueryRow(ctx, `
		INSERT INTO quotations (user_id, recipe_id, client_name, margin_pct, final_price)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, recipe_id, client_name, margin_pct, final_price, status, created_at, updated_at`,
		in.UserID, in.RecipeID, in.ClientName, in.MarginPct, in.FinalPrice,
	).Scan(&q.ID, &q.UserID, &q.RecipeID, &q.ClientName, &q.MarginPct, &q.FinalPrice, &q.Status, &q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		return q, err
	}
	// Fetch recipe name
	_ = r.db.QueryRow(ctx, `SELECT name FROM recipes WHERE id = $1`, q.RecipeID).Scan(&q.RecipeName)
	return q, nil
}

func (r *QuotationRepository) List(ctx context.Context, userID string, status string) ([]Quotation, error) {
	query := `
		SELECT q.id, q.user_id, q.recipe_id, r.name, q.client_name, q.margin_pct, q.final_price, q.status, q.created_at, q.updated_at
		FROM quotations q
		JOIN recipes r ON r.id = q.recipe_id
		WHERE q.user_id = $1`
	args := []any{userID}
	if status != "" {
		query += ` AND q.status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY q.created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Quotation
	for rows.Next() {
		var q Quotation
		if err := rows.Scan(&q.ID, &q.UserID, &q.RecipeID, &q.RecipeName, &q.ClientName,
			&q.MarginPct, &q.FinalPrice, &q.Status, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, q)
	}
	return result, nil
}

func (r *QuotationRepository) UpdateStatus(ctx context.Context, id, userID, status string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE quotations SET status = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3`,
		status, id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}

var errNotFound = &notFoundError{}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "quotation not found" }

func IsNotFound(err error) bool {
	_, ok := err.(*notFoundError)
	return ok
}
