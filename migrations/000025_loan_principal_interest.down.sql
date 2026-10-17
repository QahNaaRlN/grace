-- Refuse a rollback that would discard loan parts, allocations, prepayments or snapshots.
DO $$ BEGIN IF EXISTS(SELECT 1 FROM loan_settlement_allocations) OR EXISTS(SELECT 1 FROM loan_prepayments)
 OR EXISTS(SELECT 1 FROM loan_allocation_revisions) OR EXISTS(SELECT 1 FROM loan_debt_snapshots)
 OR EXISTS(SELECT 1 FROM scheduled_payments WHERE parts_source IS NOT NULL)
 OR EXISTS(SELECT 1 FROM loan_details WHERE repayment_type<>'unknown') THEN
 RAISE EXCEPTION 'Cannot roll back retained loan principal and interest data'; END IF; END $$;
DROP TABLE loan_debt_snapshots,loan_allocation_revisions,loan_prepayments,loan_settlement_allocations;
ALTER TABLE scheduled_payments DROP CONSTRAINT scheduled_payments_parts_check,
 DROP COLUMN parts_source, DROP COLUMN other_minor, DROP COLUMN interest_minor, DROP COLUMN principal_minor;
ALTER TABLE loan_details DROP COLUMN repayment_type;
