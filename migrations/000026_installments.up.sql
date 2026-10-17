-- SPEC-0054: explicit financed purchases and transfer allocations. Legacy schedules
-- remain legacy until a separate, reviewed migration is implemented.
DROP INDEX scheduled_active_due;
CREATE TABLE installment_plans (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 obligation_id uuid NOT NULL, account_id uuid NOT NULL, purchase_event_id uuid,
 mode text NOT NULL CHECK(mode IN ('standalone','credit_card')),
 provider text NOT NULL CHECK(length(trim(provider)) BETWEEN 1 AND 200),
 contract_number text NOT NULL DEFAULT '' CHECK(length(contract_number)<=200),
 opened_on date NOT NULL, observed_on date NOT NULL,
 principal_minor bigint NOT NULL CHECK(principal_minor BETWEEN 1 AND 9007199254740991),
 initial_remaining_minor bigint NOT NULL CHECK(initial_remaining_minor BETWEEN 1 AND principal_minor),
 historical boolean NOT NULL DEFAULT false, reminders boolean NOT NULL DEFAULT false,
 revision integer NOT NULL DEFAULT 1 CHECK(revision>0),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(id,owner_id), UNIQUE(obligation_id,owner_id), UNIQUE(purchase_event_id,owner_id),
 CHECK(observed_on>=opened_on), CHECK(historical OR (observed_on=opened_on AND initial_remaining_minor=principal_minor AND purchase_event_id IS NOT NULL)),
 FOREIGN KEY(obligation_id,owner_id) REFERENCES obligations(id,owner_id),
 FOREIGN KEY(account_id,owner_id) REFERENCES accounts(id,owner_id),
 FOREIGN KEY(purchase_event_id,owner_id) REFERENCES financial_events(id,owner_id)
);
CREATE UNIQUE INDEX installment_standalone_account ON installment_plans(account_id) WHERE mode='standalone';
CREATE INDEX installment_account ON installment_plans(account_id,opened_on);
CREATE TABLE installment_allocation_batches (
 event_id uuid PRIMARY KEY, owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 revision integer NOT NULL DEFAULT 0, reason text NOT NULL DEFAULT '' CHECK(length(reason)<=500),
 UNIQUE(event_id,owner_id), FOREIGN KEY(event_id,owner_id) REFERENCES financial_events(id,owner_id)
);
CREATE TABLE installment_payment_allocations (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 event_id uuid NOT NULL, payment_id uuid NOT NULL, amount_minor bigint NOT NULL CHECK(amount_minor BETWEEN 1 AND 9007199254740991),
 active boolean NOT NULL DEFAULT true, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(event_id,payment_id),
 FOREIGN KEY(event_id,owner_id) REFERENCES installment_allocation_batches(event_id,owner_id),
 FOREIGN KEY(payment_id,owner_id) REFERENCES scheduled_payments(id,owner_id)
);
CREATE INDEX installment_allocation_payment ON installment_payment_allocations(payment_id) WHERE active;
CREATE TABLE installment_allocation_revisions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 event_id uuid NOT NULL, previous jsonb, current jsonb, reason text NOT NULL,
 changed_at timestamptz NOT NULL DEFAULT now(), FOREIGN KEY(event_id,owner_id) REFERENCES financial_events(id,owner_id)
);
CREATE TABLE installment_schedule_drafts (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 plan_id uuid NOT NULL, revision integer NOT NULL DEFAULT 1, plan_revision integer NOT NULL,
 payments jsonb NOT NULL, applied boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(plan_id,owner_id) REFERENCES installment_plans(id,owner_id)
);
CREATE TABLE installment_commands (
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 key text NOT NULL CHECK(length(key) BETWEEN 1 AND 160), request_hash text NOT NULL,
 response jsonb NOT NULL, PRIMARY KEY(owner_id,key)
);
ALTER TABLE credit_card_snapshots
 ADD COLUMN installment_request_hash text NOT NULL DEFAULT '',
 ADD COLUMN installments_included boolean NOT NULL DEFAULT false,
 ADD COLUMN minimum_in_grace boolean NOT NULL DEFAULT false,
 ADD COLUMN period_from date,
 ADD COLUMN confirmed_through timestamptz,
 ADD CONSTRAINT installment_snapshot_period CHECK(NOT installments_included OR
  (period_from IS NOT NULL AND period_from<=taken_on AND grace_amount_minor IS NOT NULL AND grace_due_on IS NOT NULL AND grace_due_on>=taken_on
   AND (minimum_minor IS NULL)=(minimum_due_on IS NULL) AND confirmed_through IS NOT NULL));
ALTER TABLE payment_reminders DROP CONSTRAINT payment_reminders_source_kind_check;
ALTER TABLE payment_reminders ADD CONSTRAINT payment_reminders_source_kind_check CHECK(source_kind IN ('subscription','loan','credit_card','installment'));

DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['installment_plans','installment_allocation_batches','installment_payment_allocations','installment_allocation_revisions','installment_schedule_drafts','installment_commands'] LOOP
  EXECUTE format('CREATE INDEX %I ON %I(owner_id)',t||'_owner_idx',t);
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id())',t);
 END LOOP;
END $$;
GRANT SELECT,INSERT ON installment_commands,installment_allocation_revisions TO grace_financial;
GRANT SELECT,INSERT,UPDATE ON installment_plans,installment_allocation_batches,installment_payment_allocations,installment_schedule_drafts TO grace_financial;

-- Direct SQL uses the same invariants as HTTP. All checks are invoker-scoped;
-- no SECURITY DEFINER function can expose another owner's financial records.
CREATE FUNCTION grace_installment_plan_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE a accounts; o obligations; e financial_events; amount bigint; n integer;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('ledger:'||NEW.owner_id::text,0));
 SELECT * INTO STRICT a FROM accounts WHERE id=NEW.account_id AND owner_id=NEW.owner_id FOR UPDATE;
 SELECT * INTO STRICT o FROM obligations WHERE id=NEW.obligation_id AND owner_id=NEW.owner_id FOR UPDATE;
 IF o.kind<>'installment' OR o.currency<>a.currency OR o.total_minor<>NEW.initial_remaining_minor
  OR (NEW.mode='standalone' AND a.type<>'installment') OR (NEW.mode='credit_card' AND a.type<>'credit') THEN
  RAISE EXCEPTION 'Invalid installment account or obligation' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND (NEW.account_id,NEW.obligation_id,NEW.purchase_event_id,NEW.mode,NEW.opened_on,NEW.observed_on,NEW.principal_minor,NEW.initial_remaining_minor,NEW.historical)
  IS DISTINCT FROM (OLD.account_id,OLD.obligation_id,OLD.purchase_event_id,OLD.mode,OLD.opened_on,OLD.observed_on,OLD.principal_minor,OLD.initial_remaining_minor,OLD.historical) THEN
  RAISE EXCEPTION 'Installment basis is immutable' USING ERRCODE='23514'; END IF;
 IF EXISTS(SELECT 1 FROM scheduled_payment_settlements s JOIN scheduled_payments p ON p.id=s.scheduled_payment_id WHERE p.obligation_id=o.id) THEN
  RAISE EXCEPTION 'Legacy settlements require a reviewed migration' USING ERRCODE='23514'; END IF;
 IF NEW.purchase_event_id IS NOT NULL THEN
  SELECT * INTO STRICT e FROM financial_events WHERE id=NEW.purchase_event_id AND owner_id=NEW.owner_id;
  SELECT count(*),COALESCE(sum(amount_minor),0) INTO n,amount FROM postings WHERE financial_event_id=e.id AND account_id=a.id AND direction='debit';
  IF e.kind<>'purchase' OR e.voided_at IS NOT NULL OR n<>1 OR amount<>NEW.principal_minor
   OR COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<>NEW.opened_on
   OR EXISTS(SELECT 1 FROM financial_events WHERE original_event_id=e.id AND voided_at IS NULL) THEN
   RAISE EXCEPTION 'Purchase does not match installment' USING ERRCODE='23514'; END IF;
 END IF;
 IF NEW.mode='standalone' AND EXISTS(SELECT 1 FROM postings p JOIN financial_events ev ON ev.id=p.financial_event_id WHERE p.account_id=a.id AND COALESCE(ev.occurred_date,(ev.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<NEW.observed_on) THEN
  RAISE EXCEPTION 'Posting before observation date' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER installment_plan_guard BEFORE INSERT OR UPDATE ON installment_plans FOR EACH ROW EXECUTE FUNCTION grace_installment_plan_guard();

CREATE FUNCTION grace_installment_allocation_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE credit postings; e financial_events; plan installment_plans; p scheduled_payments; total bigint;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('ledger:'||NEW.owner_id::text,0));
 IF TG_OP='UPDATE' AND (NEW.event_id,NEW.payment_id,NEW.owner_id) IS DISTINCT FROM (OLD.event_id,OLD.payment_id,OLD.owner_id) THEN
  RAISE EXCEPTION 'Allocation identity is immutable' USING ERRCODE='23514'; END IF;
 SELECT * INTO STRICT e FROM financial_events WHERE id=NEW.event_id AND owner_id=NEW.owner_id FOR UPDATE;
 SELECT * INTO STRICT credit FROM postings WHERE financial_event_id=e.id AND direction='credit' FOR UPDATE;
 SELECT * INTO STRICT p FROM scheduled_payments WHERE id=NEW.payment_id AND owner_id=NEW.owner_id FOR UPDATE;
 SELECT * INTO STRICT plan FROM installment_plans WHERE obligation_id=p.obligation_id AND owner_id=NEW.owner_id FOR UPDATE;
 IF NOT NEW.active THEN RETURN NEW; END IF;
 IF e.kind<>'internal_transfer' OR e.voided_at IS NOT NULL OR NOT p.plan_active OR credit.account_id<>plan.account_id
  OR (SELECT count(*) FROM postings WHERE financial_event_id=e.id)<>2
  OR COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<plan.observed_on THEN
  RAISE EXCEPTION 'Invalid installment transfer' USING ERRCODE='23514'; END IF;
 SELECT COALESCE(sum(amount_minor),0) INTO total FROM installment_payment_allocations WHERE active AND event_id=e.id AND id<>NEW.id;
 IF total+NEW.amount_minor>credit.amount_minor THEN RAISE EXCEPTION 'Transfer overallocated' USING ERRCODE='23514'; END IF;
 SELECT COALESCE(sum(amount_minor),0) INTO total FROM installment_payment_allocations WHERE active AND payment_id=p.id AND id<>NEW.id;
 IF total+NEW.amount_minor>p.amount_minor THEN RAISE EXCEPTION 'Schedule row overpaid' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER installment_allocation_guard BEFORE INSERT OR UPDATE ON installment_payment_allocations FOR EACH ROW EXECUTE FUNCTION grace_installment_allocation_guard();
CREATE FUNCTION grace_installment_allocation_audit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO installment_allocation_revisions(owner_id,event_id,previous,current,reason)
 SELECT NEW.owner_id,NEW.event_id,CASE WHEN TG_OP='UPDATE' THEN to_jsonb(OLD) ELSE NULL END,to_jsonb(NEW),reason
 FROM installment_allocation_batches WHERE event_id=NEW.event_id;
 RETURN NEW;
END $$;
CREATE TRIGGER installment_allocation_audit AFTER INSERT OR UPDATE ON installment_payment_allocations FOR EACH ROW EXECUTE FUNCTION grace_installment_allocation_audit();

CREATE FUNCTION grace_installment_event_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='UPDATE' AND (NEW.occurred_at,NEW.occurred_date) IS DISTINCT FROM (OLD.occurred_at,OLD.occurred_date) AND
 (EXISTS(SELECT 1 FROM installment_plans WHERE purchase_event_id=NEW.id) OR EXISTS(SELECT 1 FROM installment_payment_allocations WHERE event_id=NEW.id AND active)) THEN
 RAISE EXCEPTION 'Installment event date is linked' USING ERRCODE='23514'; END IF;
 IF EXISTS(SELECT 1 FROM postings p JOIN installment_plans i ON i.account_id=p.account_id WHERE p.financial_event_id=NEW.id AND i.mode='standalone'
 AND COALESCE(NEW.occurred_date,(NEW.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<i.observed_on) THEN
 RAISE EXCEPTION 'Posting predates installment observation' USING ERRCODE='23514'; END IF;
 IF NEW.kind='refund' AND EXISTS(SELECT 1 FROM installment_plans WHERE purchase_event_id=NEW.original_event_id) THEN
  RAISE EXCEPTION 'Refund requires installment plan revision' USING ERRCODE='23514'; END IF;
 IF TG_OP='UPDATE' AND NEW.voided_at IS NOT NULL AND OLD.voided_at IS NULL AND
  (EXISTS(SELECT 1 FROM installment_plans WHERE purchase_event_id=NEW.id) OR EXISTS(SELECT 1 FROM installment_payment_allocations WHERE event_id=NEW.id AND active)) THEN
  RAISE EXCEPTION 'Installment dependency prevents void' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER installment_event_guard BEFORE INSERT OR UPDATE ON financial_events FOR EACH ROW EXECUTE FUNCTION grace_installment_event_guard();
CREATE FUNCTION grace_installment_posting_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM installment_plans i JOIN financial_events e ON e.id=NEW.financial_event_id
  WHERE i.account_id=NEW.account_id AND i.mode='standalone' AND COALESCE(e.occurred_date,(e.occurred_at AT TIME ZONE 'Asia/Yekaterinburg')::date)<i.observed_on) THEN
  RAISE EXCEPTION 'Posting predates installment observation' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER installment_posting_guard BEFORE INSERT ON postings FOR EACH ROW EXECUTE FUNCTION grace_installment_posting_guard();

CREATE FUNCTION grace_installment_schedule_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.plan_active AND NOT EXISTS(SELECT 1 FROM installment_plans WHERE obligation_id=NEW.obligation_id) AND EXISTS(SELECT 1 FROM scheduled_payments WHERE obligation_id=NEW.obligation_id AND due_on=NEW.due_on AND plan_active AND id<>NEW.id) THEN
 RAISE EXCEPTION 'Legacy schedule dates must be unique' USING ERRCODE='23505'; END IF;
 IF TG_OP='UPDATE' AND EXISTS(SELECT 1 FROM installment_payment_allocations WHERE payment_id=OLD.id AND active)
  AND (NEW.due_on,NEW.amount_minor,NEW.position,NEW.plan_active) IS DISTINCT FROM (OLD.due_on,OLD.amount_minor,OLD.position,OLD.plan_active) THEN
  RAISE EXCEPTION 'Paid installment rows are immutable' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER installment_schedule_guard BEFORE INSERT OR UPDATE ON scheduled_payments FOR EACH ROW EXECUTE FUNCTION grace_installment_schedule_guard();
CREATE FUNCTION grace_installment_legacy_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM installment_plans i JOIN scheduled_payments p ON p.obligation_id=i.obligation_id WHERE p.id=NEW.scheduled_payment_id) THEN
  RAISE EXCEPTION 'Installment plan uses transfer allocations' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER installment_legacy_guard BEFORE INSERT ON scheduled_payment_settlements FOR EACH ROW EXECUTE FUNCTION grace_installment_legacy_guard();

CREATE FUNCTION grace_installment_schedule_total() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE plan installment_plans; oid uuid; total bigint;
BEGIN
 IF TG_TABLE_NAME='installment_plans' THEN oid:=NEW.obligation_id; ELSE oid:=NEW.obligation_id; END IF;
 SELECT * INTO plan FROM installment_plans WHERE obligation_id=oid;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT COALESCE(sum(amount_minor),0) INTO total FROM scheduled_payments WHERE obligation_id=oid AND plan_active;
 IF total<>plan.initial_remaining_minor OR EXISTS(SELECT 1 FROM scheduled_payments WHERE obligation_id=oid AND plan_active
  AND (due_on<plan.observed_on OR principal_minor IS NOT NULL OR interest_minor IS NOT NULL OR other_minor IS NOT NULL)) THEN
  RAISE EXCEPTION 'Installment schedule must equal observed principal' USING ERRCODE='23514'; END IF;
 RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER installment_schedule_total AFTER INSERT OR UPDATE ON scheduled_payments DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION grace_installment_schedule_total();
CREATE CONSTRAINT TRIGGER installment_plan_total AFTER INSERT OR UPDATE ON installment_plans DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION grace_installment_schedule_total();
CREATE FUNCTION grace_installment_account_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (NEW.type,NEW.currency) IS DISTINCT FROM (OLD.type,OLD.currency) AND EXISTS(SELECT 1 FROM installment_plans WHERE account_id=OLD.id) THEN
 RAISE EXCEPTION 'Installment account type and currency are immutable' USING ERRCODE='23514'; END IF;
 IF (NEW.opening_balance_minor,NEW.balance_reconciled,NEW.type,NEW.currency) IS DISTINCT FROM (OLD.opening_balance_minor,OLD.balance_reconciled,OLD.type,OLD.currency)
  AND EXISTS(SELECT 1 FROM installment_plans WHERE account_id=OLD.id AND mode='standalone') THEN
  RAISE EXCEPTION 'Installment account basis is immutable' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER installment_account_guard BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION grace_installment_account_guard();
