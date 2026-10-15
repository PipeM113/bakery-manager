-- Row level security on every table of the current schema, with no policies: only the
-- owner (the application role) can read or write. Supabase's Data API roles get nothing
-- even if the schema is exposed by mistake.
DO $$
DECLARE t record;
BEGIN
  FOR t IN SELECT tablename FROM pg_tables WHERE schemaname = current_schema() LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t.tablename);
  END LOOP;
END
$$;
