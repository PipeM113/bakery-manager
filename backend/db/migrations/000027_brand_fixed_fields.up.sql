-- The name of the business and its Instagram belong to the brand and live in the code
-- (internal/quotations/service), so the screen no longer edits them.
ALTER TABLE business_profile DROP COLUMN name;
ALTER TABLE business_profile DROP COLUMN instagram;
