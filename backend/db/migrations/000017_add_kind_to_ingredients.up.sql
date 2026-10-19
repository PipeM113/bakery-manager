-- An ingredient row is either an ingredient or a supply (a box, cardboard): a direct cost
-- of the recipe that the indirect and labor percentages do not apply to.
ALTER TABLE ingredients
    ADD COLUMN kind TEXT NOT NULL DEFAULT 'ingredient'
    CONSTRAINT ingredients_kind_check CHECK (kind IN ('ingredient', 'supply'));
