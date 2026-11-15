-- Sprint 16. A recipe may have a type ("Tortas bizcocho", "Hojarasca", "Merengue"...) that the owner creates
-- and manages. The type is optional: deleting one leaves its recipes without type.
CREATE TABLE recipe_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(50) NOT NULL CHECK (length(btrim(name)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX recipe_categories_name_lower_key ON recipe_categories (lower(name));
ALTER TABLE recipe_categories ENABLE ROW LEVEL SECURITY;

ALTER TABLE recipes ADD COLUMN category_id UUID REFERENCES recipe_categories(id) ON DELETE SET NULL;
CREATE INDEX recipes_category_id_idx ON recipes (category_id);
