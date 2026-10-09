CREATE TABLE receipt_drafts (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 image_id TEXT NOT NULL CHECK(image_id ~ '^[0-9a-f]{48}$'),
 image_hash CHAR(64) NOT NULL,
 revision BIGINT NOT NULL DEFAULT 1,
 status TEXT NOT NULL DEFAULT 'uploaded' CHECK(status IN ('uploaded','extracting','extracted','reviewed','confirmed','cancelled','expired')),
 photo_status TEXT NOT NULL DEFAULT 'stored' CHECK(photo_status IN ('stored','pending_delete','deleted')),
 fields JSONB NOT NULL DEFAULT '{"items":[],"action":""}',
 extractions JSONB NOT NULL DEFAULT '[]',
 extraction_key TEXT NOT NULL DEFAULT '',
 extraction_started_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 expires_at TIMESTAMPTZ NOT NULL DEFAULT now()+interval '24 hours',
 receipt_id UUID,
 event_id UUID,
 UNIQUE(id,owner_id), UNIQUE(owner_id,image_hash),
 FOREIGN KEY(receipt_id,owner_id) REFERENCES receipts(id,owner_id),
 FOREIGN KEY(event_id,owner_id) REFERENCES financial_events(id,owner_id)
);
CREATE TABLE receipt_draft_previews (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 draft_id UUID NOT NULL,
 revision BIGINT NOT NULL,
 hash CHAR(64) NOT NULL,
 payload JSONB NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL DEFAULT now()+interval '10 minutes',
 UNIQUE(id,owner_id),
 FOREIGN KEY(draft_id,owner_id) REFERENCES receipt_drafts(id,owner_id)
);
CREATE TABLE receipt_draft_confirmations (
 owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 request_key TEXT NOT NULL CHECK(length(request_key) BETWEEN 1 AND 160),
 request_hash CHAR(64) NOT NULL,
 draft_id UUID NOT NULL,
 preview_id UUID NOT NULL,
 receipt_id UUID NOT NULL,
 event_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(owner_id,request_key), UNIQUE(draft_id,owner_id),
 FOREIGN KEY(draft_id,owner_id) REFERENCES receipt_drafts(id,owner_id),
 FOREIGN KEY(preview_id,owner_id) REFERENCES receipt_draft_previews(id,owner_id),
 FOREIGN KEY(receipt_id,owner_id) REFERENCES receipts(id,owner_id),
 FOREIGN KEY(event_id,owner_id) REFERENCES financial_events(id,owner_id)
);
DO $$ DECLARE t TEXT; BEGIN
 FOREACH t IN ARRAY ARRAY['receipt_drafts','receipt_draft_previews','receipt_draft_confirmations'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id())',t);
  EXECUTE format('GRANT SELECT,INSERT ON %I TO grace_financial',t);
 END LOOP;
 GRANT UPDATE(revision,status,photo_status,fields,extractions,extraction_key,extraction_started_at,receipt_id,event_id) ON receipt_drafts TO grace_financial;
END $$;
CREATE FUNCTION grace_receipt_draft_immutable() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF OLD.status='confirmed' AND (NEW.revision IS DISTINCT FROM OLD.revision OR NEW.fields IS DISTINCT FROM OLD.fields OR NEW.extractions IS DISTINCT FROM OLD.extractions OR NEW.receipt_id IS DISTINCT FROM OLD.receipt_id OR NEW.event_id IS DISTINCT FROM OLD.event_id OR NEW.status IS DISTINCT FROM OLD.status) THEN RAISE EXCEPTION 'Confirmed receipt source is immutable'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER receipt_draft_immutable BEFORE UPDATE ON receipt_drafts FOR EACH ROW EXECUTE FUNCTION grace_receipt_draft_immutable();
