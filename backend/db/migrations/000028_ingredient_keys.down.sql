-- Best effort: purchases become corrections, and the rows that would share a name are renamed with
-- their brand and supplier so that the old unique name can come back.
UPDATE stock_adjustments SET reason = 'correccion' WHERE reason = 'compra';
ALTER TABLE stock_adjustments DROP CONSTRAINT stock_adjustments_reason_check;
ALTER TABLE stock_adjustments ADD CONSTRAINT stock_adjustments_reason_check
    CHECK (reason IN ('merma', 'correccion', 'otro'));

DROP INDEX ingredients_identity;
UPDATE ingredients i SET name = left(i.name || ' ' || coalesce(nullif(i.brand, ''), '') || ' ' || i.supplier || ' ' || left(i.id::text, 4), 150)
WHERE EXISTS (SELECT 1 FROM ingredients o WHERE o.id <> i.id AND lower(o.name) = lower(i.name));
ALTER TABLE ingredients ADD CONSTRAINT ingredients_name_unique UNIQUE (name);

ALTER TABLE ingredients DROP COLUMN grams_per_unit, DROP COLUMN grams_per_liter, DROP COLUMN supplier;
