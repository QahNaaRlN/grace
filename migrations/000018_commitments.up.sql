ALTER TABLE payment_instruments ADD UNIQUE(id,account_id,owner_id);
CREATE TABLE subscriptions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 160),
 amount_minor bigint NOT NULL CHECK (amount_minor BETWEEN 1 AND 9007199254740991),
 currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
 period text NOT NULL CHECK (period IN ('weekly','monthly','yearly')),
 starts_on date NOT NULL,
 ends_on date CHECK (ends_on >= starts_on),
 account_id uuid,
 payment_instrument_id uuid,
 paused boolean NOT NULL DEFAULT false,
 reminders boolean NOT NULL DEFAULT false,
 revision integer NOT NULL DEFAULT 1,
 creation_key text NOT NULL CHECK(length(creation_key) BETWEEN 1 AND 160),
 request_hash char(64) NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(id,owner_id),
 UNIQUE(owner_id,creation_key),
 FOREIGN KEY(account_id,owner_id) REFERENCES accounts(id,owner_id),
 CHECK(payment_instrument_id IS NULL OR account_id IS NOT NULL),
 FOREIGN KEY(payment_instrument_id,account_id,owner_id) REFERENCES payment_instruments(id,account_id,owner_id)
);
CREATE TABLE subscription_expenses (
 subscription_id uuid NOT NULL,
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 financial_event_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(financial_event_id),
 FOREIGN KEY(subscription_id,owner_id) REFERENCES subscriptions(id,owner_id),
 FOREIGN KEY(financial_event_id,owner_id) REFERENCES financial_events(id,owner_id)
);
CREATE TABLE loan_details (
 obligation_id uuid PRIMARY KEY,
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 lender text NOT NULL DEFAULT '' CHECK(length(lender)<=160),
 agreement_number text NOT NULL DEFAULT '' CHECK(length(agreement_number)<=160),
 agreement_on date,
 principal_minor bigint CHECK(principal_minor BETWEEN 1 AND 9007199254740991),
 annual_rate_basis_points integer CHECK(annual_rate_basis_points BETWEEN 0 AND 100000),
 reminders boolean NOT NULL DEFAULT false,
 revision integer NOT NULL DEFAULT 1,
 FOREIGN KEY(obligation_id,owner_id) REFERENCES obligations(id,owner_id)
);
CREATE TABLE loan_documents (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 obligation_id uuid NOT NULL,
 filename text NOT NULL CHECK(length(filename) BETWEEN 1 AND 180),
 version integer NOT NULL CHECK(version>0),
 sha256 char(64) NOT NULL,
 content bytea NOT NULL CHECK(octet_length(content) BETWEEN 5 AND 16777216),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(obligation_id,version),
 FOREIGN KEY(obligation_id,owner_id) REFERENCES obligations(id,owner_id)
);
ALTER TABLE scheduled_payments ADD COLUMN plan_active boolean NOT NULL DEFAULT true;
ALTER TABLE scheduled_payments DROP CONSTRAINT scheduled_payments_obligation_id_position_key;
ALTER TABLE scheduled_payments DROP CONSTRAINT scheduled_payments_obligation_id_due_on_key;
CREATE UNIQUE INDEX scheduled_active_position ON scheduled_payments(obligation_id,position) WHERE plan_active;
CREATE UNIQUE INDEX scheduled_active_due ON scheduled_payments(obligation_id,due_on) WHERE plan_active;
CREATE TABLE payment_schedule_drafts (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 obligation_id uuid NOT NULL,
 base_hash char(64) NOT NULL,
 payments jsonb NOT NULL CHECK(jsonb_typeof(payments)='array'),
 revision integer NOT NULL DEFAULT 1,
 applied_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(obligation_id,owner_id) REFERENCES obligations(id,owner_id)
);
CREATE TABLE payment_reminders (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 source_kind text NOT NULL CHECK(source_kind IN ('subscription','loan')),
 source_id uuid NOT NULL,
 due_on date NOT NULL,
 lead_days integer NOT NULL CHECK(lead_days IN (1,3)),
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','dispatching','sent','cancelled','uncertain')),
 dispatched_at timestamptz,
 sent_at timestamptz,
 UNIQUE(owner_id,source_kind,source_id,due_on,lead_days)
);
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['subscriptions','subscription_expenses','loan_details','loan_documents','payment_schedule_drafts','payment_reminders'] LOOP
  EXECUTE format('CREATE INDEX %I ON %I(owner_id)',t||'_owner_idx',t);
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id())',t);
 END LOOP;
END $$;
GRANT SELECT,INSERT,UPDATE ON subscriptions,loan_details,payment_schedule_drafts,payment_reminders TO grace_financial;
GRANT SELECT,INSERT ON subscription_expenses,loan_documents TO grace_financial;
