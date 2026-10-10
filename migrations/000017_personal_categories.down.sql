DO $$ BEGIN
    IF EXISTS(SELECT 1 FROM categories WHERE owner_id IS NOT NULL) THEN
        RAISE EXCEPTION 'Personal categories exist; preserve ownership and use application rollback';
    END IF;
END $$;
DROP TRIGGER event_category_owner ON financial_events;
DROP FUNCTION check_event_category_owner();
REVOKE INSERT(name,kind,owner_id), UPDATE(name) ON categories FROM grace_financial;
DROP POLICY category_read ON categories;
DROP POLICY category_create ON categories;
DROP POLICY category_rename ON categories;
ALTER TABLE categories DISABLE ROW LEVEL SECURITY;
DROP INDEX personal_category_name_unique;
DROP INDEX global_category_name_unique;
ALTER TABLE categories DROP CONSTRAINT personal_category_valid;
ALTER TABLE categories ADD UNIQUE(name,kind);
ALTER TABLE categories DROP COLUMN owner_id;
