package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecipeIngredient struct {
	ID           string  `json:"id"`
	RecipeID     string  `json:"recipe_id"`
	IngredientID string  `json:"ingredient_id"`
	Name         string  `json:"name"`
	Kind         string  `json:"kind"` // "ingredient" or "supply"
	Brand        string  `json:"brand"`
	Supplier     string  `json:"supplier"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
}

type Recipe struct {
	ID              string             `json:"id"`
	CreatedBy       *string            `json:"created_by"`
	ParentID        *string            `json:"parent_id"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	Yield           float64            `json:"yield"`
	YieldUnit       string             `json:"yield_unit"`
	PhotoURL        *string            `json:"photo_url"`
	IsBase          bool               `json:"is_base"`
	ScaleFactor     float64            `json:"scale_factor"`
	IndirectCostPct float64            `json:"indirect_cost_pct"`
	LaborCostPct    float64            `json:"labor_cost_pct"`
	MarginPct       float64            `json:"margin_pct"`
	ExtraCharge     int                `json:"extra_charge"`
	DeliveryCost    int                `json:"delivery_cost"`
	Ingredients     []RecipeIngredient `json:"ingredients"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type RecipeRepository struct {
	db *pgxpool.Pool
}

func NewRecipeRepository(db *pgxpool.Pool) *RecipeRepository {
	return &RecipeRepository{db: db}
}

func (r *RecipeRepository) Create(ctx context.Context, recipe Recipe) (Recipe, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Recipe{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if recipe.ScaleFactor == 0 {
		recipe.ScaleFactor = 1.0
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO recipes (created_by, parent_id, name, description, yield, yield_unit, photo_url, is_base, scale_factor, indirect_cost_pct, labor_cost_pct, margin_pct, extra_charge, delivery_cost)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at`,
		recipe.CreatedBy, recipe.ParentID, recipe.Name, recipe.Description,
		recipe.Yield, recipe.YieldUnit, recipe.PhotoURL, recipe.IsBase,
		recipe.ScaleFactor, recipe.IndirectCostPct, recipe.LaborCostPct, recipe.MarginPct,
		recipe.ExtraCharge, recipe.DeliveryCost,
	).Scan(&recipe.ID, &recipe.CreatedAt, &recipe.UpdatedAt)
	if err != nil {
		return Recipe{}, fmt.Errorf("insert recipe: %w", err)
	}

	for i, ing := range recipe.Ingredients {
		err = tx.QueryRow(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, ingredient_id, quantity, unit)
			VALUES ($1,$2,$3,$4) RETURNING id`,
			recipe.ID, ing.IngredientID, ing.Quantity, ing.Unit,
		).Scan(&recipe.Ingredients[i].ID)
		if err != nil {
			return Recipe{}, fmt.Errorf("insert ingredient: %w", err)
		}
		recipe.Ingredients[i].RecipeID = recipe.ID
	}

	if err := tx.Commit(ctx); err != nil {
		return Recipe{}, fmt.Errorf("commit tx: %w", err)
	}
	return r.GetByID(ctx, recipe.ID)
}

// GetAll lists every recipe with its ingredients, in two queries however many recipes there are.
func (r *RecipeRepository) GetAll(ctx context.Context) ([]Recipe, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, created_by, parent_id, name, description, yield, yield_unit,
		       photo_url, is_base, scale_factor, indirect_cost_pct, labor_cost_pct, margin_pct,
		       extra_charge, delivery_cost, created_at, updated_at
		FROM recipes`)
	if err != nil {
		return nil, fmt.Errorf("get all recipes: %w", err)
	}
	defer rows.Close()

	var recipes []Recipe
	index := map[string]int{}
	for rows.Next() {
		var rec Recipe
		err := rows.Scan(
			&rec.ID, &rec.CreatedBy, &rec.ParentID, &rec.Name, &rec.Description,
			&rec.Yield, &rec.YieldUnit, &rec.PhotoURL, &rec.IsBase,
			&rec.ScaleFactor, &rec.IndirectCostPct, &rec.LaborCostPct, &rec.MarginPct,
			&rec.ExtraCharge, &rec.DeliveryCost, &rec.CreatedAt, &rec.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan recipe: %w", err)
		}
		rec.Ingredients = []RecipeIngredient{}
		index[rec.ID] = len(recipes)
		recipes = append(recipes, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read recipes: %w", err)
	}
	rows.Close()
	if len(recipes) == 0 {
		return recipes, nil
	}
	// Alphabetical without case or accents (the database collation does not do that).
	sort.SliceStable(recipes, func(a, b int) bool { return kernel.SortKey(recipes[a].Name) < kernel.SortKey(recipes[b].Name) })
	for i, rec := range recipes {
		index[rec.ID] = i
	}

	ids := make([]string, 0, len(recipes))
	for _, rec := range recipes {
		ids = append(ids, rec.ID)
	}
	ingRows, err := r.db.Query(ctx, `
		SELECT ri.id, ri.recipe_id, ri.ingredient_id, i.name, i.kind, i.brand, i.supplier, ri.quantity, ri.unit
		FROM recipe_ingredients ri
		JOIN ingredients i ON i.id = ri.ingredient_id
		WHERE ri.recipe_id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("get ingredients: %w", err)
	}
	defer ingRows.Close()
	for ingRows.Next() {
		var ing RecipeIngredient
		if err := ingRows.Scan(&ing.ID, &ing.RecipeID, &ing.IngredientID, &ing.Name, &ing.Kind, &ing.Brand, &ing.Supplier, &ing.Quantity, &ing.Unit); err != nil {
			return nil, fmt.Errorf("scan ingredient: %w", err)
		}
		i := index[ing.RecipeID]
		recipes[i].Ingredients = append(recipes[i].Ingredients, ing)
	}
	if err := ingRows.Err(); err != nil {
		return nil, fmt.Errorf("read ingredients: %w", err)
	}
	return recipes, nil
}

func (r *RecipeRepository) GetByID(ctx context.Context, id string) (Recipe, error) {
	var rec Recipe
	err := r.db.QueryRow(ctx, `
		SELECT id, created_by, parent_id, name, description, yield, yield_unit,
		       photo_url, is_base, scale_factor, indirect_cost_pct, labor_cost_pct, margin_pct,
		       extra_charge, delivery_cost, created_at, updated_at
		FROM recipes WHERE id = $1`, id,
	).Scan(
		&rec.ID, &rec.CreatedBy, &rec.ParentID, &rec.Name, &rec.Description,
		&rec.Yield, &rec.YieldUnit, &rec.PhotoURL, &rec.IsBase,
		&rec.ScaleFactor, &rec.IndirectCostPct, &rec.LaborCostPct, &rec.MarginPct,
		&rec.ExtraCharge, &rec.DeliveryCost, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return Recipe{}, fmt.Errorf("get recipe: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT ri.id, ri.recipe_id, ri.ingredient_id, i.name, i.kind, i.brand, i.supplier, ri.quantity, ri.unit
		FROM recipe_ingredients ri
		JOIN ingredients i ON i.id = ri.ingredient_id
		WHERE ri.recipe_id = $1`, id)
	if err != nil {
		return Recipe{}, fmt.Errorf("get recipe ingredients: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ing RecipeIngredient
		if err := rows.Scan(&ing.ID, &ing.RecipeID, &ing.IngredientID, &ing.Name, &ing.Kind, &ing.Brand, &ing.Supplier, &ing.Quantity, &ing.Unit); err != nil {
			return Recipe{}, fmt.Errorf("scan ingredient: %w", err)
		}
		rec.Ingredients = append(rec.Ingredients, ing)
	}
	if rec.Ingredients == nil {
		rec.Ingredients = []RecipeIngredient{}
	}
	return rec, nil
}

func (r *RecipeRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM recipes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete recipe: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("recipe not found")
	}
	return nil
}

func (r *RecipeRepository) Update(ctx context.Context, id string, recipe Recipe) (Recipe, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Recipe{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE recipes
		SET name=$1, description=$2, yield=$3, yield_unit=$4, indirect_cost_pct=$5,
		    labor_cost_pct=$6,
		    scale_factor=CASE WHEN $8 > 0 THEN $8 ELSE scale_factor END,
		    updated_at=NOW()
		WHERE id=$7`,
		recipe.Name, recipe.Description, recipe.Yield, recipe.YieldUnit,
		recipe.IndirectCostPct, recipe.LaborCostPct, id, recipe.ScaleFactor,
	)
	if err != nil {
		return Recipe{}, fmt.Errorf("update recipe: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Recipe{}, fmt.Errorf("recipe not found")
	}

	_, err = tx.Exec(ctx, `DELETE FROM recipe_ingredients WHERE recipe_id=$1`, id)
	if err != nil {
		return Recipe{}, fmt.Errorf("delete old ingredients: %w", err)
	}

	for _, ing := range recipe.Ingredients {
		_, err = tx.Exec(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, ingredient_id, quantity, unit)
			VALUES ($1,$2,$3,$4)`,
			id, ing.IngredientID, ing.Quantity, ing.Unit,
		)
		if err != nil {
			return Recipe{}, fmt.Errorf("insert ingredient: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Recipe{}, fmt.Errorf("commit tx: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *RecipeRepository) UpdatePhotoURL(ctx context.Context, id string, photoURL string) error {
	_, err := r.db.Exec(ctx, `UPDATE recipes SET photo_url=$1, updated_at=NOW() WHERE id=$2`, photoURL, id)
	return err
}

func (r *RecipeRepository) UpdateCosts(ctx context.Context, id string, indirectPct, laborPct, marginPct float64, extraCharge, deliveryCost int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE recipes
		SET indirect_cost_pct=$1, labor_cost_pct=$2, margin_pct=$3,
		    extra_charge=$4, delivery_cost=$5, updated_at=NOW()
		WHERE id=$6`,
		indirectPct, laborPct, marginPct, extraCharge, deliveryCost, id,
	)
	if err != nil {
		return fmt.Errorf("update costs: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("recipe not found")
	}
	return nil
}

func (r *RecipeRepository) SaveScaled(ctx context.Context, recipeID string, scaleFactor float64, newName string) (Recipe, error) {
	original, err := r.GetByID(ctx, recipeID)
	if err != nil {
		return Recipe{}, fmt.Errorf("get original recipe: %w", err)
	}

	scaled := Recipe{
		CreatedBy:       original.CreatedBy,
		ParentID:        &recipeID,
		Name:            newName,
		Description:     original.Description,
		Yield:           original.Yield * scaleFactor,
		YieldUnit:       original.YieldUnit,
		PhotoURL:        original.PhotoURL,
		IsBase:          false,
		ScaleFactor:     scaleFactor,
		IndirectCostPct: original.IndirectCostPct,
		LaborCostPct:    original.LaborCostPct,
		MarginPct:       original.MarginPct,
		ExtraCharge:     original.ExtraCharge,
		DeliveryCost:    original.DeliveryCost,
	}
	for _, ing := range original.Ingredients {
		scaled.Ingredients = append(scaled.Ingredients, RecipeIngredient{
			IngredientID: ing.IngredientID,
			Quantity:     ing.Quantity * scaleFactor,
			Unit:         ing.Unit,
		})
	}
	return r.Create(ctx, scaled)
}

// PhotoURLOf is the photo address a recipe has now ("" when it has none or does not exist).
func (r *RecipeRepository) PhotoURLOf(ctx context.Context, id string) (string, error) {
	var url *string
	err := r.db.QueryRow(ctx, `SELECT photo_url FROM recipes WHERE id = $1`, id).Scan(&url)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("read photo url: %w", err)
	}
	if url == nil {
		return "", nil
	}
	return *url, nil
}

// PhotoInUse says whether a photo is still shown by another recipe or by a saved quotation: a
// quotation keeps the address of the photo it had, so its PDF still shows it.
func (r *RecipeRepository) PhotoInUse(ctx context.Context, url, exceptRecipeID string) (bool, error) {
	var used bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM recipes WHERE photo_url = $1 AND id::text <> $2)
		    OR EXISTS (SELECT 1 FROM quotation_lines WHERE photo_url = $1)`, url, exceptRecipeID).Scan(&used)
	if err != nil {
		return true, fmt.Errorf("check photo use: %w", err) // when in doubt, keep it
	}
	return used, nil
}

// UnitError is a line whose unit cannot be used with its ingredient.
type UnitError struct{ Message string }

func (e *UnitError) Error() string { return e.Message }

// CheckUnits refuses a line whose unit is of another family than its ingredient's when the ingredient
// does not have the equivalence that joins them (the weight of 1 litre or of 1 unit).
func (r *RecipeRepository) CheckUnits(ctx context.Context, lines []RecipeIngredient) error {
	ids := make([]string, 0, len(lines))
	for _, l := range lines {
		ids = append(ids, l.IngredientID)
	}
	rows, err := r.db.Query(ctx, `SELECT id::text, name, default_unit, grams_per_liter, grams_per_unit
		FROM ingredients WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return fmt.Errorf("get ingredient units: %w", err)
	}
	defer rows.Close()
	type known struct {
		name, unit string
		eq         kernel.Equivalence
	}
	byID := map[string]known{}
	for rows.Next() {
		var id string
		var k known
		if err := rows.Scan(&id, &k.name, &k.unit, &k.eq.GramsPerLiter, &k.eq.GramsPerUnit); err != nil {
			return fmt.Errorf("scan ingredient units: %w", err)
		}
		byID[id] = k
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, l := range lines {
		k, ok := byID[l.IngredientID]
		if !ok {
			continue // the insert reports an ingredient that does not exist
		}
		if _, err := kernel.Convert(1, l.Unit, k.unit, k.eq); err != nil {
			return &UnitError{fmt.Sprintf("%s en %s: %v", k.name, l.Unit, err)}
		}
	}
	return nil
}
