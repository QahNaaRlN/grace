-- Reconciliations and balance adjustments are financial history; refuse to
-- drop them. An enum value cannot be removed, so 'balance_adjustment' stays.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM account_reconciliations)
       OR EXISTS (SELECT 1 FROM payment_instruments WHERE disabled_at IS NOT NULL) THEN
        RAISE EXCEPTION 'Cannot roll back account reconciliation: reconciliations or disabled cards exist';
    END IF;
END $$;
DROP TABLE account_reconciliations;
REVOKE UPDATE (name, bank, last4, disabled_at) ON payment_instruments FROM grace_financial;
ALTER TABLE payment_instruments DROP COLUMN disabled_at;
ALTER TABLE accounts DROP COLUMN revision;
