-- Sprint 14. The same product can be bought from two suppliers at two prices, so an ingredient is
-- identified by name + brand + supplier. The two equivalences (weight of 1 litre, weight of 1 unit)
-- let a recipe use a unit of another family; 0 means "not known" and nothing is preloaded.
-- A purchase is a new reason of stock adjustment.
ALTER TABLE ingredients
    ADD COLUMN supplier        TEXT    NOT NULL DEFAULT '',
    ADD COLUMN grams_per_liter NUMERIC NOT NULL DEFAULT 0 CHECK (grams_per_liter >= 0),
    ADD COLUMN grams_per_unit  NUMERIC NOT NULL DEFAULT 0 CHECK (grams_per_unit >= 0);

UPDATE ingredients SET brand = '' WHERE brand IS NULL;

ALTER TABLE ingredients DROP CONSTRAINT ingredients_name_unique;
CREATE UNIQUE INDEX ingredients_identity ON ingredients (lower(name), lower(coalesce(brand, '')), lower(supplier));

ALTER TABLE stock_adjustments DROP CONSTRAINT stock_adjustments_reason_check;
ALTER TABLE stock_adjustments ADD CONSTRAINT stock_adjustments_reason_check
    CHECK (reason IN ('merma', 'correccion', 'otro', 'compra'));
