ALTER TABLE accounts ALTER COLUMN opening_balance_minor DROP NOT NULL;
ALTER TABLE accounts ADD COLUMN balance_reconciled boolean NOT NULL DEFAULT true;
ALTER TABLE accounts ADD CONSTRAINT accounts_reconciled_opening CHECK (NOT balance_reconciled OR opening_balance_minor IS NOT NULL);
