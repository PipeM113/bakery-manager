ALTER TABLE fixed_costs
    ADD COLUMN monthly_amount INTEGER,
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- The latest amount is the one the cost keeps; the history is lost.
UPDATE fixed_costs fc
SET monthly_amount = COALESCE((
        SELECT v.monthly_amount FROM fixed_cost_versions v
        WHERE v.fixed_cost_id = fc.id ORDER BY v.valid_from DESC LIMIT 1), 1),
    is_active = (fc.ended_from IS NULL);

ALTER TABLE fixed_costs ALTER COLUMN monthly_amount SET NOT NULL;
ALTER TABLE fixed_costs DROP COLUMN ended_from;
DROP TABLE fixed_cost_versions;
