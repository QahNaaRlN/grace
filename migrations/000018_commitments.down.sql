-- Refuse destructive rollback if a replaced schedule exists. Historical rows must be retained.
DO $$ BEGIN IF EXISTS(SELECT 1 FROM scheduled_payments WHERE NOT plan_active)
 OR EXISTS(SELECT 1 FROM subscriptions) OR EXISTS(SELECT 1 FROM loan_documents)
 OR EXISTS(SELECT 1 FROM loan_details) OR EXISTS(SELECT 1 FROM payment_schedule_drafts)
 OR EXISTS(SELECT 1 FROM payment_reminders) THEN
 RAISE EXCEPTION 'Cannot roll back retained schedule history'; END IF; END $$;
DROP TABLE payment_reminders,payment_schedule_drafts,loan_documents,loan_details,subscription_expenses,subscriptions;
ALTER TABLE payment_instruments DROP CONSTRAINT payment_instruments_id_account_id_owner_id_key;
DROP INDEX scheduled_active_position;
DROP INDEX scheduled_active_due;
ALTER TABLE scheduled_payments DROP COLUMN plan_active;
ALTER TABLE scheduled_payments ADD UNIQUE(obligation_id,position), ADD UNIQUE(obligation_id,due_on);
