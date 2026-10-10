ALTER TABLE receipts ADD COLUMN issued_date DATE;
ALTER TABLE receipts ALTER COLUMN issued_at DROP NOT NULL;
ALTER TABLE receipts ADD CONSTRAINT receipt_occurrence_exactly_one
CHECK((issued_at IS NOT NULL)<>(issued_date IS NOT NULL));
ALTER TABLE receipt_items ADD COLUMN brand TEXT;
ALTER TABLE receipt_items ADD COLUMN package TEXT;
ALTER TABLE receipt_items ADD COLUMN broad_group TEXT;
