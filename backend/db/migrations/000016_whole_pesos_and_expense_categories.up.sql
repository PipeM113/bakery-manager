-- Money is whole pesos, and an expense points at a category the user can manage.

-- 1. Whole pesos (existing amounts are rounded to the nearest peso, halves up).
ALTER TABLE sales
    ALTER COLUMN unit_price TYPE INTEGER USING ROUND(unit_price)::integer;
ALTER TABLE fixed_costs
    ALTER COLUMN monthly_amount TYPE INTEGER USING ROUND(monthly_amount)::integer;
ALTER TABLE quotations
    ALTER COLUMN final_price TYPE INTEGER USING ROUND(final_price)::integer;
-- The expense keeps its "amount > 0" check, so an amount under half a peso stays at 1.
ALTER TABLE operational_expenses
    ALTER COLUMN amount TYPE INTEGER USING GREATEST(ROUND(amount), 1)::integer;

-- 2. Expense categories.
CREATE TABLE expense_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(50) NOT NULL CHECK (length(btrim(name)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX expense_categories_name_lower_key ON expense_categories (lower(name));
ALTER TABLE expense_categories ENABLE ROW LEVEL SECURITY;

INSERT INTO expense_categories (name) VALUES
    ('Gas'), ('Luz'), ('Agua'), ('Arriendo'), ('Envases'), ('Transporte'), ('Sueldos'), ('Otros');

-- Whatever category the old expenses used and the list above does not have yet.
INSERT INTO expense_categories (name)
SELECT DISTINCT initcap(e.category)
FROM operational_expenses e
WHERE NOT EXISTS (SELECT 1 FROM expense_categories c WHERE lower(c.name) = lower(e.category));

ALTER TABLE operational_expenses
    ADD COLUMN category_id UUID REFERENCES expense_categories(id) ON DELETE RESTRICT;
UPDATE operational_expenses e
SET category_id = c.id
FROM expense_categories c
WHERE lower(c.name) = lower(e.category);
ALTER TABLE operational_expenses ALTER COLUMN category_id SET NOT NULL;
ALTER TABLE operational_expenses DROP COLUMN category;
CREATE INDEX idx_operational_expenses_category_id ON operational_expenses(category_id);
