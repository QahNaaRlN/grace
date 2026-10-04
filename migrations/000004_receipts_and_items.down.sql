DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM receipts) THEN
        RAISE EXCEPTION 'rollback refused: confirmed receipts exist';
    END IF;
END $$;

DROP TABLE receipt_items;
DROP TABLE receipts;
