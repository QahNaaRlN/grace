-- SPEC-0044: the import history shows the file and its format, and review
-- filters by «questions» (source issues + match candidates) without scanning
-- the whole batch report for every row.
ALTER TABLE import_batches
    ADD COLUMN file_name TEXT CHECK (length(file_name) BETWEEN 1 AND 255),
    ADD COLUMN format TEXT CHECK (format IN ('json', 'csv'));

ALTER TABLE import_records ADD COLUMN questions INTEGER NOT NULL DEFAULT 0 CHECK (questions >= 0);
UPDATE import_records r SET questions = jsonb_array_length(COALESCE(r.source->'issues', '[]'::jsonb)) + (
    SELECT count(*) FROM import_batches b, jsonb_array_elements(COALESCE(b.report->'candidates', '[]'::jsonb)) c
    WHERE b.id = r.batch_id AND (c->>'recordId' = r.snapshot_record_id OR c->>'otherRecordId' = r.snapshot_record_id));

CREATE INDEX import_records_batch_review_idx ON import_records (batch_id, published, (decision->>'action'));
