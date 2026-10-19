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

// Line is one product of a quotation as it was quoted: the name, the photo and the prices are
// copies, so the saved quotation does not change when recipes or costs do.
type Line struct {
	ID              string   `json:"id"`
	Position        int      `json:"position"`
	RecipeID        *string  `json:"recipe_id"`
	RecipeName      string   `json:"recipe_name"`
	PhotoURL        *string  `json:"-"`
	SoldAs          string   `json:"sold_as"`
	Quantity        int      `json:"quantity"`
	UnitPrice       int64    `json:"unit_price"`
	LineTotal       int64    `json:"line_total"`
	MarginPct       *float64 `json:"margin_pct"`
	IndirectCostPct *float64 `json:"indirect_cost_pct"`
	LaborCostPct    *float64 `json:"labor_cost_pct"`
	ExtraCharge     int64    `json:"extra_charge"`
}

type Quotation struct {
	ID           string    `json:"id"`
	CreatedBy    *string   `json:"created_by"`
	ClientName   string    `json:"client_name"`
	DeliveryCost int64     `json:"delivery_cost"`
	Total        int64     `json:"total"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Lines        []Line    `json:"lines"`
}

// LineInput is a line already priced by the caller.
type LineInput struct {
	RecipeID        string
	RecipeName      string
	PhotoURL        *string
	SoldAs          string
	Quantity        int
	UnitPrice       int64
	LineTotal       int64
	MarginPct       float64
	IndirectCostPct float64
	LaborCostPct    float64
	ExtraCharge     int64
}

type SaveInput struct {
	CreatedBy    string
	ClientName   string
	DeliveryCost int64
	CreatedAt    time.Time // from the business clock
	Lines        []LineInput
}

type QuotationRepository struct {
	db *pgxpool.Pool
}

func NewQuotationRepository(db *pgxpool.Pool) *QuotationRepository {
	return &QuotationRepository{db: db}
}

const quotationColumns = `id, created_by, client_name, delivery_cost, final_price, status, created_at, updated_at`

func scanQuotation(row pgx.Row) (Quotation, error) {
	var q Quotation
	err := row.Scan(&q.ID, &q.CreatedBy, &q.ClientName, &q.DeliveryCost, &q.Total, &q.Status, &q.CreatedAt, &q.UpdatedAt)
	q.Lines = []Line{}
	return q, err
}

// Save stores the quotation and its lines together, or nothing. The total is the sum of the lines
// and the delivery.
func (r *QuotationRepository) Save(ctx context.Context, in SaveInput) (Quotation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Quotation{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	total := in.DeliveryCost
	for _, l := range in.Lines {
		total += l.LineTotal
	}
	q, err := scanQuotation(tx.QueryRow(ctx, `
		INSERT INTO quotations (created_by, client_name, delivery_cost, final_price, created_at, updated_at)
		VALUES (NULLIF($1, '')::uuid, $2, $3, $4, $5, $5)
		RETURNING `+quotationColumns, in.CreatedBy, in.ClientName, in.DeliveryCost, total, in.CreatedAt))
	if err != nil {
		return Quotation{}, fmt.Errorf("insert quotation: %w", err)
	}
	for i, l := range in.Lines {
		if _, err := tx.Exec(ctx, `
			INSERT INTO quotation_lines (quotation_id, position, recipe_id, recipe_name, photo_url, sold_as, quantity,
			                             unit_price, line_total, margin_pct, indirect_cost_pct, labor_cost_pct, extra_charge)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			q.ID, i+1, l.RecipeID, l.RecipeName, l.PhotoURL, l.SoldAs, l.Quantity,
			l.UnitPrice, l.LineTotal, l.MarginPct, l.IndirectCostPct, l.LaborCostPct, l.ExtraCharge); err != nil {
			return Quotation{}, fmt.Errorf("insert line %d: %w", i+1, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Quotation{}, fmt.Errorf("commit: %w", err)
	}
	return r.Get(ctx, q.ID)
}

func (r *QuotationRepository) loadLines(ctx context.Context, quotations []Quotation) error {
	if len(quotations) == 0 {
		return nil
	}
	ids := make([]string, len(quotations))
	index := map[string]*Quotation{}
	for i := range quotations {
		ids[i] = quotations[i].ID
		index[quotations[i].ID] = &quotations[i]
	}
	rows, err := r.db.Query(ctx, `
		SELECT quotation_id, id, position, recipe_id, recipe_name, photo_url, sold_as, quantity, unit_price, line_total,
		       margin_pct, indirect_cost_pct, labor_cost_pct, extra_charge
		FROM quotation_lines WHERE quotation_id = ANY($1::uuid[]) ORDER BY quotation_id, position`, ids)
	if err != nil {
		return fmt.Errorf("list lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var qid string
		var l Line
		if err := rows.Scan(&qid, &l.ID, &l.Position, &l.RecipeID, &l.RecipeName, &l.PhotoURL, &l.SoldAs, &l.Quantity,
			&l.UnitPrice, &l.LineTotal, &l.MarginPct, &l.IndirectCostPct, &l.LaborCostPct, &l.ExtraCharge); err != nil {
			return fmt.Errorf("scan line: %w", err)
		}
		index[qid].Lines = append(index[qid].Lines, l)
	}
	return rows.Err()
}

// Get returns one quotation with its lines, or ErrNotFound (also for an id that is not a UUID).
func (r *QuotationRepository) Get(ctx context.Context, id string) (Quotation, error) {
	q, err := scanQuotation(r.db.QueryRow(ctx, `SELECT `+quotationColumns+` FROM quotations WHERE id = $1`, id))
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02") {
		return Quotation{}, errNotFound
	}
	if err != nil {
		return Quotation{}, fmt.Errorf("get quotation: %w", err)
	}
	list := []Quotation{q}
	if err := r.loadLines(ctx, list); err != nil {
		return Quotation{}, err
	}
	return list[0], nil
}

func (r *QuotationRepository) List(ctx context.Context, status string) ([]Quotation, error) {
	query := `SELECT ` + quotationColumns + ` FROM quotations WHERE TRUE`
	args := []any{}
	if status != "" {
		query += ` AND status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list quotations: %w", err)
	}
	defer rows.Close()
	result := []Quotation{}
	for rows.Next() {
		q, err := scanQuotation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan quotation: %w", err)
		}
		result = append(result, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read quotations: %w", err)
	}
	if err := r.loadLines(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *QuotationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	tag, err := r.db.Exec(ctx, `UPDATE quotations SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return errNotFound
	}
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
