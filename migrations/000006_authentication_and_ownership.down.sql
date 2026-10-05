-- Never flatten financial data from multiple users into an owner-less schema.
DO $$
DECLARE t TEXT; unsafe BOOLEAN;
BEGIN
    FOREACH t IN ARRAY ARRAY['accounts','financial_events','postings','receipts',
        'receipt_items','obligations','scheduled_payments','scheduled_payment_settlements']
    LOOP
        EXECUTE format('SELECT EXISTS(SELECT 1 FROM %I WHERE owner_id <> %L::uuid)',
            t, '00000000-0000-4000-8000-000000000001') INTO unsafe;
        IF unsafe THEN RAISE EXCEPTION 'unsafe downgrade: another owner has financial data'; END IF;
    END LOOP;
END $$;

ALTER TABLE postings DROP CONSTRAINT postings_event_owner_fk, DROP CONSTRAINT postings_account_owner_fk;
ALTER TABLE financial_events DROP CONSTRAINT events_original_owner_fk;
ALTER TABLE receipts DROP CONSTRAINT receipts_purchase_owner_fk;
ALTER TABLE receipt_items DROP CONSTRAINT receipt_items_owner_fk;
ALTER TABLE scheduled_payments DROP CONSTRAINT scheduled_payments_owner_fk;
ALTER TABLE scheduled_payment_settlements DROP CONSTRAINT settlements_payment_owner_fk, DROP CONSTRAINT settlements_event_owner_fk;

DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['accounts','financial_events','postings','receipts',
        'receipt_items','obligations','scheduled_payments','scheduled_payment_settlements']
    LOOP
        EXECUTE format('DROP POLICY owner_access ON %I', t);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('REVOKE ALL ON %I FROM grace_financial', t);
        EXECUTE format('ALTER TABLE %I DROP COLUMN owner_id', t);
    END LOOP;
END $$;
ALTER TABLE financial_events ADD UNIQUE(kind, idempotency_key);
ALTER TABLE receipts ADD UNIQUE(idempotency_key);
ALTER TABLE obligations ADD UNIQUE(idempotency_key);
ALTER TABLE scheduled_payment_settlements ADD UNIQUE(idempotency_key);
REVOKE SELECT ON categories FROM grace_financial;
DROP FUNCTION grace_owner_id();
DROP TABLE auth_sessions, auth_login_flows, auth_invitations, auth_identities, users;
