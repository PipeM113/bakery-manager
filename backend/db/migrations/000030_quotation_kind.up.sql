-- A quotation is for a person (the default: only final prices are shown) or for a company (the detail and
-- the conditions of the order are shown). The conditions belong to the company ones only.
ALTER TABLE quotations
    ADD COLUMN kind          text     NOT NULL DEFAULT 'person' CHECK (kind IN ('person', 'company')),
    ADD COLUMN delivery_date date,
    ADD COLUMN payment_terms text     NOT NULL DEFAULT '' CHECK (char_length(payment_terms) <= 120),
    ADD COLUMN valid_days    smallint CHECK (valid_days BETWEEN 1 AND 90),
    ADD CONSTRAINT quotations_conditions_only_for_companies
        CHECK (kind = 'company' OR (delivery_date IS NULL AND payment_terms = '' AND valid_days IS NULL));
