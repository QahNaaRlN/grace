-- SPEC-0051: credit card terms (versioned, with revision history), bank
-- snapshots, per-card settings and a shared table for source documents.
CREATE TABLE documents (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 kind text NOT NULL CHECK(kind IN ('credit_card_terms')),
 filename text NOT NULL CHECK(length(filename) BETWEEN 1 AND 180),
 sha256 char(64) NOT NULL,
 content bytea NOT NULL CHECK(octet_length(content) BETWEEN 5 AND 16777216),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(id,owner_id)
);
CREATE TABLE credit_card_terms (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 account_id uuid NOT NULL,
 effective_from date NOT NULL,
 limit_minor bigint CHECK(limit_minor BETWEEN 1 AND 9007199254740991),
 purchase_rate_bp integer CHECK(purchase_rate_bp BETWEEN 0 AND 100000),
 cash_rate_bp integer CHECK(cash_rate_bp BETWEEN 0 AND 100000),
 grace_model text NOT NULL CHECK(grace_model IN ('statement','from_first_purchase','none','unknown')),
 statement_day integer CHECK(statement_day BETWEEN 1 AND 31),
 payment_days integer CHECK(payment_days BETWEEN 1 AND 90),
 grace_days integer CHECK(grace_days BETWEEN 1 AND 400),
 min_percent_bp integer CHECK(min_percent_bp BETWEEN 0 AND 10000),
 min_amount_minor bigint CHECK(min_amount_minor BETWEEN 0 AND 9007199254740991),
 min_rounding_minor bigint CHECK(min_rounding_minor BETWEEN 1 AND 100000000),
 min_percent_is_cap boolean NOT NULL DEFAULT false,
 min_includes_interest boolean NOT NULL DEFAULT false,
 document_id uuid,
 note text NOT NULL DEFAULT '' CHECK(length(note)<=500),
 revision integer NOT NULL DEFAULT 1 CHECK(revision>0),
 creation_key text NOT NULL CHECK(length(creation_key) BETWEEN 1 AND 160),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(owner_id,account_id,effective_from),
 UNIQUE(owner_id,creation_key),
 UNIQUE(id,owner_id),
 CHECK(grace_model<>'statement' OR (statement_day IS NOT NULL AND payment_days IS NOT NULL)),
 CHECK(grace_model<>'from_first_purchase' OR grace_days IS NOT NULL),
 FOREIGN KEY(account_id,owner_id) REFERENCES accounts(id,owner_id),
 FOREIGN KEY(document_id,owner_id) REFERENCES documents(id,owner_id)
);
-- Every edit of a version keeps the replaced values (append-only).
CREATE TABLE credit_card_term_revisions (
 terms_id uuid NOT NULL,
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 revision integer NOT NULL,
 previous jsonb NOT NULL,
 changed_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(terms_id,revision),
 FOREIGN KEY(terms_id,owner_id) REFERENCES credit_card_terms(id,owner_id)
);
CREATE TABLE credit_card_snapshots (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 account_id uuid NOT NULL,
 taken_on date NOT NULL,
 debt_minor bigint NOT NULL CHECK(debt_minor BETWEEN -9007199254740991 AND 9007199254740991),
 available_minor bigint CHECK(available_minor BETWEEN 0 AND 9007199254740991),
 grace_amount_minor bigint CHECK(grace_amount_minor BETWEEN 0 AND 9007199254740991),
 grace_due_on date,
 minimum_minor bigint CHECK(minimum_minor BETWEEN 0 AND 9007199254740991),
 minimum_due_on date,
 note text NOT NULL DEFAULT '' CHECK(length(note)<=500),
 creation_key text NOT NULL CHECK(length(creation_key) BETWEEN 1 AND 160),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(owner_id,creation_key),
 FOREIGN KEY(account_id,owner_id) REFERENCES accounts(id,owner_id)
);
CREATE TABLE credit_card_settings (
 account_id uuid PRIMARY KEY,
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 reminders boolean NOT NULL DEFAULT false,
 FOREIGN KEY(account_id,owner_id) REFERENCES accounts(id,owner_id)
);
CREATE INDEX credit_card_terms_account_idx ON credit_card_terms(account_id,effective_from DESC);
CREATE INDEX credit_card_snapshots_account_idx ON credit_card_snapshots(account_id,taken_on DESC,created_at DESC);
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['documents','credit_card_terms','credit_card_term_revisions','credit_card_snapshots','credit_card_settings'] LOOP
  EXECUTE format('CREATE INDEX %I ON %I(owner_id)',t||'_owner_idx',t);
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id())',t);
 END LOOP;
END $$;
GRANT SELECT,INSERT,UPDATE ON credit_card_terms,credit_card_settings TO grace_financial;
GRANT SELECT,INSERT ON documents,credit_card_term_revisions,credit_card_snapshots TO grace_financial;
ALTER TABLE payment_reminders DROP CONSTRAINT payment_reminders_source_kind_check;
ALTER TABLE payment_reminders ADD CONSTRAINT payment_reminders_source_kind_check
 CHECK(source_kind IN ('subscription','loan','credit_card'));
