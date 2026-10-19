-- An order groups the sales of several products. Each line is a sale, so the stock and the reports
-- work as before; line_total is what the line really brought in (an order may have its own total
-- price, shared between its lines).
CREATE TABLE orders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    customer_name TEXT NOT NULL DEFAULT '',
    notes         TEXT NOT NULL DEFAULT '',
    order_date    DATE NOT NULL,
    total_price   INTEGER NOT NULL CHECK (total_price >= 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_orders_user_date ON orders (user_id, order_date DESC);
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

ALTER TABLE sales ADD COLUMN order_id UUID REFERENCES orders(id) ON DELETE CASCADE;
ALTER TABLE sales ADD COLUMN line_total BIGINT;
UPDATE sales SET line_total = quantity_sold::bigint * unit_price WHERE line_total IS NULL;
ALTER TABLE sales ALTER COLUMN line_total SET NOT NULL;
ALTER TABLE sales ADD CONSTRAINT sales_line_total_nonnegative CHECK (line_total >= 0);
CREATE INDEX idx_sales_order ON sales (order_id);
