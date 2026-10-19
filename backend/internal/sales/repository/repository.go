package repository

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/PipeM113/bakery-manager/internal/shared/kernel"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidationError is a sale the recipe cannot support (units that do not convert, a recipe that
// yields nothing); the owner can fix it.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// Shortfall is what a sale needed and the stock did not have.
type Shortfall struct {
	Ingredient string  `json:"ingredient"`
	Missing    float64 `json:"missing"`
	Unit       string  `json:"unit"`
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
	UnitPrice    int64            `json:"unit_price"`
	TotalPrice   int64            `json:"total_price"`
	SaleDate     string           `json:"sale_date"`
	SoldAs       string           `json:"sold_as"`
	Notes        string           `json:"notes"`
	Ingredients  []SaleIngredient `json:"ingredients"`
	// StockShortfalls lists what the stock could not give; empty when it was enough.
	StockShortfalls []Shortfall `json:"stock_shortfalls"`
	CreatedAt       time.Time   `json:"created_at"`
}

type RegisterSaleInput struct {
	UserID       string
	RecipeID     string
	QuantitySold int
	UnitPrice    int64
	SoldAs       string // "recipe" or "portion"
	SaleDate     string
	Notes        string
}

type SaleRepository struct {
	db *pgxpool.Pool
}

func NewSaleRepository(db *pgxpool.Pool) *SaleRepository {
	return &SaleRepository{db: db}
}

func round(x float64, decimals int) float64 {
	p := math.Pow10(decimals)
	return math.Round(x*p) / p
}

// RegisterSale records a sale and takes its ingredients out of the stock. Each quantity of the
// recipe is converted to the unit of the ingredient, and a sale of portions takes the share of
// the recipe that those portions are. A sale without enough stock is registered anyway: the stock
// stops at 0 and the sale lists what was missing.
func (r *SaleRepository) RegisterSale(ctx context.Context, input RegisterSaleInput) (Sale, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Sale{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	soldAs := input.SoldAs
	if soldAs == "" {
		soldAs = "recipe"
	}
	// Share of the recipe each sold unit takes.
	share := 1.0
	if soldAs == "portion" {
		var yield float64
		if err := tx.QueryRow(ctx, `SELECT yield FROM recipes WHERE id = $1`, input.RecipeID).Scan(&yield); err != nil {
			return Sale{}, fmt.Errorf("fetch recipe yield: %w", err)
		}
		if yield <= 0 {
			return Sale{}, &ValidationError{"la receta no tiene rendimiento: no se puede vender por porción"}
		}
		share = 1 / yield
	}

	// The recipe's ingredients, with the unit the ingredient is kept in.
	type recipeIng struct {
		ingredientID string
		name         string
		used         float64 // in the unit of the ingredient
		unit         string  // the unit of the ingredient
		pricePerUnit float64
	}
	rows, err := tx.Query(ctx, `
		SELECT ri.ingredient_id, i.name, ri.quantity, ri.unit, i.default_unit, i.price_per_unit
		FROM recipe_ingredients ri
		JOIN ingredients i ON i.id = ri.ingredient_id
		WHERE ri.recipe_id = $1`, input.RecipeID)
	if err != nil {
		return Sale{}, fmt.Errorf("fetch recipe ingredients: %w", err)
	}
	var ings []recipeIng
	for rows.Next() {
		var ing recipeIng
		var quantity float64
		var recipeUnit string
		if err := rows.Scan(&ing.ingredientID, &ing.name, &quantity, &recipeUnit, &ing.unit, &ing.pricePerUnit); err != nil {
			rows.Close()
			return Sale{}, fmt.Errorf("scan ingredient: %w", err)
		}
		converted, err := kernel.ConvertToBase(quantity*float64(input.QuantitySold)*share, recipeUnit, ing.unit)
		if err != nil {
			rows.Close()
			return Sale{}, &ValidationError{fmt.Sprintf("%s: %v", ing.name, err)}
		}
		ing.used = round(converted, 4)
		ings = append(ings, ing)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Sale{}, fmt.Errorf("read recipe ingredients: %w", err)
	}

	saleDate := input.SaleDate
	if saleDate == "" {
		return Sale{}, fmt.Errorf("a sale needs a date")
	}
	var sale Sale
	err = tx.QueryRow(ctx, `
		INSERT INTO sales (user_id, recipe_id, quantity_sold, unit_price, sold_as, sale_date, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, user_id, recipe_id, quantity_sold, unit_price, sold_as, sale_date::text, notes, created_at`,
		input.UserID, input.RecipeID, input.QuantitySold, input.UnitPrice, soldAs, saleDate, input.Notes,
	).Scan(&sale.ID, &sale.UserID, &sale.RecipeID, &sale.QuantitySold, &sale.UnitPrice, &sale.SoldAs,
		&sale.SaleDate, &sale.Notes, &sale.CreatedAt)
	if err != nil {
		return Sale{}, fmt.Errorf("insert sale: %w", err)
	}
	sale.TotalPrice = int64(sale.QuantitySold) * sale.UnitPrice
	sale.StockShortfalls = []Shortfall{}

	// Take the stock one ingredient at a time, always in the same order, with the row locked: two
	// sales at once wait for each other instead of both reading the same stock.
	sort.SliceStable(ings, func(i, j int) bool { return ings[i].ingredientID < ings[j].ingredientID })
	for _, ing := range ings {
		var available float64
		if err := tx.QueryRow(ctx,
			`SELECT stock_quantity FROM ingredients WHERE id = $1 FOR UPDATE`, ing.ingredientID,
		).Scan(&available); err != nil {
			return Sale{}, fmt.Errorf("lock stock for %s: %w", ing.name, err)
		}
		need := round(ing.used, 3)
		deducted := math.Min(available, need)
		if _, err := tx.Exec(ctx, `
			UPDATE ingredients SET stock_quantity = stock_quantity - $1, updated_at = NOW()
			WHERE id = $2`, deducted, ing.ingredientID); err != nil {
			return Sale{}, fmt.Errorf("update stock for %s: %w", ing.name, err)
		}
		if missing := round(need-deducted, 3); missing > 0 {
			sale.StockShortfalls = append(sale.StockShortfalls, Shortfall{Ingredient: ing.name, Missing: missing, Unit: ing.unit})
		}

		var si SaleIngredient
		err = tx.QueryRow(ctx, `
			INSERT INTO sale_ingredients (sale_id, ingredient_id, quantity_used, unit, price_at_time, stock_deducted)
			VALUES ($1,$2,$3,$4,$5,$6)
			RETURNING id, sale_id, ingredient_id, quantity_used, unit, price_at_time`,
			sale.ID, ing.ingredientID, ing.used, ing.unit, ing.pricePerUnit, deducted,
		).Scan(&si.ID, &si.SaleID, &si.IngredientID, &si.QuantityUsed, &si.Unit, &si.PriceAtTime)
		if err != nil {
			return Sale{}, fmt.Errorf("insert sale_ingredient: %w", err)
		}
		si.Name = ing.name
		sale.Ingredients = append(sale.Ingredients, si)
	}

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
		       s.quantity_sold::bigint * s.unit_price AS total_price,
		       s.sold_as, s.sale_date::text, s.notes, s.created_at
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
			&s.QuantitySold, &s.UnitPrice, &s.TotalPrice, &s.SoldAs, &s.SaleDate, &s.Notes, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan sale: %w", err)
		}
		s.StockShortfalls = []Shortfall{}
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
		SELECT ingredient_id, stock_deducted FROM sale_ingredients WHERE sale_id = $1`, id)
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
