ALTER TABLE statement_sources ADD COLUMN mapping_snapshot jsonb;
GRANT UPDATE(mapping_snapshot) ON statement_sources TO grace_financial;
ALTER TABLE statement_sources ADD CONSTRAINT statement_mapping_original CHECK(mapping_snapshot IS NULL OR layout IS NOT NULL);
CREATE FUNCTION grace_statement_mapping_original() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.layout IS NULL THEN
  NEW.mapping_snapshot:=NULL;
 ELSIF TG_OP='UPDATE' AND NEW.mapping_snapshot IS DISTINCT FROM OLD.mapping_snapshot
 AND OLD.batch_id IS NOT NULL AND EXISTS(SELECT 1 FROM import_records WHERE batch_id=OLD.batch_id AND owner_id=OLD.owner_id AND published) THEN
  RAISE EXCEPTION 'Published statement mapping is immutable';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER statement_mapping_original BEFORE INSERT OR UPDATE ON statement_sources FOR EACH ROW EXECUTE FUNCTION grace_statement_mapping_original();
-- A previous application version writes review without knowing this column.
-- Invalidate its stale snapshot even when the new review is byte-identical.
CREATE FUNCTION grace_statement_mapping_review() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.mapping_snapshot IS NOT DISTINCT FROM OLD.mapping_snapshot THEN
  IF OLD.batch_id IS NOT NULL AND EXISTS(SELECT 1 FROM import_records WHERE batch_id=OLD.batch_id AND owner_id=OLD.owner_id AND published) THEN
   RAISE EXCEPTION 'Published statement mapping is immutable';
  END IF;
  NEW.mapping_snapshot:=NULL;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER statement_mapping_review BEFORE UPDATE OF review ON statement_sources FOR EACH ROW EXECUTE FUNCTION grace_statement_mapping_review();
