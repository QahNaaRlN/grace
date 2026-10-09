DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM auth_password_credentials) THEN
  RAISE EXCEPTION 'Preserve password credentials; rollback auth mode without dropping data';
 END IF;
END $$;
DROP TABLE auth_password_tokens;
DROP TABLE auth_password_credentials;
