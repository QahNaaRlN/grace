DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM receipt_draft_confirmations) THEN RAISE EXCEPTION 'Cannot discard confirmed AI receipt provenance'; END IF;
 IF EXISTS(SELECT 1 FROM receipt_drafts WHERE photo_status<>'deleted') THEN RAISE EXCEPTION 'Delete temporary receipt photos before downgrade'; END IF;
END $$;
DROP TABLE receipt_draft_confirmations,receipt_draft_previews,receipt_drafts;
DROP FUNCTION grace_receipt_draft_immutable();
