CREATE TABLE sales (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipe_id   UUID NOT NULL REFERENCES recipes(id),
    quantity_sold INT NOT NULL CHECK (quantity_sold > 0),
    unit_price  NUMERIC(12, 2) NOT NULL CHECK (unit_price >= 0),
    sale_date   DATE NOT NULL DEFAULT CURRENT_DATE,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sale_ingredients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id         UUID NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    ingredient_id   UUID NOT NULL REFERENCES ingredients(id),
    quantity_used   NUMERIC(12, 4) NOT NULL,
    unit            TEXT NOT NULL,
    price_at_time   NUMERIC(12, 4) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sales_user_id   ON sales(user_id);
CREATE INDEX idx_sales_sale_date ON sales(sale_date);
CREATE INDEX idx_sale_ingredients_sale_id ON sale_ingredients(sale_id);
