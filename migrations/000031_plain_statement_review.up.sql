-- SPEC-0058: explicit plain drafts, no change to historical evidence modes.
ALTER TABLE statement_sources ADD COLUMN plain_state jsonb;
ALTER TABLE statement_sources DROP CONSTRAINT statement_sources_reading_mode_check;
ALTER TABLE statement_sources ADD CONSTRAINT statement_sources_reading_mode_check CHECK(reading_mode IN ('native_text','image_review','plain_review'));
GRANT UPDATE(plain_state,reading_mode) ON statement_sources TO grace_financial;
CREATE OR REPLACE FUNCTION grace_visual_source_guard() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.source_kind IS DISTINCT FROM OLD.source_kind
 OR (NEW.rendered_pages IS DISTINCT FROM OLD.rendered_pages AND NEW.rendered_pages IS NOT NULL)
 OR (OLD.deleted_at IS NOT NULL AND NEW.rendered_pages IS NOT NULL)
 THEN RAISE EXCEPTION 'Visual source is immutable'; END IF;
 IF NEW.reading_mode IS DISTINCT FROM OLD.reading_mode THEN
  IF OLD.deleted_at IS NOT NULL OR NEW.revision<>OLD.revision+1 OR NEW.review IS NOT NULL OR NEW.plain_state IS NOT NULL OR NEW.visual_state IS NOT NULL OR NEW.roles_confirmed OR NEW.reviewed_pages<>'[]'::jsonb
  OR EXISTS(SELECT 1 FROM import_records WHERE batch_id=OLD.batch_id AND owner_id=OLD.owner_id AND published)
  THEN RAISE EXCEPTION 'Invalid reading mode transition'; END IF;
 END IF;
 IF (NEW.visual_state IS DISTINCT FROM OLD.visual_state OR NEW.plain_state IS DISTINCT FROM OLD.plain_state) AND OLD.batch_id IS NOT NULL AND NEW.layout IS NOT NULL AND EXISTS(SELECT 1 FROM import_records WHERE batch_id=OLD.batch_id AND owner_id=OLD.owner_id AND published) THEN RAISE EXCEPTION 'Published document review is immutable'; END IF;
 IF NEW.layout IS NULL THEN NEW.rendered_pages:=NULL; NEW.plain_state:=NULL; END IF;
 RETURN NEW;
END $$;
