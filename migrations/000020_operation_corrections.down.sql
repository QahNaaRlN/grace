-- Refuse to forget corrections or voided operations: their history would be lost
-- and voided money would silently disappear from the archive.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM financial_event_revisions) OR EXISTS (SELECT 1 FROM voided_postings)
       OR EXISTS (SELECT 1 FROM financial_events WHERE voided_at IS NOT NULL OR edited_by_user) THEN
        RAISE EXCEPTION 'Cannot roll back operation corrections: corrected or voided operations exist';
    END IF;
END $$;
DROP TRIGGER no_postings_for_voided ON postings;
DROP FUNCTION grace_no_postings_for_voided();
DROP TRIGGER voided_event_immutable ON financial_events;
DROP FUNCTION grace_voided_event_immutable();
DROP POLICY delete_voided_only ON postings;
REVOKE DELETE ON postings FROM grace_financial;
DROP TABLE voided_postings;
DROP TABLE financial_event_revisions;
ALTER TABLE financial_events
    DROP CONSTRAINT void_reason_with_void,
    DROP COLUMN void_reason,
    DROP COLUMN voided_at,
    DROP COLUMN edited_by_user,
    DROP COLUMN revision;
