-- Brings back the old key and drops the detail of a purchase. It CANNOT split what the up migration merged:
-- the merged ingredients stay merged (restore the backup if that is needed).
ALTER TABLE stock_adjustments
    DROP CONSTRAINT stock_adjustments_package_together,
    DROP CONSTRAINT stock_adjustments_purchase_detail,
    DROP COLUMN package_price,
    DROP COLUMN package_size,
    DROP COLUMN supplier;

DROP INDEX ingredients_identity;
CREATE UNIQUE INDEX ingredients_identity ON ingredients (lower(name), lower(coalesce(brand, '')), lower(supplier));
