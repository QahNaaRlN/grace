-- SPEC-0056: additive document provenance, private render and visual review.
ALTER TABLE statement_sources ADD COLUMN source_kind text NOT NULL DEFAULT 'pdf' CHECK(source_kind IN ('pdf','image'));
ALTER TABLE statement_sources ADD COLUMN reading_mode text NOT NULL DEFAULT 'native_text' CHECK(reading_mode IN ('native_text','image_review'));
ALTER TABLE statement_sources ADD COLUMN rendered_pages jsonb;
ALTER TABLE statement_sources ADD COLUMN visual_state jsonb;
ALTER TABLE statement_sources ADD CONSTRAINT visual_original CHECK(rendered_pages IS NULL OR deleted_at IS NULL);
GRANT UPDATE(visual_state,rendered_pages) ON statement_sources TO grace_financial;
ALTER TABLE statement_extraction_previews DROP CONSTRAINT statement_extraction_previews_maximum_micros_check;
ALTER TABLE statement_extraction_previews ADD CONSTRAINT statement_extraction_previews_maximum_micros_check CHECK(maximum_micros BETWEEN 1 AND 10000000);
CREATE FUNCTION grace_visual_source_guard() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.source_kind IS DISTINCT FROM OLD.source_kind OR NEW.reading_mode IS DISTINCT FROM OLD.reading_mode
 OR (NEW.rendered_pages IS DISTINCT FROM OLD.rendered_pages AND NEW.rendered_pages IS NOT NULL)
 OR (OLD.deleted_at IS NOT NULL AND NEW.rendered_pages IS NOT NULL)
 THEN RAISE EXCEPTION 'Visual source is immutable'; END IF;
 IF NEW.visual_state IS DISTINCT FROM OLD.visual_state AND OLD.batch_id IS NOT NULL AND EXISTS(SELECT 1 FROM import_records WHERE batch_id=OLD.batch_id AND owner_id=OLD.owner_id AND published) THEN RAISE EXCEPTION 'Published visual review is immutable'; END IF;
 IF NEW.layout IS NULL THEN NEW.rendered_pages:=NULL; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER visual_source_guard BEFORE UPDATE ON statement_sources FOR EACH ROW EXECUTE FUNCTION grace_visual_source_guard();
ALTER TABLE receipt_drafts ADD COLUMN source_kind text NOT NULL DEFAULT 'image' CHECK(source_kind IN ('image','pdf'));
ALTER TABLE receipt_drafts ADD COLUMN source_pages jsonb NOT NULL DEFAULT '[]';
ALTER TABLE receipt_drafts ADD COLUMN reading_consent jsonb;
GRANT UPDATE(reading_consent) ON receipt_drafts TO grace_financial;
CREATE FUNCTION grace_receipt_document_guard() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF ROW(NEW.source_kind,NEW.source_pages) IS DISTINCT FROM ROW(OLD.source_kind,OLD.source_pages) THEN RAISE EXCEPTION 'Receipt document source is immutable'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER receipt_document_guard BEFORE UPDATE ON receipt_drafts FOR EACH ROW EXECUTE FUNCTION grace_receipt_document_guard();

ALTER TABLE import_batches DROP CONSTRAINT import_batches_format_check;
ALTER TABLE import_batches ADD CONSTRAINT import_batches_format_check CHECK(format IN ('json','csv','bank_statement_pdf','bank_statement_image'));
