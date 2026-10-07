DROP TABLE auth_email_rate_windows, auth_email_challenges;
ALTER TABLE auth_invitations DROP COLUMN bound_email;
ALTER TABLE users DROP COLUMN avatar_png, DROP COLUMN last_name, DROP COLUMN first_name;
