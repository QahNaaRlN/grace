DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM receipts WHERE issued_date IS NOT NULL) OR
 EXISTS(SELECT 1 FROM receipt_items WHERE brand IS NOT NULL OR package IS NOT NULL OR broad_group IS NOT NULL)
 THEN RAISE EXCEPTION 'Cannot discard structured receipt history'; END IF;
END $$;
ALTER TABLE receipts DROP CONSTRAINT receipt_occurrence_exactly_one;
ALTER TABLE receipts ALTER COLUMN issued_at SET NOT NULL;
ALTER TABLE receipts DROP COLUMN issued_date;
ALTER TABLE receipt_items DROP COLUMN brand,DROP COLUMN package,DROP COLUMN broad_group;
