-- SPEC-0055: bounded local sources. No financial events are created here.
ALTER TABLE documents DROP CONSTRAINT documents_kind_check;
ALTER TABLE documents ADD CONSTRAINT documents_kind_check CHECK(kind IN ('credit_card_terms','bank_statement'));
ALTER TABLE documents ALTER COLUMN content DROP NOT NULL;
ALTER TABLE documents ADD COLUMN original_deleted_at timestamptz;
ALTER TABLE documents ADD CONSTRAINT documents_original_state CHECK (
 (kind='credit_card_terms' AND content IS NOT NULL AND original_deleted_at IS NULL) OR
 (kind='bank_statement' AND ((content IS NOT NULL AND octet_length(content)<=10485760 AND original_deleted_at IS NULL) OR (content IS NULL AND original_deleted_at IS NOT NULL)))
);
CREATE UNIQUE INDEX documents_bank_statement_sha_idx ON documents(owner_id,sha256) WHERE kind='bank_statement';
CREATE FUNCTION grace_document_original_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF ROW(NEW.id,NEW.owner_id,NEW.kind,NEW.filename,NEW.sha256,NEW.created_at) IS DISTINCT FROM ROW(OLD.id,OLD.owner_id,OLD.kind,OLD.filename,OLD.sha256,OLD.created_at)
 OR OLD.kind<>'bank_statement' OR OLD.content IS NULL OR NEW.content IS NOT NULL OR NEW.original_deleted_at IS NULL
 THEN RAISE EXCEPTION 'Document originals are immutable; only statement tombstones are allowed'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER document_original_guard BEFORE UPDATE ON documents FOR EACH ROW EXECUTE FUNCTION grace_document_original_guard();
GRANT UPDATE(content,original_deleted_at) ON documents TO grace_financial;
ALTER TABLE import_batches DROP CONSTRAINT import_batches_format_check;
ALTER TABLE import_batches ADD CONSTRAINT import_batches_format_check CHECK(format IN ('json','csv','bank_statement_pdf'));
CREATE TABLE statement_sources (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 document_id uuid NOT NULL,
 account_id uuid NOT NULL,
 batch_id uuid,
 revision bigint NOT NULL DEFAULT 1 CHECK(revision>0),
 layout jsonb,
 page_count integer NOT NULL CHECK(page_count BETWEEN 1 AND 50),
 original_size_bytes integer NOT NULL CHECK(original_size_bytes BETWEEN 5 AND 10485760),
 review jsonb,
 status text NOT NULL DEFAULT 'ready' CHECK(status IN ('ready','reviewing','deleted','expired')),
 roles_confirmed boolean NOT NULL DEFAULT false,
 reviewed_pages jsonb NOT NULL DEFAULT '[]',
 retention text NOT NULL DEFAULT 'delete_after_confirmation' CHECK(retention IN ('delete_after_confirmation','keep')),
 expires_at timestamptz NOT NULL DEFAULT now()+interval '7 days',
 deleted_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(id,owner_id), UNIQUE(document_id,owner_id), UNIQUE(batch_id,owner_id),
 FOREIGN KEY(document_id,owner_id) REFERENCES documents(id,owner_id),
 FOREIGN KEY(account_id,owner_id) REFERENCES accounts(id,owner_id),
 FOREIGN KEY(batch_id,owner_id) REFERENCES import_batches(id,owner_id),
 CHECK((deleted_at IS NULL AND layout IS NOT NULL AND status NOT IN ('deleted','expired')) OR (deleted_at IS NOT NULL AND layout IS NULL AND status IN ('deleted','expired')))
);
CREATE TABLE statement_audit (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 statement_id uuid NOT NULL,
 revision bigint NOT NULL,
 action text NOT NULL,
 reason text NOT NULL CHECK(length(reason) BETWEEN 1 AND 2000),
 metadata jsonb NOT NULL DEFAULT '{}',
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(statement_id,owner_id) REFERENCES statement_sources(id,owner_id)
);
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['statement_sources','statement_audit'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id())',t);
  EXECUTE format('GRANT SELECT,INSERT ON %I TO grace_financial',t);
 END LOOP;
END $$;
GRANT UPDATE(account_id,batch_id,revision,layout,review,status,roles_confirmed,reviewed_pages,retention,deleted_at) ON statement_sources TO grace_financial;
CREATE FUNCTION grace_statement_source_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='INSERT' THEN
  IF NOT EXISTS(SELECT 1 FROM documents WHERE id=NEW.document_id AND owner_id=NEW.owner_id AND kind='bank_statement' AND content IS NOT NULL) OR NOT EXISTS(SELECT 1 FROM accounts WHERE id=NEW.account_id AND owner_id=NEW.owner_id AND type='debit' AND currency='RUB') THEN RAISE EXCEPTION 'Statement requires an owned bank document and debit RUB account'; END IF;
  RETURN NEW;
 END IF;
 IF NEW.account_id IS DISTINCT FROM OLD.account_id AND NOT EXISTS(SELECT 1 FROM accounts WHERE id=NEW.account_id AND owner_id=NEW.owner_id AND type='debit' AND currency='RUB') THEN RAISE EXCEPTION 'Statement requires debit RUB account'; END IF;
 IF ROW(NEW.id,NEW.owner_id,NEW.document_id,NEW.created_at,NEW.expires_at,NEW.page_count,NEW.original_size_bytes) IS DISTINCT FROM ROW(OLD.id,OLD.owner_id,OLD.document_id,OLD.created_at,OLD.expires_at,OLD.page_count,OLD.original_size_bytes)
 OR (OLD.deleted_at IS NOT NULL AND (NEW.layout IS NOT NULL OR NEW.deleted_at IS DISTINCT FROM OLD.deleted_at))
 OR (NEW.layout IS NOT NULL AND NEW.layout IS DISTINCT FROM OLD.layout)
 OR (OLD.batch_id IS NOT NULL AND NEW.batch_id IS DISTINCT FROM OLD.batch_id)
 THEN RAISE EXCEPTION 'Statement source is immutable'; END IF;
 IF NEW.deleted_at IS NOT NULL AND EXISTS(SELECT 1 FROM documents WHERE id=NEW.document_id AND owner_id=NEW.owner_id AND content IS NOT NULL) THEN RAISE EXCEPTION 'Delete the original before clearing source state'; END IF;
 IF OLD.batch_id IS NOT NULL AND EXISTS(SELECT 1 FROM import_records WHERE batch_id=OLD.batch_id AND published)
 AND (NEW.account_id IS DISTINCT FROM OLD.account_id OR NEW.review IS DISTINCT FROM OLD.review) THEN RAISE EXCEPTION 'Published statement binding/evidence is immutable'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER statement_source_guard BEFORE INSERT OR UPDATE ON statement_sources FOR EACH ROW EXECUTE FUNCTION grace_statement_source_guard();
CREATE INDEX statement_sources_expiry_idx ON statement_sources(expires_at) WHERE deleted_at IS NULL AND retention='delete_after_confirmation';
-- Restrict the privileged source patch to binding only, before any publication.
CREATE FUNCTION grace_statement_rebind(source_id uuid,new_account uuid) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public AS $$
DECLARE b uuid; own uuid:=grace_owner_id(); rec record; patched jsonb;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('ledger:'||own::text,0));
 SELECT batch_id INTO b FROM statement_sources WHERE id=source_id AND owner_id=own FOR UPDATE;
 IF NOT FOUND OR b IS NULL THEN RAISE EXCEPTION 'Statement not found'; END IF;
 IF NOT EXISTS(SELECT 1 FROM accounts WHERE id=new_account AND owner_id=own AND type='debit' AND currency='RUB')
 OR EXISTS(SELECT 1 FROM import_records WHERE batch_id=b AND owner_id=own AND published) THEN RAISE EXCEPTION 'Binding is not editable'; END IF;
 FOR rec IN SELECT id,source FROM import_records WHERE batch_id=b AND owner_id=own FOR UPDATE LOOP
  patched:=jsonb_set(jsonb_set(rec.source,'{pdf,accountId}',to_jsonb(new_account::text)),'{normalized,accountId}',to_jsonb(new_account::text));
  UPDATE import_records SET source=patched WHERE id=rec.id AND owner_id=own;
 END LOOP;
 UPDATE import_batches SET report=jsonb_set(report,'{records}',(SELECT jsonb_agg(source ORDER BY (source->'pdf'->>'ordinal')::int) FROM import_records WHERE batch_id=b AND owner_id=own)) WHERE id=b AND owner_id=own;
END $$;
REVOKE ALL ON FUNCTION grace_statement_rebind(uuid,uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION grace_statement_rebind(uuid,uuid) TO grace_financial;
-- Before publication, re-group within the same batch; preserve the replaced
-- minimal source/decisions as an audit entry and invalidate every old preview.
CREATE FUNCTION grace_statement_prepare_regroup(source_id uuid,new_report jsonb,reason text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public AS $$
DECLARE b uuid;own uuid:=grace_owner_id();rev bigint;archived jsonb;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('ledger:'||own::text,0));
 SELECT batch_id,revision INTO b,rev FROM statement_sources WHERE id=source_id AND owner_id=own AND deleted_at IS NULL FOR UPDATE;
 IF NOT FOUND OR b IS NULL OR length(reason) NOT BETWEEN 1 AND 2000 OR jsonb_array_length(new_report->'records') NOT BETWEEN 1 AND 5000
 OR EXISTS(SELECT 1 FROM import_records WHERE batch_id=b AND owner_id=own AND published) THEN RAISE EXCEPTION 'Statement regroup is not allowed'; END IF;
 SELECT COALESCE(jsonb_agg(jsonb_build_object('source',r.source,'decision',r.decision,'history',(SELECT COALESCE(jsonb_agg(d.decision ORDER BY d.revision),'[]') FROM import_decisions d WHERE d.record_id=r.id AND d.owner_id=own))),'[]') INTO archived FROM import_records r WHERE r.batch_id=b AND r.owner_id=own;
 INSERT INTO statement_audit(owner_id,statement_id,revision,action,reason,metadata) VALUES(own,source_id,rev,'regroup_archive',reason,jsonb_build_object('previousRecords',archived));
 DELETE FROM import_decisions WHERE owner_id=own AND record_id IN (SELECT id FROM import_records WHERE batch_id=b AND owner_id=own);
 DELETE FROM import_records WHERE batch_id=b AND owner_id=own;
 UPDATE import_batches SET report=new_report,revision=revision+1 WHERE id=b AND owner_id=own;
 RETURN b;
END $$;
REVOKE ALL ON FUNCTION grace_statement_prepare_regroup(uuid,jsonb,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION grace_statement_prepare_regroup(uuid,jsonb,text) TO grace_financial;
-- A direct, owner-scoped SQL deletion must clear the extracted original too;
-- it cannot leave full text behind a document tombstone.
CREATE FUNCTION grace_statement_tombstone_original() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.kind='bank_statement' AND NEW.content IS NULL AND OLD.content IS NOT NULL THEN
  UPDATE statement_sources SET layout=NULL,deleted_at=NEW.original_deleted_at,status='deleted',revision=revision+1 WHERE document_id=NEW.id AND owner_id=NEW.owner_id AND deleted_at IS NULL;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER statement_tombstone_original AFTER UPDATE ON documents FOR EACH ROW EXECUTE FUNCTION grace_statement_tombstone_original();
