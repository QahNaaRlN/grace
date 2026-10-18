DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM statement_sources) OR EXISTS(SELECT 1 FROM statement_audit) OR EXISTS(SELECT 1 FROM documents WHERE kind='bank_statement') OR EXISTS(SELECT 1 FROM import_batches WHERE format='bank_statement_pdf')
 THEN RAISE EXCEPTION 'Cannot downgrade bank statement evidence or tombstones'; END IF;
END $$;
DROP TRIGGER statement_tombstone_original ON documents;
DROP FUNCTION grace_statement_tombstone_original();
DROP FUNCTION grace_statement_rebind(uuid,uuid);
DROP FUNCTION grace_statement_prepare_regroup(uuid,jsonb,text);
DROP TABLE statement_audit;
DROP TABLE statement_sources;
DROP FUNCTION grace_statement_source_guard();
DROP TRIGGER document_original_guard ON documents;
DROP FUNCTION grace_document_original_guard();
REVOKE UPDATE(content,original_deleted_at) ON documents FROM grace_financial;
DROP INDEX documents_bank_statement_sha_idx;
ALTER TABLE documents DROP CONSTRAINT documents_original_state;
ALTER TABLE documents DROP COLUMN original_deleted_at;
ALTER TABLE documents ALTER COLUMN content SET NOT NULL;
ALTER TABLE documents DROP CONSTRAINT documents_kind_check;
ALTER TABLE documents ADD CONSTRAINT documents_kind_check CHECK(kind IN ('credit_card_terms'));
ALTER TABLE import_batches DROP CONSTRAINT import_batches_format_check;
ALTER TABLE import_batches ADD CONSTRAINT import_batches_format_check CHECK(format IN ('json','csv'));
