-- SPEC-0044: the import history shows the file and its format, and review
-- filters by «questions» (source issues + match candidates) without scanning
-- the whole batch report for every row.
ALTER TABLE import_batches
    ADD COLUMN file_name TEXT CHECK (length(file_name) BETWEEN 1 AND 255),
    ADD COLUMN format TEXT CHECK (format IN ('json', 'csv'));

ALTER TABLE import_records ADD COLUMN questions INTEGER NOT NULL DEFAULT 0 CHECK (questions >= 0);
-- Published rows are immutable (trigger import_published_immutable) and are
-- never «with questions»; only rows still under review need the count.
-- Candidates are expanded once per batch, not once per row.
UPDATE import_records SET questions = jsonb_array_length(COALESCE(source->'issues', '[]'::jsonb))
WHERE NOT published;
UPDATE import_records r SET questions = r.questions + involved.n
FROM (
    SELECT b.id AS batch_id, x.record_id, count(*) AS n
    FROM import_batches b
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(b.report->'candidates', '[]'::jsonb)) c
    CROSS JOIN LATERAL (VALUES (c->>'recordId'),
        (CASE WHEN c->>'otherRecordId' IS DISTINCT FROM c->>'recordId' THEN c->>'otherRecordId' END)) x(record_id)
    WHERE x.record_id IS NOT NULL
    GROUP BY b.id, x.record_id
) involved
WHERE NOT r.published AND r.batch_id = involved.batch_id AND r.snapshot_record_id = involved.record_id;

CREATE INDEX import_records_batch_review_idx ON import_records (batch_id, published, (decision->>'action'));
