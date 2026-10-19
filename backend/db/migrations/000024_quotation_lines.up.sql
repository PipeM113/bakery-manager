-- A quotation has several lines (recipe, quantity, price) and one delivery. What a line says
-- (name, photo, prices) is copied, so a saved quotation never changes when costs or recipes do.
-- Quotations that already exist become quotations of one line; their total keeps its meaning.
ALTER TABLE quotations ADD COLUMN delivery_cost BIGINT NOT NULL DEFAULT 0 CHECK (delivery_cost >= 0);
ALTER TABLE quotations ALTER COLUMN recipe_id DROP NOT NULL;
-- The percentages now live in each line.
ALTER TABLE quotations ALTER COLUMN margin_pct DROP NOT NULL;

CREATE TABLE quotation_lines (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quotation_id      UUID NOT NULL REFERENCES quotations(id) ON DELETE CASCADE,
    position          INTEGER NOT NULL CHECK (position >= 1),
    recipe_id         UUID REFERENCES recipes(id),
    recipe_name       TEXT NOT NULL,
    photo_url         TEXT,
    sold_as           TEXT NOT NULL DEFAULT 'recipe' CHECK (sold_as IN ('recipe', 'portion')),
    quantity          INTEGER NOT NULL CHECK (quantity >= 1),
    unit_price        BIGINT NOT NULL CHECK (unit_price >= 0),
    line_total        BIGINT NOT NULL CHECK (line_total >= 0),
    margin_pct        NUMERIC(6, 4),
    indirect_cost_pct NUMERIC(6, 4),
    labor_cost_pct    NUMERIC(6, 4),
    extra_charge      BIGINT NOT NULL DEFAULT 0 CHECK (extra_charge >= 0),
    UNIQUE (quotation_id, position)
);
CREATE INDEX idx_quotation_lines_quotation_id ON quotation_lines(quotation_id);
ALTER TABLE quotation_lines ENABLE ROW LEVEL SECURITY;

INSERT INTO quotation_lines (quotation_id, position, recipe_id, recipe_name, photo_url, sold_as, quantity, unit_price, line_total, margin_pct)
SELECT q.id, 1, q.recipe_id, r.name, r.photo_url, 'recipe', 1, q.final_price, q.final_price, q.margin_pct
FROM quotations q
JOIN recipes r ON r.id = q.recipe_id;
