-- SPEC-0053: loan repayment type, principal/interest/other parts of schedule
-- rows, owner allocations of settlements, prepayments and bank debt snapshots.
ALTER TABLE loan_details ADD COLUMN repayment_type text NOT NULL DEFAULT 'unknown'
 CHECK(repayment_type IN ('annuity','differentiated','unknown'));
ALTER TABLE scheduled_payments
 ADD COLUMN principal_minor bigint,
 ADD COLUMN interest_minor bigint,
 ADD COLUMN other_minor bigint,
 ADD COLUMN parts_source text,
 ADD CONSTRAINT scheduled_payments_parts_check CHECK(
  (principal_minor IS NULL AND interest_minor IS NULL AND other_minor IS NULL AND parts_source IS NULL)
  OR (principal_minor >= 0 AND interest_minor >= 0 AND other_minor >= 0
   AND principal_minor + interest_minor + other_minor = amount_minor
   AND parts_source IN ('bank','calculated')));
-- The part of an operation that paid a schedule row, split by the owner.
CREATE TABLE loan_settlement_allocations (
 settlement_id uuid PRIMARY KEY,
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 principal_minor bigint NOT NULL CHECK(principal_minor BETWEEN 0 AND 9007199254740991),
 interest_minor bigint NOT NULL CHECK(interest_minor BETWEEN 0 AND 9007199254740991),
 other_minor bigint NOT NULL CHECK(other_minor BETWEEN 0 AND 9007199254740991),
 revision integer NOT NULL DEFAULT 1 CHECK(revision>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(settlement_id,owner_id) REFERENCES scheduled_payment_settlements(id,owner_id)
);
-- A payment (or its remainder after a schedule row) that repays the loan early.
CREATE TABLE loan_prepayments (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 obligation_id uuid NOT NULL,
 financial_event_id uuid NOT NULL,
 amount_minor bigint NOT NULL CHECK(amount_minor BETWEEN 1 AND 9007199254740991),
 principal_minor bigint NOT NULL CHECK(principal_minor >= 0),
 interest_minor bigint NOT NULL CHECK(interest_minor >= 0),
 other_minor bigint NOT NULL CHECK(other_minor >= 0),
 revision integer NOT NULL DEFAULT 1 CHECK(revision>0),
 creation_key text NOT NULL CHECK(length(creation_key) BETWEEN 1 AND 160),
 request_hash char(64) NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(id,owner_id),
 UNIQUE(owner_id,financial_event_id),
 UNIQUE(owner_id,creation_key),
 CHECK(principal_minor + interest_minor + other_minor = amount_minor),
 FOREIGN KEY(obligation_id,owner_id) REFERENCES obligations(id,owner_id),
 FOREIGN KEY(financial_event_id,owner_id) REFERENCES financial_events(id,owner_id)
);
-- Every change of an allocation keeps the replaced values (append-only).
CREATE TABLE loan_allocation_revisions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 financial_event_id uuid NOT NULL,
 part text NOT NULL CHECK(part IN ('schedule','prepayment')),
 previous jsonb,
 current jsonb,
 changed_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(financial_event_id,owner_id) REFERENCES financial_events(id,owner_id)
);
CREATE TABLE loan_debt_snapshots (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 obligation_id uuid NOT NULL,
 taken_on date NOT NULL,
 principal_minor bigint NOT NULL CHECK(principal_minor BETWEEN 0 AND 9007199254740991),
 accrued_interest_minor bigint CHECK(accrued_interest_minor BETWEEN 0 AND 9007199254740991),
 payoff_minor bigint CHECK(payoff_minor BETWEEN 0 AND 9007199254740991),
 note text NOT NULL DEFAULT '' CHECK(length(note)<=500),
 revision integer NOT NULL DEFAULT 1 CHECK(revision>0),
 creation_key text NOT NULL CHECK(length(creation_key) BETWEEN 1 AND 160),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(owner_id,obligation_id,taken_on),
 UNIQUE(owner_id,creation_key),
 FOREIGN KEY(obligation_id,owner_id) REFERENCES obligations(id,owner_id)
);
CREATE INDEX loan_prepayments_obligation_idx ON loan_prepayments(obligation_id);
CREATE INDEX loan_allocation_revisions_event_idx ON loan_allocation_revisions(financial_event_id,changed_at);
CREATE INDEX loan_debt_snapshots_obligation_idx ON loan_debt_snapshots(obligation_id,taken_on DESC);
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['loan_settlement_allocations','loan_prepayments','loan_allocation_revisions','loan_debt_snapshots'] LOOP
  EXECUTE format('CREATE INDEX %I ON %I(owner_id)',t||'_owner_idx',t);
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id())',t);
 END LOOP;
END $$;
GRANT SELECT,INSERT,UPDATE,DELETE ON loan_settlement_allocations TO grace_financial;
GRANT SELECT,INSERT,UPDATE ON loan_prepayments,loan_debt_snapshots TO grace_financial;
GRANT SELECT,INSERT ON loan_allocation_revisions TO grace_financial;
