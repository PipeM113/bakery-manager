-- The business is one: both accounts see and administer everything. A record only remembers who
-- registered it (created_by). If that user is deleted the record stays, with no creator.
DO $$
DECLARE
    t text;
    c text;
BEGIN
    FOREACH t IN ARRAY ARRAY['recipes', 'fixed_costs', 'sales', 'orders', 'operational_expenses', 'quotations', 'stock_adjustments'] LOOP
        EXECUTE format('ALTER TABLE %I RENAME COLUMN user_id TO created_by', t);
        EXECUTE format('ALTER TABLE %I ALTER COLUMN created_by DROP NOT NULL', t);
        FOR c IN SELECT conname FROM pg_constraint
                 WHERE conrelid = format('%I', t)::regclass AND contype = 'f' AND confrelid = 'users'::regclass LOOP
            EXECUTE format('ALTER TABLE %I DROP CONSTRAINT %I', t, c);
        END LOOP;
        EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL', t, t || '_created_by_fkey');
    END LOOP;
END
$$;
