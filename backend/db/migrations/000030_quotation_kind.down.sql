ALTER TABLE quotations
    DROP CONSTRAINT quotations_conditions_only_for_companies,
    DROP COLUMN valid_days,
    DROP COLUMN payment_terms,
    DROP COLUMN delivery_date,
    DROP COLUMN kind;
