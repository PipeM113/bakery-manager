package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RecipeIngredient struct {
	ID           string  `json:"id"`
	RecipeID     string  `json:"recipe_id"`
	IngredientID string  `json:"ingredient_id"`
	Name         string  `json:"name"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
}

type Recipe struct {
	ID              string             `json:"id"`
	UserID          string             `json:"user_id"`
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
		INSERT INTO recipes (user_id, parent_id, name, description, yield, yield_unit, photo_url, is_base, scale_factor, indirect_cost_pct, labor_cost_pct)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at, updated_at`,
		recipe.UserID, recipe.ParentID, recipe.Name, recipe.Description,
		recipe.Yield, recipe.YieldUnit, recipe.PhotoURL, recipe.IsBase,
		recipe.ScaleFactor, recipe.IndirectCostPct, recipe.LaborCostPct,
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

func (r *RecipeRepository) GetAll(ctx context.Context) ([]Recipe, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, parent_id, name, description, yield, yield_unit,
		       photo_url, is_base, scale_factor, indirect_cost_pct, labor_cost_pct, created_at, updated_at
		FROM recipes ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("get all recipes: %w", err)
	}
	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var rec Recipe
		err := rows.Scan(
			&rec.ID, &rec.UserID, &rec.ParentID, &rec.Name, &rec.Description,
			&rec.Yield, &rec.YieldUnit, &rec.PhotoURL, &rec.IsBase,
			&rec.ScaleFactor, &rec.IndirectCostPct, &rec.LaborCostPct, &rec.CreatedAt, &rec.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan recipe: %w", err)
		}
		// Cargar ingredientes de cada receta
		ingRows, err := r.db.Query(ctx, `
			SELECT ri.id, ri.recipe_id, ri.ingredient_id, i.name, ri.quantity, ri.unit
			FROM recipe_ingredients ri
			JOIN ingredients i ON i.id = ri.ingredient_id
			WHERE ri.recipe_id = $1`, rec.ID)
		if err != nil {
			return nil, fmt.Errorf("get ingredients: %w", err)
		}
		for ingRows.Next() {
			var ing RecipeIngredient
			if err := ingRows.Scan(&ing.ID, &ing.RecipeID, &ing.IngredientID, &ing.Name, &ing.Quantity, &ing.Unit); err != nil {
				ingRows.Close()
				return nil, fmt.Errorf("scan ingredient: %w", err)
			}
			rec.Ingredients = append(rec.Ingredients, ing)
		}
		ingRows.Close()
		if rec.Ingredients == nil {
			rec.Ingredients = []RecipeIngredient{}
		}
		recipes = append(recipes, rec)
	}
	return recipes, nil
}

func (r *RecipeRepository) GetByID(ctx context.Context, id string) (Recipe, error) {
	var rec Recipe
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, parent_id, name, description, yield, yield_unit,
		       photo_url, is_base, scale_factor, indirect_cost_pct, labor_cost_pct, created_at, updated_at
		FROM recipes WHERE id = $1`, id,
	).Scan(
		&rec.ID, &rec.UserID, &rec.ParentID, &rec.Name, &rec.Description,
		&rec.Yield, &rec.YieldUnit, &rec.PhotoURL, &rec.IsBase,
		&rec.ScaleFactor, &rec.IndirectCostPct, &rec.LaborCostPct, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return Recipe{}, fmt.Errorf("get recipe: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT ri.id, ri.recipe_id, ri.ingredient_id, i.name, ri.quantity, ri.unit
		FROM recipe_ingredients ri
		JOIN ingredients i ON i.id = ri.ingredient_id
		WHERE ri.recipe_id = $1`, id)
	if err != nil {
		return Recipe{}, fmt.Errorf("get recipe ingredients: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ing RecipeIngredient
		if err := rows.Scan(&ing.ID, &ing.RecipeID, &ing.IngredientID, &ing.Name, &ing.Quantity, &ing.Unit); err != nil {
			return Recipe{}, fmt.Errorf("scan ingredient: %w", err)
		}
		rec.Ingredients = append(rec.Ingredients, ing)
	}
	if rec.Ingredients == nil {
		rec.Ingredients = []RecipeIngredient{}
	}
	return rec, nil
}

func (r *RecipeRepository) Delete(ctx context.Context, id, userID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM recipes WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete recipe: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("recipe not found or not owned by user")
	}
	return nil
}

func (r *RecipeRepository) Update(ctx context.Context, id, userID string, recipe Recipe) (Recipe, error) {
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
		WHERE id=$7 AND user_id=$9`,
		recipe.Name, recipe.Description, recipe.Yield, recipe.YieldUnit,
		recipe.IndirectCostPct, recipe.LaborCostPct, id, recipe.ScaleFactor, userID,
	)
	if err != nil {
		return Recipe{}, fmt.Errorf("update recipe: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Recipe{}, fmt.Errorf("recipe not found or not owned by user")
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

func (r *RecipeRepository) SaveScaled(ctx context.Context, recipeID string, scaleFactor float64, newName string) (Recipe, error) {
	original, err := r.GetByID(ctx, recipeID)
	if err != nil {
		return Recipe{}, fmt.Errorf("get original recipe: %w", err)
	}

	scaled := Recipe{
		UserID:          original.UserID,
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
