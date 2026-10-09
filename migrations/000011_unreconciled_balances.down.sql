DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM accounts WHERE NOT balance_reconciled OR opening_balance_minor IS NULL) THEN
 RAISE EXCEPTION 'Cannot downgrade with unreconciled accounts; preserve their unknown balances';
 END IF;
END $$;
ALTER TABLE accounts DROP CONSTRAINT accounts_reconciled_opening;
ALTER TABLE accounts DROP COLUMN balance_reconciled;
ALTER TABLE accounts ALTER COLUMN opening_balance_minor SET NOT NULL;
