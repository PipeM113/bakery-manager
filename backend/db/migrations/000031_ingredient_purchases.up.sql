-- ADR-005: an ingredient is name + brand. The supplier, the size of the package and the price paid belong to
-- each purchase (a stock adjustment with the reason 'compra').
--
-- 1. Merge what the old key (name + brand + supplier) kept apart. The merge comes first: when it needs a person
--    to decide, the whole migration stops, with a message, before anything else changes.
DO $$
DECLARE
    g       record;
    keep    uuid;
    others  uuid[];
    label   text;
BEGIN
    FOR g IN
        SELECT lower(name) AS n, lower(coalesce(brand, '')) AS b, min(name) AS shown, min(coalesce(brand, '')) AS shown_brand
        FROM ingredients
        GROUP BY 1, 2
        HAVING count(*) > 1
    LOOP
        label := format('"%s" (marca "%s")', g.shown, g.shown_brand);

        IF (SELECT count(DISTINCT default_unit) FROM ingredients WHERE lower(name) = g.n AND lower(coalesce(brand, '')) = g.b) > 1 THEN
            RAISE EXCEPTION 'No se puede fusionar %: las filas tienen unidades distintas. Corrígelas a la misma unidad y vuelve a aplicar.', label;
        END IF;
        IF (SELECT count(DISTINCT kind) FROM ingredients WHERE lower(name) = g.n AND lower(coalesce(brand, '')) = g.b) > 1 THEN
            RAISE EXCEPTION 'No se puede fusionar %: una fila es ingrediente y otra insumo. Corrígelo y vuelve a aplicar.', label;
        END IF;
        IF (SELECT count(DISTINCT grams_per_liter) FROM ingredients WHERE lower(name) = g.n AND lower(coalesce(brand, '')) = g.b AND grams_per_liter > 0) > 1
           OR (SELECT count(DISTINCT grams_per_unit) FROM ingredients WHERE lower(name) = g.n AND lower(coalesce(brand, '')) = g.b AND grams_per_unit > 0) > 1 THEN
            RAISE EXCEPTION 'No se puede fusionar %: las filas tienen equivalencias distintas (peso de 1 litro o de 1 unidad). Deja la correcta y vuelve a aplicar.', label;
        END IF;

        -- The row that stays is the one with the most recent price.
        SELECT i.id INTO keep
        FROM ingredients i
        WHERE lower(i.name) = g.n AND lower(coalesce(i.brand, '')) = g.b
        ORDER BY (SELECT max(h.effective_date) FROM ingredient_price_history h WHERE h.ingredient_id = i.id) DESC NULLS LAST,
                 i.updated_at DESC, i.id
        LIMIT 1;
        SELECT array_agg(i.id) INTO others
        FROM ingredients i
        WHERE lower(i.name) = g.n AND lower(coalesce(i.brand, '')) = g.b AND i.id <> keep;

        UPDATE ingredients k SET
            stock_quantity  = s.stock,
            alert_threshold = s.threshold,
            grams_per_liter = s.per_liter,
            grams_per_unit  = s.per_unit,
            updated_at      = now()
        FROM (
            SELECT sum(stock_quantity) AS stock, max(alert_threshold) AS threshold,
                   max(grams_per_liter) AS per_liter, max(grams_per_unit) AS per_unit
            FROM ingredients WHERE lower(name) = g.n AND lower(coalesce(brand, '')) = g.b
        ) s
        WHERE k.id = keep;

        UPDATE recipe_ingredients       SET ingredient_id = keep WHERE ingredient_id = ANY (others);
        UPDATE sale_ingredients         SET ingredient_id = keep WHERE ingredient_id = ANY (others);
        UPDATE stock_adjustments        SET ingredient_id = keep WHERE ingredient_id = ANY (others);
        UPDATE ingredient_price_history SET ingredient_id = keep WHERE ingredient_id = ANY (others);
        DELETE FROM ingredients WHERE id = ANY (others);
    END LOOP;
END $$;

-- 2. The key is the name and the brand.
DROP INDEX ingredients_identity;
CREATE UNIQUE INDEX ingredients_identity ON ingredients (lower(name), lower(coalesce(brand, '')));

-- 3. The detail of a purchase: supplier, size of the package (in the unit of the ingredient) and the price paid for it.
ALTER TABLE stock_adjustments
    ADD COLUMN supplier      text,
    ADD COLUMN package_size  numeric(12,4) CHECK (package_size > 0),
    ADD COLUMN package_price numeric(12,2) CHECK (package_price > 0),
    ADD CONSTRAINT stock_adjustments_purchase_detail
        CHECK (reason = 'compra' OR (supplier IS NULL AND package_size IS NULL AND package_price IS NULL)),
    ADD CONSTRAINT stock_adjustments_package_together
        CHECK ((package_size IS NULL) = (package_price IS NULL));
