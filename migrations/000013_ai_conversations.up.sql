CREATE TABLE ai_consents (
 owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 scope TEXT NOT NULL CHECK(scope IN ('analysis','receipt')),
 policy_hash CHAR(64) NOT NULL,
 granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(owner_id,scope)
);
CREATE TABLE ai_conversations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(id,owner_id)
);
CREATE TABLE ai_messages (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 conversation_id UUID NOT NULL,
 request_key TEXT NOT NULL CHECK(length(request_key) BETWEEN 1 AND 160),
 request_hash CHAR(64) NOT NULL,
 question TEXT NOT NULL CHECK(octet_length(question)<=2000),
 month TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','completed','failed')),
 facts JSONB NOT NULL,
 response JSONB,
 error_code TEXT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(id,owner_id), UNIQUE(owner_id,conversation_id,request_key),
 FOREIGN KEY(conversation_id,owner_id) REFERENCES ai_conversations(id,owner_id) ON DELETE CASCADE
);
DO $$ DECLARE t TEXT; BEGIN
 FOREACH t IN ARRAY ARRAY['ai_consents','ai_conversations','ai_messages'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id())',t);
  EXECUTE format('GRANT SELECT,INSERT,DELETE ON %I TO grace_financial',t);
 END LOOP;
 GRANT UPDATE(policy_hash,granted_at) ON ai_consents TO grace_financial;
 GRANT UPDATE(status,response,error_code) ON ai_messages TO grace_financial;
END $$;
CREATE FUNCTION grace_ai_message_complete_immutable() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF OLD.status<>'pending' THEN RAISE EXCEPTION 'Completed AI response is immutable'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER ai_message_immutable BEFORE UPDATE ON ai_messages
FOR EACH ROW EXECUTE FUNCTION grace_ai_message_complete_immutable();
