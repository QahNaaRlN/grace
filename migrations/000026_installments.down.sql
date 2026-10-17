DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM installment_plans) OR EXISTS(SELECT 1 FROM installment_commands)
  OR EXISTS(SELECT 1 FROM installment_allocation_revisions) OR EXISTS(SELECT 1 FROM credit_card_snapshots WHERE installments_included)
  OR EXISTS(SELECT 1 FROM payment_reminders WHERE source_kind='installment') THEN
  RAISE EXCEPTION 'Rollback refused: installment history exists'; END IF;
END $$;
DROP TRIGGER installment_legacy_guard ON scheduled_payment_settlements;
DROP TRIGGER installment_account_guard ON accounts;
DROP TRIGGER installment_schedule_total ON scheduled_payments;
DROP TRIGGER installment_plan_total ON installment_plans;
DROP TRIGGER installment_schedule_guard ON scheduled_payments;
DROP TRIGGER installment_posting_guard ON postings;
DROP TRIGGER installment_event_guard ON financial_events;
DROP TRIGGER installment_allocation_audit ON installment_payment_allocations;
DROP TRIGGER installment_allocation_guard ON installment_payment_allocations;
DROP TRIGGER installment_plan_guard ON installment_plans;
DROP FUNCTION grace_installment_legacy_guard(),grace_installment_schedule_guard(),grace_installment_posting_guard(),grace_installment_event_guard(),grace_installment_allocation_audit(),grace_installment_allocation_guard(),grace_installment_plan_guard();
DROP FUNCTION grace_installment_account_guard(),grace_installment_schedule_total();
ALTER TABLE credit_card_snapshots DROP COLUMN installment_request_hash,DROP COLUMN installments_included,DROP COLUMN minimum_in_grace,DROP COLUMN period_from,DROP COLUMN confirmed_through;
ALTER TABLE payment_reminders DROP CONSTRAINT payment_reminders_source_kind_check;
ALTER TABLE payment_reminders ADD CONSTRAINT payment_reminders_source_kind_check CHECK(source_kind IN ('subscription','loan','credit_card'));
DROP TABLE installment_commands,installment_schedule_drafts,installment_allocation_revisions,installment_payment_allocations,installment_allocation_batches,installment_plans;
CREATE UNIQUE INDEX scheduled_active_due ON scheduled_payments(obligation_id,due_on) WHERE plan_active;
