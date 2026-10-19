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

var (
	ErrNotFound  = errors.New("categoría no encontrada")
	ErrDuplicate = errors.New("ya existe una categoría con ese nombre")
	ErrInUse     = errors.New("la categoría tiene gastos")
)

type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type CategoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func (r *CategoryRepository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, created_at FROM expense_categories ORDER BY lower(name)`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read categories: %w", err)
	}
	return out, nil
}

func (r *CategoryRepository) Create(ctx context.Context, name string) (Category, error) {
	var c Category
	err := r.db.QueryRow(ctx,
		`INSERT INTO expense_categories (name) VALUES ($1) RETURNING id, name, created_at`, name,
	).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if pgCode(err) == "23505" {
		return Category{}, ErrDuplicate
	}
	if err != nil {
		return Category{}, fmt.Errorf("create category: %w", err)
	}
	return c, nil
}

func (r *CategoryRepository) Rename(ctx context.Context, id, name string) (Category, error) {
	var c Category
	err := r.db.QueryRow(ctx, `
		UPDATE expense_categories SET name = $1, updated_at = NOW() WHERE id = $2
		RETURNING id, name, created_at`, name, id,
	).Scan(&c.ID, &c.Name, &c.CreatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Category{}, ErrNotFound
	case pgCode(err) == "23505":
		return Category{}, ErrDuplicate
	case pgCode(err) == "22P02": // not a UUID
		return Category{}, ErrNotFound
	case err != nil:
		return Category{}, fmt.Errorf("rename category: %w", err)
	}
	return c, nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM expense_categories WHERE id = $1`, id)
	switch {
	case pgCode(err) == "23503":
		return ErrInUse
	case pgCode(err) == "22P02":
		return ErrNotFound
	case err != nil:
		return fmt.Errorf("delete category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
