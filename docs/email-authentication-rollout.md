# Email authentication rollout — SPEC-0022

Mail delivery uses a TLS-only SMTP adapter with Yandex Cloud Postbox; the domain
is managed in Timeweb. Delivery, DKIM and the original owner's email migration
were verified on 2026-10-07. Email mode is active and Telegram login is temporarily
disabled. Google/Apple research remains SPEC-0019 draft. See [evidence](specs/0022-evidence.md).

## Provider and domain prerequisites

1. Owner signs into/registers Yandex Cloud, accepts its agreement personally,
   and activates the account/billing prerequisites shown by that provider.
2. Use the same folder for the sender and its service account. Grant only
   postbox.sender to the service account. The configured account is grace-mailer.
3. Create the sender identity for graceapp.ru / no-reply@graceapp.ru. Copy the
   generated DKIM records exactly into Timeweb DNS. Preserve existing A/AAAA/NS
   and existing mail records. Configure the provider's current SPF/DMARC guidance;
   do not invent DKIM values or add a second conflicting SPF record.
4. Owner creates an API key with only yc.postbox.send scope. Its identifier is
   the SMTP username and its secret is the SMTP password. Do not paste them into chat or repository.
   Install the completed email.env from the example on the VPS, owned by the
   deployment operator/root, mode 600. The example contains no live secrets.
5. Default settings are postbox.cloud.yandex.net:587 with STARTTLS. Port 465
   may be used with GRACE_SMTP_SECURITY=tls. Plain SMTP is never supported.
6. Verify delivery to the owner's confirmed address before final handoff.

Postbox currently offers 2000 outgoing emails/month without charge; the app
budget defaults to 1500/month and persists in PostgreSQL. Provider usage outside
this app is not covered by that application limit.

## Existing owner, no active session

Keep the existing users UUID/financial owner_id. After deploying the code,
operator runs grace-auth-admin prepare-owner-email EMAIL against the existing
production database. Use the protected operator environment to connect; do not
print DATABASE_URL/password. CLI prints only prepared/expiry, never the token.
The prepared invitation is bound to that email and expires in 24 hours.

The owner then enters the chosen email in the normal website login and
requests a code. The server attaches the prepared migration grant internally;
no guess based on Telegram name, email spelling or financial data is used.
The owner must enter the received code personally. This verifies the mailbox
and binds the email identity to the same existing profile, role and finances.
Telegram identities remain stored; email mode disables their login/callback.
Existing valid sessions remain usable while they are valid.

Email changes in the profile require a session authenticated within 15 minutes
and a code sent to the new email. Other sessions are revoked after a change.

## Operational boundaries

SMTP setup, a protected email.env and owner migration are release prerequisites.
Do not replace working login with an unconfigured sender. Do not expose OTP or
mail credentials in logs, public endpoints, URLs or GitHub artifacts.
Application rollback retains the additive database columns/tables. Removing the
new schema is a separate backup-controlled operation.
The deployment script snapshots the running API's authentication mode before
replacement and restores it on failure. Deploy compatible code with Telegram
still active first; install email.env only after verified delivery and a prepared
owner migration. Never roll a pre-email image back with GRACE_AUTH_MODE=email.

## References verified 2026-10-07

- https://yandex.cloud/ru/docs/postbox/pricing
- https://yandex.cloud/ru/docs/postbox/operations/send-email
- https://yandex.cloud/ru/docs/postbox/concepts/dns-records
- https://timeweb.com/ru/docs/pochta/osnovnye-voprosy-po-rabote-s-pochtoj/osnovnye-nastrojki-pochtovyh-klientov/
- https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
- https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html
