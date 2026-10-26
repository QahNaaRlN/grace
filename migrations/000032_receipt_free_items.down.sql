-- Retain confirmed free items; rollback must never rewrite financial history.
DO $$ BEGIN
 LOCK TABLE receipt_items IN ACCESS EXCLUSIVE MODE;
 IF EXISTS (SELECT 1 FROM receipt_items WHERE unit_price_minor = 0 OR line_total_minor = 0) THEN
  RAISE EXCEPTION 'Retain free receipt items; keep a compatible schema and API';
 END IF;
 ALTER TABLE receipt_items
  DROP CONSTRAINT receipt_items_unit_price_minor_check,
  DROP CONSTRAINT receipt_items_line_total_minor_check,
  ADD CONSTRAINT receipt_items_unit_price_minor_check CHECK (unit_price_minor > 0),
  ADD CONSTRAINT receipt_items_line_total_minor_check CHECK (line_total_minor > 0);
END $$;
