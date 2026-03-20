CREATE TABLE ingredients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(150) NOT NULL,
    default_unit    VARCHAR(20)  NOT NULL,
    price_per_unit  DECIMAL(12,4) NOT NULL,
    stock_quantity  DECIMAL(12,3) NOT NULL DEFAULT 0,
    alert_threshold DECIMAL(12,3) NOT NULL DEFAULT 0,
    supplier        VARCHAR(100),
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT ingredients_name_unique UNIQUE (name)
);

CREATE TABLE ingredient_price_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ingredient_id   UUID NOT NULL REFERENCES ingredients(id) ON DELETE CASCADE,
    price_per_unit  DECIMAL(12,4) NOT NULL,
    unit            VARCHAR(20) NOT NULL,
    effective_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_price_history_ingredient ON ingredient_price_history(ingredient_id);