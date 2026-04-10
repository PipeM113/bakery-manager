CREATE TYPE quotation_status AS ENUM ('pending', 'confirmed', 'cancelled');

CREATE TABLE quotations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipe_id   UUID NOT NULL REFERENCES recipes(id),
    client_name TEXT NOT NULL,
    margin_pct  NUMERIC(6, 4) NOT NULL,
    final_price NUMERIC(12, 2) NOT NULL,
    status      quotation_status NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quotations_user_id ON quotations(user_id);
CREATE INDEX idx_quotations_status  ON quotations(status);
