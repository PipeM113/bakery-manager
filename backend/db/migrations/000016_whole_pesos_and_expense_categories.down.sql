-- Back to the free-text category (the old check only allowed three values; any other
-- category goes back as 'otros') and to decimal amounts.
ALTER TABLE operational_expenses ADD COLUMN category VARCHAR(50);
UPDATE operational_expenses e
SET category = CASE WHEN lower(c.name) IN ('ingredientes', 'servicios', 'otros') THEN lower(c.name) ELSE 'otros' END
FROM expense_categories c
WHERE c.id = e.category_id;
ALTER TABLE operational_expenses ALTER COLUMN category SET NOT NULL;
ALTER TABLE operational_expenses
    ADD CONSTRAINT operational_expenses_category_check CHECK (category IN ('ingredientes', 'servicios', 'otros'));
DROP INDEX idx_operational_expenses_category_id;
ALTER TABLE operational_expenses DROP COLUMN category_id;
DROP TABLE expense_categories;

ALTER TABLE operational_expenses ALTER COLUMN amount TYPE NUMERIC(12, 2);
ALTER TABLE quotations ALTER COLUMN final_price TYPE NUMERIC(12, 2);
ALTER TABLE fixed_costs ALTER COLUMN monthly_amount TYPE NUMERIC(12, 2);
ALTER TABLE sales ALTER COLUMN unit_price TYPE NUMERIC(12, 2);
