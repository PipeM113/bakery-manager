-- A sale is of whole recipes or of portions, and each of its lines remembers how much stock it
-- really took (a sale without enough stock takes only what there was).
ALTER TABLE sales
    ADD COLUMN sold_as VARCHAR(10) NOT NULL DEFAULT 'recipe' CHECK (sold_as IN ('recipe', 'portion'));

ALTER TABLE sale_ingredients ADD COLUMN stock_deducted NUMERIC(12, 3);
UPDATE sale_ingredients SET stock_deducted = ROUND(quantity_used, 3);
ALTER TABLE sale_ingredients ALTER COLUMN stock_deducted SET NOT NULL;
ALTER TABLE sale_ingredients ADD CONSTRAINT sale_ingredients_stock_deducted_nonnegative CHECK (stock_deducted >= 0);
