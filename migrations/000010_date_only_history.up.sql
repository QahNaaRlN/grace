ALTER TABLE financial_events ADD COLUMN occurred_date DATE;
ALTER TABLE financial_events ALTER COLUMN occurred_at DROP NOT NULL;
ALTER TABLE financial_events ADD CONSTRAINT event_occurrence_exactly_one
CHECK ((occurred_at IS NOT NULL AND occurred_date IS NULL)
    OR (occurred_at IS NULL AND occurred_date IS NOT NULL));
