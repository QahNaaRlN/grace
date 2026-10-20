DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM statement_sources WHERE mapping_snapshot IS NOT NULL) THEN
  RAISE EXCEPTION 'Cannot discard saved statement mapping';
 END IF;
END $$;
DROP TRIGGER statement_mapping_review ON statement_sources;
DROP FUNCTION grace_statement_mapping_review();
DROP TRIGGER statement_mapping_original ON statement_sources;
DROP FUNCTION grace_statement_mapping_original();
ALTER TABLE statement_sources DROP CONSTRAINT statement_mapping_original;
REVOKE UPDATE(mapping_snapshot) ON statement_sources FROM grace_financial;
ALTER TABLE statement_sources DROP COLUMN mapping_snapshot;
