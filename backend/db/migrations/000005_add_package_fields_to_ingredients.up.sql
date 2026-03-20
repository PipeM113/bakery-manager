ALTER TABLE ingredients
  ADD COLUMN package_size  DECIMAL(12,4) NOT NULL DEFAULT 1,
  ADD COLUMN package_price DECIMAL(12,2) NOT NULL DEFAULT 0;

UPDATE ingredients
SET package_price = price_per_unit,
    package_size  = 1;