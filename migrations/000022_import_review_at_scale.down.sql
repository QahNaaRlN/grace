DROP INDEX import_records_batch_review_idx;
ALTER TABLE import_records DROP COLUMN questions;
ALTER TABLE import_batches DROP COLUMN format, DROP COLUMN file_name;
