DO $$ BEGIN
    IF EXISTS(SELECT 1 FROM financial_events WHERE source::text='import') OR EXISTS(SELECT 1 FROM import_confirmations) THEN
        RAISE EXCEPTION 'Cannot remove import provenance after publication';
    END IF;
END $$;
DROP TABLE import_requests,import_provenance,import_confirmations,import_previews,import_decisions,import_records,import_batches;
DROP FUNCTION grace_import_receipt_immutable();
DROP FUNCTION grace_import_published_immutable();
ALTER TYPE financial_event_source RENAME TO financial_event_source_with_import;
CREATE TYPE financial_event_source AS ENUM ('manual','bank_import','receipt');
ALTER TABLE financial_events ALTER COLUMN source TYPE financial_event_source USING source::text::financial_event_source;
DROP TYPE financial_event_source_with_import;
