ALTER TABLE categories ADD COLUMN owner_id UUID REFERENCES users(id);
ALTER TABLE categories DROP CONSTRAINT categories_name_kind_key;
CREATE UNIQUE INDEX global_category_name_unique ON categories(name,kind) WHERE owner_id IS NULL;
ALTER TABLE categories ADD CONSTRAINT personal_category_valid CHECK(owner_id IS NULL OR
    (kind='expense' AND char_length(name) BETWEEN 1 AND 80 AND name=btrim(name) AND name !~ E'[\n\r\t]'));
CREATE UNIQUE INDEX personal_category_name_unique ON categories(owner_id, lower(name), kind)
    WHERE owner_id IS NOT NULL;
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE categories FORCE ROW LEVEL SECURITY;
CREATE POLICY category_read ON categories FOR SELECT TO grace_financial
    USING(owner_id IS NULL OR owner_id=grace_owner_id());
CREATE POLICY category_create ON categories FOR INSERT TO grace_financial
    WITH CHECK(owner_id=grace_owner_id() AND kind='expense');
CREATE POLICY category_rename ON categories FOR UPDATE TO grace_financial
    USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id());
GRANT INSERT(name,kind,owner_id), UPDATE(name) ON categories TO grace_financial;

-- A category can be global or belong to the event owner, never another user.
-- Invoker rights respect category RLS as well as direct superuser fixtures.
CREATE FUNCTION check_event_category_owner() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.category_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM categories WHERE id=NEW.category_id
          AND (owner_id IS NULL OR owner_id=NEW.owner_id)
    ) THEN RAISE EXCEPTION 'category does not belong to event owner' USING ERRCODE='23514'; END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER event_category_owner BEFORE INSERT OR UPDATE OF category_id,owner_id
    ON financial_events FOR EACH ROW EXECUTE FUNCTION check_event_category_owner();
