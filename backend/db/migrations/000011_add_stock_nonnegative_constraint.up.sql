-- Fix any existing rows that violate the constraint before adding it
UPDATE ingredients SET stock_quantity = 0 WHERE stock_quantity < 0;

ALTER TABLE ingredients
  ADD CONSTRAINT check_stock_nonnegative
  CHECK (stock_quantity >= 0);
