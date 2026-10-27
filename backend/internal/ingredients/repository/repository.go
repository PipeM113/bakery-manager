package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/PipeM113/bakery-manager/internal/shared/clock"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDuplicate is returned when another ingredient has the same name, brand and supplier.
var ErrDuplicate = errors.New("ingredient with the same name, brand and supplier already exists")

// EquivalenceNeededError says that a recipe would stop working with the unit or the equivalences
// the ingredient is being given: it uses the ingredient in another family of units.
type EquivalenceNeededError struct {
	Recipe string
	Unit   string // the unit the recipe uses
	Err    *kernel.EquivalenceError
}

func (e *EquivalenceNeededError) Error() string {
	return fmt.Sprintf("la receta %q usa este insumo en %s: %v", e.Recipe, e.Unit, e.Err)
}

func isDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

const columns = `id, kind, name, default_unit, package_size, package_price, price_per_unit,
		stock_quantity, alert_threshold, brand, supplier, grams_per_liter, grams_per_unit, created_at, updated_at`

type scanner interface{ Scan(dest ...any) error }

func scan(row scanner, i *Ingredient) error {
	return row.Scan(&i.ID, &i.Kind, &i.Name, &i.DefaultUnit, &i.PackageSize, &i.PackagePrice, &i.PricePerUnit,
		&i.StockQuantity, &i.AlertThreshold, &i.Brand, &i.Supplier, &i.GramsPerLiter, &i.GramsPerUnit,
		&i.CreatedAt, &i.UpdatedAt)
}

// Kinds of ingredient row.
const (
	KindIngredient = "ingredient"
	KindSupply     = "supply"
)

// ValidKind reports whether k is one of the two kinds.
func ValidKind(k string) bool { return k == KindIngredient || k == KindSupply }

type Ingredient struct {
	ID             string    `json:"id"`
	Kind           string    `json:"kind"`
	Name           string    `json:"name"`
	DefaultUnit    string    `json:"default_unit"`
	PackageSize    float64   `json:"package_size"`
	PackagePrice   float64   `json:"package_price"`
	PricePerUnit   float64   `json:"price_per_unit"`
	StockQuantity  float64   `json:"stock_quantity"`
	AlertThreshold float64   `json:"alert_threshold"`
	Brand          string    `json:"brand"`
	Supplier       string    `json:"supplier"`
	GramsPerLiter  float64   `json:"grams_per_liter"`
	GramsPerUnit   float64   `json:"grams_per_unit"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PriceHistory struct {
	ID            string    `json:"id"`
	IngredientID  string    `json:"ingredient_id"`
	PricePerUnit  float64   `json:"price_per_unit"`
	Unit          string    `json:"unit"`
	EffectiveDate time.Time `json:"effective_date"`
}

type IngredientRepository struct {
	db  *pgxpool.Pool
	clk clock.Clock
}

func NewIngredientRepository(db *pgxpool.Pool, clk clock.Clock) *IngredientRepository {
	return &IngredientRepository{db: db, clk: clk}
}

func calcPricePerUnit(packagePrice, packageSize float64) float64 {
	if packageSize <= 0 {
		return 0
	}
	return packagePrice / packageSize
}

func (r *IngredientRepository) Create(ctx context.Context, i Ingredient) (Ingredient, error) {
	i.PricePerUnit = calcPricePerUnit(i.PackagePrice, i.PackageSize)
	if i.Kind == "" {
		i.Kind = KindIngredient
	}

	err := scan(r.db.QueryRow(ctx, `
		INSERT INTO ingredients
		  (name, default_unit, package_size, package_price, price_per_unit,
		   stock_quantity, alert_threshold, brand, kind, supplier, grams_per_liter, grams_per_unit)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING `+columns,
		i.Name, i.DefaultUnit, i.PackageSize, i.PackagePrice, i.PricePerUnit,
		i.StockQuantity, i.AlertThreshold, i.Brand, i.Kind, i.Supplier, i.GramsPerLiter, i.GramsPerUnit,
	), &i)
	if isDuplicate(err) {
		return Ingredient{}, ErrDuplicate
	}
	if err != nil {
		return Ingredient{}, fmt.Errorf("create ingredient: %w", err)
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO ingredient_price_history (ingredient_id, price_per_unit, unit, effective_date)
		 VALUES ($1, $2, $3, $4::date)`,
		i.ID, i.PricePerUnit, i.DefaultUnit, r.clk.Today(),
	)
	if err != nil {
		return Ingredient{}, fmt.Errorf("create price history: %w", err)
	}

	return i, nil
}

// GetAll lists the ingredient rows, optionally only one kind ("" means both), by name.
func (r *IngredientRepository) GetAll(ctx context.Context, kind string) ([]Ingredient, error) {
	rows, err := r.db.Query(ctx, `SELECT `+columns+`
		FROM ingredients
		WHERE ($1 = '' OR kind = $1)`, kind)
	if err != nil {
		return nil, fmt.Errorf("get all ingredients: %w", err)
	}
	defer rows.Close()

	var ingredients []Ingredient
	for rows.Next() {
		var i Ingredient
		if err := scan(rows, &i); err != nil {
			return nil, fmt.Errorf("scan ingredient: %w", err)
		}
		ingredients = append(ingredients, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read ingredients: %w", err)
	}
	// Alphabetical without case or accents (the database collation does not do that), then by brand and supplier.
	sort.SliceStable(ingredients, func(a, b int) bool {
		x, y := ingredients[a], ingredients[b]
		if kx, ky := kernel.SortKey(x.Name), kernel.SortKey(y.Name); kx != ky {
			return kx < ky
		}
		if kx, ky := kernel.SortKey(x.Brand), kernel.SortKey(y.Brand); kx != ky {
			return kx < ky
		}
		return kernel.SortKey(x.Supplier) < kernel.SortKey(y.Supplier)
	})
	return ingredients, nil
}

func (r *IngredientRepository) GetByID(ctx context.Context, id string) (Ingredient, error) {
	var i Ingredient
	if err := scan(r.db.QueryRow(ctx, `SELECT `+columns+` FROM ingredients WHERE id = $1`, id), &i); err != nil {
		return Ingredient{}, fmt.Errorf("get ingredient: %w", err)
	}
	return i, nil
}

// Update edits an ingredient. A change of the stock leaves a "correccion" in the stock
// adjustments, so no change of the stock goes without a record.
func (r *IngredientRepository) Update(ctx context.Context, i Ingredient, userID string) (Ingredient, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Ingredient{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldPrice, oldStock float64
	err = tx.QueryRow(ctx,
		`SELECT price_per_unit, stock_quantity FROM ingredients WHERE id = $1 FOR UPDATE`, i.ID,
	).Scan(&oldPrice, &oldStock)
	if err != nil {
		return Ingredient{}, fmt.Errorf("get old price: %w", err)
	}

	i.PricePerUnit = calcPricePerUnit(i.PackagePrice, i.PackageSize)

	if err := r.checkRecipes(ctx, tx, i); err != nil {
		return Ingredient{}, err
	}

	err = scan(tx.QueryRow(ctx, `
		UPDATE ingredients
		SET name=$1, default_unit=$2, package_size=$3, package_price=$4,
		    price_per_unit=$5, stock_quantity=$6, alert_threshold=$7,
		    brand=$8, kind=COALESCE(NULLIF($10, ''), kind), supplier=$11,
		    grams_per_liter=$12, grams_per_unit=$13, updated_at=NOW()
		WHERE id=$9
		RETURNING `+columns,
		i.Name, i.DefaultUnit, i.PackageSize, i.PackagePrice, i.PricePerUnit,
		i.StockQuantity, i.AlertThreshold, i.Brand, i.ID, i.Kind, i.Supplier, i.GramsPerLiter, i.GramsPerUnit,
	), &i)
	if isDuplicate(err) {
		return Ingredient{}, ErrDuplicate
	}
	if err != nil {
		return Ingredient{}, fmt.Errorf("update ingredient: %w", err)
	}

	if oldPrice != i.PricePerUnit {
		_, err = tx.Exec(ctx,
			`INSERT INTO ingredient_price_history (ingredient_id, price_per_unit, unit, effective_date)
			 VALUES ($1, $2, $3, $4::date)`,
			i.ID, i.PricePerUnit, i.DefaultUnit, r.clk.Today(),
		)
		if err != nil {
			return Ingredient{}, fmt.Errorf("create price history: %w", err)
		}
	}

	if before, after := math.Round(oldStock*1000)/1000, math.Round(i.StockQuantity*1000)/1000; before != after {
		_, err = tx.Exec(ctx, `
			INSERT INTO stock_adjustments (created_by, ingredient_id, reason, quantity_delta, unit, stock_before, stock_after, adjusted_on)
			VALUES ($1, $2, 'correccion', $3, $4, $5, $6, $7::date)`,
			userID, i.ID, math.Round((after-before)*1000)/1000, i.DefaultUnit, before, after, r.clk.Today(),
		)
		if err != nil {
			return Ingredient{}, fmt.Errorf("record stock correction: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Ingredient{}, fmt.Errorf("commit tx: %w", err)
	}
	return i, nil
}

func (r *IngredientRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM ingredients WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ingredient: %w", err)
	}
	return nil
}

func (r *IngredientRepository) GetPriceHistory(ctx context.Context, id string) ([]PriceHistory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, ingredient_id, price_per_unit, unit, effective_date
		FROM ingredient_price_history
		WHERE ingredient_id = $1
		ORDER BY effective_date DESC`, id)
	if err != nil {
		return nil, fmt.Errorf("get price history: %w", err)
	}
	defer rows.Close()

	var history []PriceHistory
	for rows.Next() {
		var h PriceHistory
		err := rows.Scan(&h.ID, &h.IngredientID, &h.PricePerUnit, &h.Unit, &h.EffectiveDate)
		if err != nil {
			return nil, fmt.Errorf("scan price history: %w", err)
		}
		history = append(history, h)
	}
	return history, nil
}

// checkRecipes makes sure that every recipe that uses the ingredient can still be converted to its unit
// with the unit and the equivalences it is being given. Only a missing equivalence is a refusal.
func (r *IngredientRepository) checkRecipes(ctx context.Context, tx pgx.Tx, i Ingredient) error {
	rows, err := tx.Query(ctx, `
		SELECT r.name, ri.quantity, ri.unit
		FROM recipe_ingredients ri JOIN recipes r ON r.id = ri.recipe_id
		WHERE ri.ingredient_id = $1 ORDER BY r.name`, i.ID)
	if err != nil {
		return fmt.Errorf("get recipes of the ingredient: %w", err)
	}
	defer rows.Close()
	eq := kernel.Equivalence{GramsPerLiter: i.GramsPerLiter, GramsPerUnit: i.GramsPerUnit}
	for rows.Next() {
		var name, unit string
		var quantity float64
		if err := rows.Scan(&name, &quantity, &unit); err != nil {
			return fmt.Errorf("scan recipe line: %w", err)
		}
		_, err := kernel.Convert(quantity, unit, i.DefaultUnit, eq)
		var need *kernel.EquivalenceError
		if errors.As(err, &need) {
			return &EquivalenceNeededError{Recipe: name, Unit: unit, Err: need}
		}
	}
	return rows.Err()
}
