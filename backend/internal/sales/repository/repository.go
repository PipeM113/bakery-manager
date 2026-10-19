package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/PipeM113/bakery-manager/internal/shared/kernel"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	OrderID      *string          `json:"order_id"`
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

// ErrNotFound means the sale or the order does not exist (or is somebody else's).
var ErrNotFound = errors.New("not found")

// ErrPartOfOrder means a sale cannot be cancelled alone because it is a line of an order.
var ErrPartOfOrder = errors.New("la venta es parte de un pedido")

// LineInput is one product of a sale or of an order.
type LineInput struct {
	RecipeID     string
	QuantitySold int
	UnitPrice    int64  // normal price of one unit (a recipe or a portion)
	SoldAs       string // "recipe" or "portion"
}

type need struct {
	ingredientID string
	name         string
	used         float64 // in the unit of the ingredient
	unit         string  // the unit of the ingredient
	pricePerUnit float64
}

type db interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// needsOf converts what the recipe asks for, for the units sold, to the unit each ingredient is
// kept in. A sale of portions takes the share of the recipe that those portions are.
func needsOf(ctx context.Context, tx db, line LineInput) ([]need, error) {
	var yield float64
	if err := tx.QueryRow(ctx, `SELECT yield FROM recipes WHERE id = $1`, line.RecipeID).Scan(&yield); err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02") {
			return nil, &ValidationError{"la receta no existe"}
		}
		return nil, fmt.Errorf("fetch recipe: %w", err)
	}
	share := 1.0
	if line.SoldAs == "portion" {
		if yield <= 0 {
			return nil, &ValidationError{"la receta no tiene rendimiento: no se puede vender por porción"}
		}
		share = 1 / yield
	}

	rows, err := tx.Query(ctx, `
		SELECT ri.ingredient_id, i.name, ri.quantity, ri.unit, i.default_unit, i.price_per_unit
		FROM recipe_ingredients ri
		JOIN ingredients i ON i.id = ri.ingredient_id
		WHERE ri.recipe_id = $1`, line.RecipeID)
	if err != nil {
		return nil, fmt.Errorf("fetch recipe ingredients: %w", err)
	}
	defer rows.Close()
	var needs []need
	for rows.Next() {
		var n need
		var quantity float64
		var recipeUnit string
		if err := rows.Scan(&n.ingredientID, &n.name, &quantity, &recipeUnit, &n.unit, &n.pricePerUnit); err != nil {
			return nil, fmt.Errorf("scan ingredient: %w", err)
		}
		converted, err := kernel.ConvertToBase(quantity*float64(line.QuantitySold)*share, recipeUnit, n.unit)
		if err != nil {
			return nil, &ValidationError{fmt.Sprintf("%s: %v", n.name, err)}
		}
		n.used = round(converted, 4)
		needs = append(needs, n)
	}
	return needs, rows.Err()
}

// takeLines records each line as a sale (with the money it brought in) and takes its ingredients
// out of the stock. Every ingredient of every line is locked first, in a fixed order, so sales and
// orders at the same time wait for each other instead of reading the same stock. Without enough
// stock the lines are still registered: the stock stops at 0 and the shortfalls say what was missing.
func takeLines(ctx context.Context, tx db, userID string, orderID *string, saleDate, notes string, lines []LineInput, lineTotals []int64) ([]Sale, []Shortfall, error) {
	allNeeds := make([][]need, len(lines))
	for i, line := range lines {
		n, err := needsOf(ctx, tx, line)
		if err != nil {
			return nil, nil, err
		}
		allNeeds[i] = n
	}

	// Lock every ingredient once, in a fixed order.
	ids := map[string]bool{}
	for _, ns := range allNeeds {
		for _, n := range ns {
			ids[n.ingredientID] = true
		}
	}
	order := make([]string, 0, len(ids))
	for id := range ids {
		order = append(order, id)
	}
	sort.Strings(order)
	available := map[string]float64{}
	for _, id := range order {
		var stock float64
		if err := tx.QueryRow(ctx, `SELECT stock_quantity FROM ingredients WHERE id = $1 FOR UPDATE`, id).Scan(&stock); err != nil {
			return nil, nil, fmt.Errorf("lock stock: %w", err)
		}
		available[id] = stock
	}

	missing := map[string]*Shortfall{}
	var missingOrder []string
	sales := make([]Sale, 0, len(lines))
	for i, line := range lines {
		var sale Sale
		err := tx.QueryRow(ctx, `
			INSERT INTO sales (user_id, recipe_id, quantity_sold, unit_price, sold_as, sale_date, notes, order_id, line_total, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, clock_timestamp())
			RETURNING id, user_id, recipe_id, quantity_sold, unit_price, line_total, sold_as, sale_date::text, notes, created_at`,
			userID, line.RecipeID, line.QuantitySold, line.UnitPrice, line.SoldAs, saleDate, notes, orderID, lineTotals[i],
		).Scan(&sale.ID, &sale.UserID, &sale.RecipeID, &sale.QuantitySold, &sale.UnitPrice, &sale.TotalPrice,
			&sale.SoldAs, &sale.SaleDate, &sale.Notes, &sale.CreatedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("insert sale: %w", err)
		}
		sale.OrderID = orderID
		sale.StockShortfalls = []Shortfall{}
		tx.QueryRow(ctx, `SELECT name FROM recipes WHERE id=$1`, line.RecipeID).Scan(&sale.RecipeName)

		for _, n := range allNeeds[i] {
			needed := round(n.used, 3)
			deducted := math.Min(available[n.ingredientID], needed)
			available[n.ingredientID] -= deducted
			if short := round(needed-deducted, 3); short > 0 {
				if m, ok := missing[n.ingredientID]; ok {
					m.Missing = round(m.Missing+short, 3)
				} else {
					missing[n.ingredientID] = &Shortfall{Ingredient: n.name, Missing: short, Unit: n.unit}
					missingOrder = append(missingOrder, n.ingredientID)
				}
			}
			var si SaleIngredient
			err := tx.QueryRow(ctx, `
				INSERT INTO sale_ingredients (sale_id, ingredient_id, quantity_used, unit, price_at_time, stock_deducted)
				VALUES ($1,$2,$3,$4,$5,$6)
				RETURNING id, sale_id, ingredient_id, quantity_used, unit, price_at_time`,
				sale.ID, n.ingredientID, n.used, n.unit, n.pricePerUnit, deducted,
			).Scan(&si.ID, &si.SaleID, &si.IngredientID, &si.QuantityUsed, &si.Unit, &si.PriceAtTime)
			if err != nil {
				return nil, nil, fmt.Errorf("insert sale_ingredient: %w", err)
			}
			si.Name = n.name
			sale.Ingredients = append(sale.Ingredients, si)
		}
		sales = append(sales, sale)
	}

	for _, id := range order {
		if _, err := tx.Exec(ctx, `UPDATE ingredients SET stock_quantity = $1, updated_at = NOW() WHERE id = $2`, round(available[id], 3), id); err != nil {
			return nil, nil, fmt.Errorf("update stock: %w", err)
		}
	}
	shortfalls := make([]Shortfall, 0, len(missingOrder))
	for _, id := range missingOrder {
		shortfalls = append(shortfalls, *missing[id])
	}
	return sales, shortfalls, nil
}

// RegisterSale records a sale of one product and takes its ingredients out of the stock (see takeLines).
func (r *SaleRepository) RegisterSale(ctx context.Context, input RegisterSaleInput) (Sale, error) {
	if input.SaleDate == "" {
		return Sale{}, fmt.Errorf("a sale needs a date")
	}
	soldAs := input.SoldAs
	if soldAs == "" {
		soldAs = "recipe"
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Sale{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	line := LineInput{RecipeID: input.RecipeID, QuantitySold: input.QuantitySold, UnitPrice: input.UnitPrice, SoldAs: soldAs}
	sales, shortfalls, err := takeLines(ctx, tx, input.UserID, nil, input.SaleDate, input.Notes,
		[]LineInput{line}, []int64{int64(input.QuantitySold) * input.UnitPrice})
	if err != nil {
		return Sale{}, err
	}
	sale := sales[0]
	sale.StockShortfalls = shortfalls
	if err := tx.Commit(ctx); err != nil {
		return Sale{}, fmt.Errorf("commit tx: %w", err)
	}
	return sale, nil
}

// GetAll returns sales for a user optionally filtered by date range.
func (r *SaleRepository) GetAll(ctx context.Context, userID, from, to string) ([]Sale, error) {
	query := `
		SELECT s.id, s.user_id, s.recipe_id, rec.name, s.quantity_sold, s.unit_price,
		       s.line_total AS total_price,
		       s.sold_as, s.order_id, s.sale_date::text, s.notes, s.created_at
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
			&s.QuantitySold, &s.UnitPrice, &s.TotalPrice, &s.SoldAs, &s.OrderID, &s.SaleDate, &s.Notes, &s.CreatedAt); err != nil {
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

	var orderID *string
	err = tx.QueryRow(ctx, `SELECT order_id FROM sales WHERE id = $1 AND user_id = $2 FOR UPDATE`, id, userID).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("sale not found")
	}
	if err != nil {
		return fmt.Errorf("fetch sale: %w", err)
	}
	if orderID != nil {
		return ErrPartOfOrder
	}

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
