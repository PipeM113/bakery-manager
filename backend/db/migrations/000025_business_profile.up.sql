-- The data of the business that the quotation PDF prints and that the owners edit in the app.
-- The RUT and the bank account number are not here: they come from the environment (BIZ_RUT, BIZ_ACCOUNT).
-- There is one row, born empty; the primary key and its check keep it that way.
CREATE TABLE business_profile (
    id           BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    name         TEXT NOT NULL DEFAULT '',
    phone        TEXT NOT NULL DEFAULT '',
    instagram    TEXT NOT NULL DEFAULT '',
    bank         TEXT NOT NULL DEFAULT '',
    account_type TEXT NOT NULL DEFAULT '',
    holder       TEXT NOT NULL DEFAULT '',
    email        TEXT NOT NULL DEFAULT '',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by   UUID REFERENCES users(id) ON DELETE SET NULL
);
ALTER TABLE business_profile ENABLE ROW LEVEL SECURITY;
INSERT INTO business_profile DEFAULT VALUES;
