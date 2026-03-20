CREATE TABLE recipes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id    UUID REFERENCES recipes(id) ON DELETE SET NULL,
    name         VARCHAR(200) NOT NULL,
    description  TEXT,
    yield        DECIMAL(10,2) NOT NULL DEFAULT 1,
    yield_unit   VARCHAR(50) NOT NULL DEFAULT 'porciones',
    photo_url    VARCHAR(500),
    is_base      BOOLEAN NOT NULL DEFAULT true,
    indirect_cost_pct DECIMAL(5,4) NOT NULL DEFAULT 0.15,
    labor_cost_pct    DECIMAL(5,4) NOT NULL DEFAULT 0.30,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE recipe_ingredients (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id     UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    ingredient_id UUID NOT NULL REFERENCES ingredients(id) ON DELETE RESTRICT,
    quantity      DECIMAL(12,4) NOT NULL,
    unit          VARCHAR(20) NOT NULL
);

CREATE INDEX idx_recipe_ingredients_recipe ON recipe_ingredients(recipe_id);
CREATE INDEX idx_recipes_parent ON recipes(parent_id);
CREATE INDEX idx_recipes_user ON recipes(user_id);