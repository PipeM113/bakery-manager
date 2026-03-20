package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Ingredient struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	DefaultUnit    string    `json:"default_unit"`
	PackageSize    float64   `json:"package_size"`
	PackagePrice   float64   `json:"package_price"`
	PricePerUnit   float64   `json:"price_per_unit"`
	StockQuantity  float64   `json:"stock_quantity"`
	AlertThreshold float64   `json:"alert_threshold"`
	Supplier       string    `json:"supplier"`
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
	db *pgxpool.Pool
}

func NewIngredientRepository(db *pgxpool.Pool) *IngredientRepository {
	return &IngredientRepository{db: db}
}

func calcPricePerUnit(packagePrice, packageSize float64) float64 {
	if packageSize <= 0 {
		return 0
	}
	return packagePrice / packageSize
}

func (r *IngredientRepository) Create(ctx context.Context, i Ingredient) (Ingredient, error) {
	i.PricePerUnit = calcPricePerUnit(i.PackagePrice, i.PackageSize)

	err := r.db.QueryRow(ctx, `
		INSERT INTO ingredients
		  (name, default_unit, package_size, package_price, price_per_unit,
		   stock_quantity, alert_threshold, supplier)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, name, default_unit, package_size, package_price, price_per_unit,
		          stock_quantity, alert_threshold, supplier, created_at, updated_at`,
		i.Name, i.DefaultUnit, i.PackageSize, i.PackagePrice, i.PricePerUnit,
		i.StockQuantity, i.AlertThreshold, i.Supplier,
	).Scan(
		&i.ID, &i.Name, &i.DefaultUnit, &i.PackageSize, &i.PackagePrice, &i.PricePerUnit,
		&i.StockQuantity, &i.AlertThreshold, &i.Supplier,
		&i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return Ingredient{}, fmt.Errorf("create ingredient: %w", err)
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO ingredient_price_history (ingredient_id, price_per_unit, unit)
		 VALUES ($1, $2, $3)`,
		i.ID, i.PricePerUnit, i.DefaultUnit,
	)
	if err != nil {
		return Ingredient{}, fmt.Errorf("create price history: %w", err)
	}

	return i, nil
}

func (r *IngredientRepository) GetAll(ctx context.Context) ([]Ingredient, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, default_unit, package_size, package_price, price_per_unit,
		       stock_quantity, alert_threshold, supplier, created_at, updated_at
		FROM ingredients ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("get all ingredients: %w", err)
	}
	defer rows.Close()

	var ingredients []Ingredient
	for rows.Next() {
		var i Ingredient
		err := rows.Scan(
			&i.ID, &i.Name, &i.DefaultUnit, &i.PackageSize, &i.PackagePrice, &i.PricePerUnit,
			&i.StockQuantity, &i.AlertThreshold, &i.Supplier,
			&i.CreatedAt, &i.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ingredient: %w", err)
		}
		ingredients = append(ingredients, i)
	}
	return ingredients, nil
}

func (r *IngredientRepository) GetByID(ctx context.Context, id string) (Ingredient, error) {
	var i Ingredient
	err := r.db.QueryRow(ctx, `
		SELECT id, name, default_unit, package_size, package_price, price_per_unit,
		       stock_quantity, alert_threshold, supplier, created_at, updated_at
		FROM ingredients WHERE id = $1`, id,
	).Scan(
		&i.ID, &i.Name, &i.DefaultUnit, &i.PackageSize, &i.PackagePrice, &i.PricePerUnit,
		&i.StockQuantity, &i.AlertThreshold, &i.Supplier,
		&i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return Ingredient{}, fmt.Errorf("get ingredient: %w", err)
	}
	return i, nil
}

func (r *IngredientRepository) Update(ctx context.Context, i Ingredient) (Ingredient, error) {
	var oldPrice float64
	err := r.db.QueryRow(ctx,
		`SELECT price_per_unit FROM ingredients WHERE id = $1`, i.ID,
	).Scan(&oldPrice)
	if err != nil {
		return Ingredient{}, fmt.Errorf("get old price: %w", err)
	}

	i.PricePerUnit = calcPricePerUnit(i.PackagePrice, i.PackageSize)

	err = r.db.QueryRow(ctx, `
		UPDATE ingredients
		SET name=$1, default_unit=$2, package_size=$3, package_price=$4,
		    price_per_unit=$5, stock_quantity=$6, alert_threshold=$7,
		    supplier=$8, updated_at=NOW()
		WHERE id=$9
		RETURNING id, name, default_unit, package_size, package_price, price_per_unit,
		          stock_quantity, alert_threshold, supplier, created_at, updated_at`,
		i.Name, i.DefaultUnit, i.PackageSize, i.PackagePrice, i.PricePerUnit,
		i.StockQuantity, i.AlertThreshold, i.Supplier, i.ID,
	).Scan(
		&i.ID, &i.Name, &i.DefaultUnit, &i.PackageSize, &i.PackagePrice, &i.PricePerUnit,
		&i.StockQuantity, &i.AlertThreshold, &i.Supplier,
		&i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return Ingredient{}, fmt.Errorf("update ingredient: %w", err)
	}

	if oldPrice != i.PricePerUnit {
		_, err = r.db.Exec(ctx,
			`INSERT INTO ingredient_price_history (ingredient_id, price_per_unit, unit)
			 VALUES ($1, $2, $3)`,
			i.ID, i.PricePerUnit, i.DefaultUnit,
		)
		if err != nil {
			return Ingredient{}, fmt.Errorf("create price history: %w", err)
		}
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
