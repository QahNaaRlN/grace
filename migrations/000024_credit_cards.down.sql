-- Refuse a rollback that would discard card terms, snapshots or documents.
DO $$ BEGIN IF EXISTS(SELECT 1 FROM credit_card_terms) OR EXISTS(SELECT 1 FROM credit_card_snapshots)
 OR EXISTS(SELECT 1 FROM documents) OR EXISTS(SELECT 1 FROM credit_card_settings)
 OR EXISTS(SELECT 1 FROM payment_reminders WHERE source_kind='credit_card') THEN
 RAISE EXCEPTION 'Cannot roll back retained credit card data'; END IF; END $$;
ALTER TABLE payment_reminders DROP CONSTRAINT payment_reminders_source_kind_check;
ALTER TABLE payment_reminders ADD CONSTRAINT payment_reminders_source_kind_check
 CHECK(source_kind IN ('subscription','loan'));
DROP TABLE credit_card_settings,credit_card_snapshots,credit_card_term_revisions,credit_card_terms,documents;
