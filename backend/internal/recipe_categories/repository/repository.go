// Package repository stores the types of recipe ("Tortas bizcocho", "Hojarasca"...). A recipe may have one;
// deleting a type leaves its recipes without type (docs/recetario.md).
package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("tipo no encontrado")
	ErrDuplicate = errors.New("ya existe un tipo con ese nombre")
)

type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	RecipeCount int       `json:"recipe_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// List gives the types in alphabetical order without case or accents, with how many recipes each has.
func (r *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.name, c.created_at, count(r.id)
		FROM recipe_categories c LEFT JOIN recipes r ON r.category_id = c.id
		GROUP BY c.id`)
	if err != nil {
		return nil, fmt.Errorf("list recipe types: %w", err)
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt, &c.RecipeCount); err != nil {
			return nil, fmt.Errorf("scan recipe type: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read recipe types: %w", err)
	}
	sort.SliceStable(out, func(a, b int) bool {
		if ka, kb := kernel.SortKey(out[a].Name), kernel.SortKey(out[b].Name); ka != kb {
			return ka < kb
		}
		return out[a].Name < out[b].Name
	})
	return out, nil
}

// taken says whether another type already has this name, without case or accents.
func (r *Repository) taken(ctx context.Context, name, exceptID string) (bool, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name FROM recipe_categories`)
	if err != nil {
		return false, fmt.Errorf("get recipe types: %w", err)
	}
	defer rows.Close()
	key := kernel.SortKey(name)
	for rows.Next() {
		var id, other string
		if err := rows.Scan(&id, &other); err != nil {
			return false, fmt.Errorf("scan recipe type: %w", err)
		}
		if id != exceptID && kernel.SortKey(other) == key {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (r *Repository) get(ctx context.Context, id string) (Category, error) {
	var c Category
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.name, c.created_at, (SELECT count(*) FROM recipes WHERE category_id = c.id)
		FROM recipe_categories c WHERE c.id = $1`, id).Scan(&c.ID, &c.Name, &c.CreatedAt, &c.RecipeCount)
	switch {
	case errors.Is(err, pgx.ErrNoRows), pgCode(err) == "22P02":
		return Category{}, ErrNotFound
	case err != nil:
		return Category{}, fmt.Errorf("get recipe type: %w", err)
	}
	return c, nil
}

func (r *Repository) Create(ctx context.Context, name string) (Category, error) {
	if dup, err := r.taken(ctx, name, ""); err != nil {
		return Category{}, err
	} else if dup {
		return Category{}, ErrDuplicate
	}
	var id string
	err := r.db.QueryRow(ctx, `INSERT INTO recipe_categories (name) VALUES ($1) RETURNING id`, name).Scan(&id)
	if pgCode(err) == "23505" {
		return Category{}, ErrDuplicate
	}
	if err != nil {
		return Category{}, fmt.Errorf("create recipe type: %w", err)
	}
	return r.get(ctx, id)
}

func (r *Repository) Rename(ctx context.Context, id, name string) (Category, error) {
	if _, err := r.get(ctx, id); err != nil {
		return Category{}, err
	}
	if dup, err := r.taken(ctx, name, id); err != nil {
		return Category{}, err
	} else if dup {
		return Category{}, ErrDuplicate
	}
	_, err := r.db.Exec(ctx, `UPDATE recipe_categories SET name = $1, updated_at = NOW() WHERE id = $2`, name, id)
	if pgCode(err) == "23505" {
		return Category{}, ErrDuplicate
	}
	if err != nil {
		return Category{}, fmt.Errorf("rename recipe type: %w", err)
	}
	return r.get(ctx, id)
}

// Delete removes the type; its recipes stay, without type (ON DELETE SET NULL).
func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM recipe_categories WHERE id = $1`, id)
	switch {
	case pgCode(err) == "22P02":
		return ErrNotFound
	case err != nil:
		return fmt.Errorf("delete recipe type: %w", err)
	case tag.RowsAffected() == 0:
		return ErrNotFound
	}
	return nil
}
