-- A fixed cost has an amount for each month: versions that start on the first of a month. The
-- cost ends from a month (ended_from) instead of being deleted, so past months keep counting.
CREATE TABLE fixed_cost_versions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fixed_cost_id UUID NOT NULL REFERENCES fixed_costs(id) ON DELETE CASCADE,
    valid_from    DATE NOT NULL CHECK (valid_from = date_trunc('month', valid_from)::date),
    monthly_amount INTEGER NOT NULL CHECK (monthly_amount > 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (fixed_cost_id, valid_from)
);
ALTER TABLE fixed_cost_versions ENABLE ROW LEVEL SECURITY;

ALTER TABLE fixed_costs
    ADD COLUMN ended_from DATE CHECK (ended_from = date_trunc('month', ended_from)::date);

-- Each existing cost keeps its amount from the month it was created, in Santiago.
INSERT INTO fixed_cost_versions (fixed_cost_id, valid_from, monthly_amount)
SELECT id, date_trunc('month', created_at AT TIME ZONE 'America/Santiago')::date, monthly_amount
FROM fixed_costs;

-- The ones switched off end in the month they were last updated.
UPDATE fixed_costs
SET ended_from = date_trunc('month', updated_at AT TIME ZONE 'America/Santiago')::date
WHERE NOT is_active;

ALTER TABLE fixed_costs
    DROP COLUMN monthly_amount,
    DROP COLUMN is_active;
