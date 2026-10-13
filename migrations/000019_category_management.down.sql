-- Icons and hidden flags are presentation data; deleted categories are not restored.
DROP TABLE category_hidden;
DROP POLICY category_delete ON categories;
REVOKE DELETE, INSERT(icon), UPDATE(name, icon) ON categories FROM grace_financial;
GRANT UPDATE(name) ON categories TO grace_financial;
ALTER TABLE categories DROP COLUMN icon;
