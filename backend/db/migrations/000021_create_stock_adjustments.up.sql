-- Every change to the stock that is not a sale leaves a record with its reason. Only a "merma"
-- (something lost) has a loss in pesos.
CREATE TABLE stock_adjustments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ingredient_id  UUID NOT NULL REFERENCES ingredients(id) ON DELETE CASCADE,
    reason         VARCHAR(12) NOT NULL CHECK (reason IN ('merma', 'correccion', 'otro')),
    quantity_delta NUMERIC(12, 3) NOT NULL,
    unit           VARCHAR(20) NOT NULL,
    stock_before   NUMERIC(12, 3) NOT NULL CHECK (stock_before >= 0),
    stock_after    NUMERIC(12, 3) NOT NULL CHECK (stock_after >= 0),
    loss_amount    INTEGER NOT NULL DEFAULT 0 CHECK (loss_amount >= 0),
    note           TEXT NOT NULL DEFAULT '',
    adjusted_on    DATE NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (reason = 'merma' OR loss_amount = 0)
);
CREATE INDEX idx_stock_adjustments_ingredient ON stock_adjustments (ingredient_id, adjusted_on DESC);
CREATE INDEX idx_stock_adjustments_user_date ON stock_adjustments (user_id, adjusted_on);
ALTER TABLE stock_adjustments ENABLE ROW LEVEL SECURITY;
