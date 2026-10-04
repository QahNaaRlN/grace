DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM obligations) THEN
        RAISE EXCEPTION 'rollback refused: obligations exist';
    END IF;
END $$;

DROP TABLE scheduled_payment_settlements;
DROP TABLE scheduled_payments;
DROP TABLE obligations;
