-- SPEC-0059 R: free child rows do not create zero-value purchases or postings.
ALTER TABLE receipt_items
 DROP CONSTRAINT receipt_items_unit_price_minor_check,
 DROP CONSTRAINT receipt_items_line_total_minor_check,
 ADD CONSTRAINT receipt_items_unit_price_minor_check CHECK (unit_price_minor >= 0),
 ADD CONSTRAINT receipt_items_line_total_minor_check CHECK (line_total_minor >= 0);
