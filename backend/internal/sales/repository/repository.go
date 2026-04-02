package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InsufficientStockError is returned when a sale would require more stock than available.
type InsufficientStockError struct {
	IngredientName string
	Available      float64
	Required       float64
	Unit           string
}

func (e *InsufficientStockError) Error() string {
	return fmt.Sprintf("stock insuficiente para %s. Disponible: %.3f %s, Requerido: %.3f %s",
		e.IngredientName, e.Available, e.Unit, e.Required, e.Unit)
}

type SaleIngredient struct {
	ID           string  `json:"id"`
	SaleID       string  `json:"sale_id"`
	IngredientID string  `json:"ingredient_id"`
	Name         string  `json:"name"`
	QuantityUsed float64 `json:"quantity_used"`
	Unit         string  `json:"unit"`
	PriceAtTime  float64 `json:"price_at_time"`
}

type Sale struct {
	ID           string           `json:"id"`
	UserID       string           `json:"user_id"`
	RecipeID     string           `json:"recipe_id"`
	RecipeName   string           `json:"recipe_name"`
	QuantitySold int              `json:"quantity_sold"`
	UnitPrice    float64          `json:"unit_price"`
	TotalPrice   float64          `json:"total_price"`
	SaleDate     string           `json:"sale_date"`
	Notes        string           `json:"notes"`
	Ingredients  []SaleIngredient `json:"ingredients"`
	CreatedAt    time.Time        `json:"created_at"`
}

type RegisterSaleInput struct {
	UserID       string
	RecipeID     string
	QuantitySold int
	UnitPrice    float64
	SaleDate     string
	Notes        string
}

type SaleRepository struct {
	db *pgxpool.Pool
}

func NewSaleRepository(db *pgxpool.Pool) *SaleRepository {
	return &SaleRepository{db: db}
}

// RegisterSale records a sale, decrements current_stock for each ingredient, and saves audit rows.
func (r *SaleRepository) RegisterSale(ctx context.Context, input RegisterSaleInput) (Sale, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Sale{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch recipe ingredients with current price
	type recipeIng struct {
		ingredientID string
		name         string
		quantity     float64
		unit         string
		pricePerUnit float64
	}
	rows, err := tx.Query(ctx, `
		SELECT ri.ingredient_id, i.name, ri.quantity, ri.unit, i.price_per_unit
		FROM recipe_ingredients ri
		JOIN ingredients i ON i.id = ri.ingredient_id
		WHERE ri.recipe_id = $1`, input.RecipeID)
	if err != nil {
		return Sale{}, fmt.Errorf("fetch recipe ingredients: %w", err)
	}
	var ings []recipeIng
	for rows.Next() {
		var ing recipeIng
		if err := rows.Scan(&ing.ingredientID, &ing.name, &ing.quantity, &ing.unit, &ing.pricePerUnit); err != nil {
			rows.Close()
			return Sale{}, fmt.Errorf("scan ingredient: %w", err)
		}
		ings = append(ings, ing)
	}
	rows.Close()

	// Validate stock before any writes
	for _, ing := range ings {
		qtyNeeded := ing.quantity * float64(input.QuantitySold)
		var available float64
		if err = tx.QueryRow(ctx,
			`SELECT stock_quantity FROM ingredients WHERE id = $1`, ing.ingredientID,
		).Scan(&available); err != nil {
			return Sale{}, fmt.Errorf("check stock for %s: %w", ing.name, err)
		}
		if available < qtyNeeded {
			return Sale{}, &InsufficientStockError{
				IngredientName: ing.name,
				Available:      available,
				Required:       qtyNeeded,
				Unit:           ing.unit,
			}
		}
	}

	// Insert sale row
	saleDate := input.SaleDate
	if saleDate == "" {
		saleDate = time.Now().Format("2006-01-02")
	}
	var sale Sale
	err = tx.QueryRow(ctx, `
		INSERT INTO sales (user_id, recipe_id, quantity_sold, unit_price, sale_date, notes)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, user_id, recipe_id, quantity_sold, unit_price, sale_date::text, notes, created_at`,
		input.UserID, input.RecipeID, input.QuantitySold, input.UnitPrice, saleDate, input.Notes,
	).Scan(&sale.ID, &sale.UserID, &sale.RecipeID, &sale.QuantitySold, &sale.UnitPrice,
		&sale.SaleDate, &sale.Notes, &sale.CreatedAt)
	if err != nil {
		return Sale{}, fmt.Errorf("insert sale: %w", err)
	}
	sale.TotalPrice = float64(sale.QuantitySold) * sale.UnitPrice

	// For each ingredient: decrement stock, insert audit row
	for _, ing := range ings {
		qtyUsed := ing.quantity * float64(input.QuantitySold)

		_, err = tx.Exec(ctx, `
			UPDATE ingredients
			SET stock_quantity = stock_quantity - $1, updated_at = NOW()
			WHERE id = $2`, qtyUsed, ing.ingredientID)
		if err != nil {
			return Sale{}, fmt.Errorf("update stock for %s: %w", ing.name, err)
		}

		var si SaleIngredient
		err = tx.QueryRow(ctx, `
			INSERT INTO sale_ingredients (sale_id, ingredient_id, quantity_used, unit, price_at_time)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id, sale_id, ingredient_id, quantity_used, unit, price_at_time`,
			sale.ID, ing.ingredientID, qtyUsed, ing.unit, ing.pricePerUnit,
		).Scan(&si.ID, &si.SaleID, &si.IngredientID, &si.QuantityUsed, &si.Unit, &si.PriceAtTime)
		if err != nil {
			return Sale{}, fmt.Errorf("insert sale_ingredient: %w", err)
		}
		si.Name = ing.name
		sale.Ingredients = append(sale.Ingredients, si)
	}

	// Get recipe name
	tx.QueryRow(ctx, `SELECT name FROM recipes WHERE id=$1`, input.RecipeID).Scan(&sale.RecipeName)

	if err := tx.Commit(ctx); err != nil {
		return Sale{}, fmt.Errorf("commit tx: %w", err)
	}
	return sale, nil
}

// GetAll returns sales for a user optionally filtered by date range.
func (r *SaleRepository) GetAll(ctx context.Context, userID, from, to string) ([]Sale, error) {
	query := `
		SELECT s.id, s.user_id, s.recipe_id, rec.name, s.quantity_sold, s.unit_price,
		       s.quantity_sold * s.unit_price AS total_price,
		       s.sale_date::text, s.notes, s.created_at
		FROM sales s
		JOIN recipes rec ON rec.id = s.recipe_id
		WHERE s.user_id = $1`
	args := []any{userID}

	if from != "" {
		args = append(args, from)
		query += fmt.Sprintf(" AND s.sale_date >= $%d", len(args))
	}
	if to != "" {
		args = append(args, to)
		query += fmt.Sprintf(" AND s.sale_date <= $%d", len(args))
	}
	query += " ORDER BY s.sale_date DESC, s.created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get all sales: %w", err)
	}
	defer rows.Close()

	var sales []Sale
	for rows.Next() {
		var s Sale
		if err := rows.Scan(&s.ID, &s.UserID, &s.RecipeID, &s.RecipeName,
			&s.QuantitySold, &s.UnitPrice, &s.TotalPrice, &s.SaleDate, &s.Notes, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan sale: %w", err)
		}
		sales = append(sales, s)
	}
	return sales, nil
}

// DeleteSale removes a sale and reverts ingredient stock.
func (r *SaleRepository) DeleteSale(ctx context.Context, id, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch audit rows before deleting
	rows, err := tx.Query(ctx, `
		SELECT ingredient_id, quantity_used FROM sale_ingredients WHERE sale_id = $1`, id)
	if err != nil {
		return fmt.Errorf("fetch sale_ingredients: %w", err)
	}
	type stockRevert struct {
		ingredientID string
		qty          float64
	}
	var reverts []stockRevert
	for rows.Next() {
		var sr stockRevert
		if err := rows.Scan(&sr.ingredientID, &sr.qty); err != nil {
			rows.Close()
			return fmt.Errorf("scan sale_ingredient: %w", err)
		}
		reverts = append(reverts, sr)
	}
	rows.Close()

	// Revert stock
	for _, sr := range reverts {
		_, err = tx.Exec(ctx, `
			UPDATE ingredients
			SET stock_quantity = stock_quantity + $1, updated_at = NOW()
			WHERE id = $2`, sr.qty, sr.ingredientID)
		if err != nil {
			return fmt.Errorf("revert stock: %w", err)
		}
	}

	// Delete sale (cascades to sale_ingredients)
	tag, err := tx.Exec(ctx, `DELETE FROM sales WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete sale: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("sale not found")
	}

	return tx.Commit(ctx)
}
