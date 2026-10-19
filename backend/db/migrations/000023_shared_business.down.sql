-- Records whose creator was deleted go to the oldest user (user_id is required again).
DO $$
DECLARE
    t text;
    c text;
BEGIN
    FOREACH t IN ARRAY ARRAY['recipes', 'fixed_costs', 'sales', 'orders', 'operational_expenses', 'quotations', 'stock_adjustments'] LOOP
        EXECUTE format('UPDATE %I SET created_by = (SELECT id FROM users ORDER BY created_at LIMIT 1) WHERE created_by IS NULL', t);
        FOR c IN SELECT conname FROM pg_constraint
                 WHERE conrelid = format('%I', t)::regclass AND contype = 'f' AND confrelid = 'users'::regclass LOOP
            EXECUTE format('ALTER TABLE %I DROP CONSTRAINT %I', t, c);
        END LOOP;
        EXECUTE format('ALTER TABLE %I ALTER COLUMN created_by SET NOT NULL', t);
        EXECUTE format('ALTER TABLE %I RENAME COLUMN created_by TO user_id', t);
        EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE', t, t || '_user_id_fkey');
    END LOOP;
END
$$;
