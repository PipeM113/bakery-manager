DROP INDEX IF EXISTS recipes_category_id_idx;
ALTER TABLE recipes DROP COLUMN category_id;
DROP TABLE recipe_categories;
