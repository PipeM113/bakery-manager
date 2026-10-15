DO $$
DECLARE t record;
BEGIN
  FOR t IN SELECT tablename FROM pg_tables WHERE schemaname = current_schema() LOOP
    EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', t.tablename);
  END LOOP;
END
$$;
