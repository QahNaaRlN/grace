-- Evidence is retained. Binary rollback is preferred over destructive downgrade.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM statement_sources) OR EXISTS(SELECT 1 FROM receipt_drafts WHERE source_kind='pdf' OR reading_consent IS NOT NULL) THEN RAISE EXCEPTION 'Document evidence prevents downgrade'; END IF;
END $$;
DROP TRIGGER receipt_document_guard ON receipt_drafts;
DROP FUNCTION grace_receipt_document_guard();
ALTER TABLE receipt_drafts DROP COLUMN reading_consent,DROP COLUMN source_pages,DROP COLUMN source_kind;
DROP TRIGGER visual_source_guard ON statement_sources;
DROP FUNCTION grace_visual_source_guard();
ALTER TABLE statement_sources DROP CONSTRAINT visual_original,DROP COLUMN visual_state,DROP COLUMN rendered_pages,DROP COLUMN reading_mode,DROP COLUMN source_kind;
ALTER TABLE statement_extraction_previews DROP CONSTRAINT statement_extraction_previews_maximum_micros_check;
ALTER TABLE statement_extraction_previews ADD CONSTRAINT statement_extraction_previews_maximum_micros_check CHECK(maximum_micros BETWEEN 1 AND 256000);

ALTER TABLE import_batches DROP CONSTRAINT import_batches_format_check;
ALTER TABLE import_batches ADD CONSTRAINT import_batches_format_check CHECK(format IN ('json','csv','bank_statement_pdf'));
