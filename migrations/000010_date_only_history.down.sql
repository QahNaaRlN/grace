DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM financial_events WHERE occurred_date IS NOT NULL) THEN
        RAISE EXCEPTION 'Cannot discard unknown-time financial history';
    END IF;
END $$;
ALTER TABLE financial_events DROP CONSTRAINT event_occurrence_exactly_one;
ALTER TABLE financial_events ALTER COLUMN occurred_at SET NOT NULL;
ALTER TABLE financial_events DROP COLUMN occurred_date;
