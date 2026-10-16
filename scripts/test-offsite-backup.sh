#!/bin/sh
# SPEC-0050: exercises the off-site backup scripts against a local S3 server
# (`rclone serve s3`) with a mock Docker for pg_dump, then runs the owner's
# restore drill on a real PostgreSQL dump. Bucket policies are not emulated
# here; the write-only key is checked in production (AC3).
# shellcheck disable=SC2016 # single-quoted commands expand inside the container.
set -eu

repo_root="$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)"
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
run_id="grace-offsite-test-$$"
network="$run_id"
image="alpine:3.20"
postgres_image="postgres:16-alpine"
tmp_dir="$(mktemp -d)"
cleanup() {
  docker rm -f "$run_id-s3" "$run_id-vps" "$run_id-pg" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

copy_in() { # copy_in CONTAINER HOST_DIR CONTAINER_DIR
  docker exec "$1" mkdir -p "$3"
  tar -C "$2" -cf - . | docker exec -i "$1" tar -C "$3" -xf -
}
vps() { docker exec "$run_id-vps" sh -c "$1"; }
fail() { echo "FAIL: $1" >&2; exit 1; }

docker network create "$network" >/dev/null
docker run -d --name "$run_id-s3" --network "$network" "$image" \
  sh -c 'for a in 1 2 3; do timeout 120 apk add --no-cache --quiet rclone && break; done && mkdir -p /data/backups &&
    exec rclone serve s3 --addr :8080 --auth-key test-access,test-secret /data' >/dev/null
docker run -d --name "$run_id-vps" --network "$network" "$image" sleep 3600 >/dev/null
vps 'for a in 1 2 3; do timeout 120 apk add --no-cache --quiet age rclone curl findutils && exit 0; done; exit 1'

# A release layout like /opt/grace, with a mock Docker that prints a synthetic dump.
mkdir -p "$tmp_dir/release" "$tmp_dir/bin" "$tmp_dir/secrets"
cp "$repo_root/deployments/production/backup.sh" "$repo_root/deployments/production/offsite-upload.sh" \
  "$repo_root/deployments/production/backup-failure-mail.sh" "$tmp_dir/release/"
printf 'RELEASE_TAG=sha-test\nGRACE_SECRETS_DIR=/opt/grace/shared/secrets\n' > "$tmp_dir/release/release.env"
printf 'test-postgres-password' > "$tmp_dir/secrets/postgres_password"
cat > "$tmp_dir/bin/docker" <<'EOF'
#!/bin/sh
case "$*" in
  *'pg_dump'*) cat /opt/dump-source ;;
  *'pg_restore --list'*) cat >/dev/null ;;
esac
EOF
copy_in "$run_id-vps" "$tmp_dir/release" /opt/grace/releases/test
copy_in "$run_id-vps" "$tmp_dir/bin" /usr/local/mock
copy_in "$run_id-vps" "$tmp_dir/secrets" /opt/grace/shared/secrets
vps 'chmod 0755 /opt/grace/releases/test/*.sh /usr/local/mock/docker &&
  ln -s /opt/grace/releases/test /opt/grace/current &&
  printf "synthetic dump one" > /opt/dump-source'

# The owner's key pair stays outside /opt/grace; only the public key reaches the VPS.
vps 'age-keygen -o /root/owner-identity 2>/dev/null'
recipient="$(vps 'age-keygen -y /root/owner-identity')"
offsite_env() { # offsite_env SECRET [RECIPIENT]
  vps "umask 077; cat > /opt/grace/shared/secrets/offsite.env <<EOF
GRACE_OFFSITE_ENDPOINT='http://$run_id-s3:8080'
GRACE_OFFSITE_REGION='ru-central1'
GRACE_OFFSITE_BUCKET='backups'
GRACE_OFFSITE_PREFIX='grace'
GRACE_OFFSITE_ACCESS_KEY_ID='test-access'
GRACE_OFFSITE_SECRET_ACCESS_KEY='$1'
GRACE_OFFSITE_AGE_RECIPIENT='${2:-$recipient}'
GRACE_OFFSITE_ALERT_TO='owner@example.test'
EOF"
}
backup() { sleep 1; vps "PATH=/usr/local/mock:\$PATH /opt/grace/current/backup.sh $1"; }

echo "AC5: without offsite.env the backup is local only, with one warning"
backup daily > "$tmp_dir/plain.out" 2> "$tmp_dir/plain.err" || fail "backup without offsite.env failed"
grep -q '^/opt/grace/backups/daily-.*-sha-test.dump$' "$tmp_dir/plain.out" || fail "dump path not printed"
[ "$(wc -l < "$tmp_dir/plain.out")" -eq 1 ] || fail "unexpected output without offsite.env"
if [ "$(wc -l < "$tmp_dir/plain.err")" -ne 1 ] || ! grep -q 'off-site backup is not configured' "$tmp_dir/plain.err"; then
  cat "$tmp_dir/plain.err" >&2
  fail "expected exactly one warning"
fi

echo "AC1/AC2: a daily backup is encrypted, uploaded and decryptable only with the owner's key"
offsite_env test-secret
vps 'grep -rlE "AGE-SECRET-KEY-1[0-9A-Z]{20}" /opt/grace' && fail "a private key is under /opt/grace"
backup daily > "$tmp_dir/upload.out" || fail "backup with off-site copy failed"
dump="$(head -n 1 "$tmp_dir/upload.out")"
name="$(basename "$dump")"
grep -q "off-site copy uploaded: $name.age sha256=" "$tmp_dir/upload.out" || fail "upload not reported"
vps "test -f '$dump' && test -f '$dump.uploaded' && ! ls /opt/grace/backups/*.age 2>/dev/null" \
  || fail "local dump, marker or cleanup is wrong"
vps "export RCLONE_CONFIG_O_TYPE=s3 RCLONE_CONFIG_O_PROVIDER=Other RCLONE_CONFIG_O_ENDPOINT=http://$run_id-s3:8080 \
  RCLONE_CONFIG_O_ACCESS_KEY_ID=test-access RCLONE_CONFIG_O_SECRET_ACCESS_KEY=test-secret
  mkdir -p /check && cd /check &&
  rclone copyto 'o:backups/grace/$name.age' '$name.age' &&
  rclone copyto 'o:backups/grace/$name.age.sha256' '$name.age.sha256' &&
  sha256sum -c '$name.age.sha256' >/dev/null &&
  test \"\$(age -d -i /root/owner-identity '$name.age')\" = 'synthetic dump one' &&
  ! grep -q 'synthetic dump' '$name.age'" || fail "off-site copy is not the encrypted dump"

echo "AC4: a failed upload keeps the dump, fails the run and leaks no secret"
offsite_env wrong-secret-value
vps 'printf "synthetic dump two" > /opt/dump-source'
if backup manual > "$tmp_dir/fail.out" 2> "$tmp_dir/fail.err"; then fail "upload with a wrong key succeeded"; fi
failed="$(head -n 1 "$tmp_dir/fail.out")"
vps "test -f '$failed' && test ! -f '$failed.uploaded' && grep -q '^stage=upload$' /opt/grace/backups/.offsite-failure" \
  || fail "failed upload state is wrong"
if grep -q 'wrong-secret-value\|test-postgres-password' "$tmp_dir/fail.out" "$tmp_dir/fail.err"; then
  fail "a secret reached the output"
fi

echo "AC4: the failure mail names the stage and keeps credentials out of curl arguments"
vps 'umask 077; printf "%s\n" "GRACE_SMTP_HOST=smtp.example.test" "GRACE_SMTP_PORT=587" \
  "GRACE_SMTP_SECURITY=starttls" "GRACE_SMTP_FROM=\"Grace <no-reply@example.test>\"" \
  "GRACE_SMTP_USERNAME=mail-user" "GRACE_SMTP_PASSWORD=mail-password-value" > /opt/grace/shared/secrets/email.env
  mkdir -p /usr/local/mail && cat > /usr/local/mail/curl <<"EOF"
#!/bin/sh
echo "$*" > /tmp/curl-args
while [ "$#" -gt 0 ]; do [ "$1" = "--upload-file" ] && cp "$2" /tmp/curl-message; shift; done
EOF
  chmod 0755 /usr/local/mail/curl'
vps 'PATH=/usr/local/mail:$PATH /opt/grace/current/backup-failure-mail.sh' >/dev/null || fail "alert script failed"
vps 'grep -q "Subject: Grace: backup failed (upload)" /tmp/curl-message &&
  grep -q "^To: owner@example.test" /tmp/curl-message &&
  grep -q -- "--mail-rcpt owner@example.test" /tmp/curl-args &&
  ! grep -q "mail-password-value\|wrong-secret-value" /tmp/curl-args /tmp/curl-message' \
  || fail "alert content is wrong"

echo "AC4: a private key in the recipient setting is refused and never printed"
private_key="$(vps 'grep "^AGE-SECRET-KEY-" /root/owner-identity')"
offsite_env test-secret "$private_key"
if backup manual > "$tmp_dir/private.out" 2> "$tmp_dir/private.err"; then fail "a private key was accepted as recipient"; fi
grep -q 'holds a private age key' "$tmp_dir/private.err" || { cat "$tmp_dir/private.err" >&2; fail "private key not reported"; }
vps 'grep -q "^stage=config$" /opt/grace/backups/.offsite-failure' || fail "config failure not recorded"
if grep -q "$private_key" "$tmp_dir/private.out" "$tmp_dir/private.err"; then fail "the private key reached the output"; fi
vps 'PATH=/usr/local/mail:$PATH /opt/grace/current/backup-failure-mail.sh' >/dev/null || fail "alert script failed"
vps "! grep -q '$private_key' /tmp/curl-message /tmp/curl-args" || fail "the private key reached the alert"

echo "AC4: age errors never reach the output"
offsite_env test-secret 'age1notavalidkey'
if backup manual > "$tmp_dir/badkey.out" 2> "$tmp_dir/badkey.err"; then fail "an invalid recipient was accepted"; fi
if grep -q 'age1notavalidkey\|age: error' "$tmp_dir/badkey.out" "$tmp_dir/badkey.err"; then fail "age output reached the log"; fi

echo "upload-pending retries the failed copy once the key is fixed"
offsite_env test-secret
vps '/opt/grace/current/backup.sh upload-pending' > "$tmp_dir/pending.out" || fail "upload-pending failed"
grep -q "off-site copy uploaded: $(basename "$failed").age" "$tmp_dir/pending.out" || fail "pending copy not uploaded"
[ "$(grep -c 'off-site copy uploaded' "$tmp_dir/pending.out")" -eq 4 ] || fail "expected the four pending dumps"
vps 'test ! -f /opt/grace/backups/.offsite-failure' || fail "failure state not cleared"

echo "AC6 (synthetic): the owner's restore drill restores a real dump"
docker run -d --name "$run_id-pg" --network "$network" -e POSTGRES_USER=grace \
  -e POSTGRES_PASSWORD=test -e POSTGRES_DB=grace "$postgres_image" >/dev/null
until docker exec "$run_id-pg" psql -U grace -d grace -c 'SELECT 1' >/dev/null 2>&1; do sleep 1; done
sleep 2
docker exec "$run_id-pg" psql -U grace -d grace -v ON_ERROR_STOP=1 -q -c "
  CREATE ROLE grace_financial NOLOGIN;
  CREATE TABLE accounts(id int PRIMARY KEY, owner_id int);
  CREATE TABLE financial_events(id int PRIMARY KEY);
  CREATE TABLE postings(id int PRIMARY KEY, event_id int REFERENCES financial_events);
  CREATE TABLE loan_documents(id int PRIMARY KEY, content bytea);
  ALTER TABLE accounts ENABLE ROW LEVEL SECURITY;
  CREATE POLICY owner_access ON accounts TO grace_financial USING (owner_id = 1);
  GRANT SELECT ON accounts TO grace_financial;
  INSERT INTO accounts VALUES (1, 1), (2, 1);
  INSERT INTO financial_events VALUES (1), (2), (3);
  INSERT INTO postings VALUES (1, 1), (2, 1), (3, 2), (4, 3);
  INSERT INTO loan_documents VALUES (1, '\\x255044462d');"
docker exec "$run_id-pg" pg_dump -U grace -d grace -Fc | docker exec -i "$run_id-vps" sh -c 'cat > /opt/dump-source'
backup daily > /dev/null || fail "backup of the real dump failed"
vps 'cat /root/owner-identity' > "$tmp_dir/identity"
cat > "$tmp_dir/owner.env" <<EOF
GRACE_OFFSITE_ENDPOINT='http://$run_id-s3:8080'
GRACE_OFFSITE_REGION='ru-central1'
GRACE_OFFSITE_BUCKET='backups'
GRACE_OFFSITE_PREFIX='grace'
GRACE_OFFSITE_ACCESS_KEY_ID='test-access'
GRACE_OFFSITE_SECRET_ACCESS_KEY='test-secret'
EOF
GRACE_RESTORE_NETWORK="$network" sh "$repo_root/scripts/restore-offsite.sh" \
  "$tmp_dir/owner.env" "$tmp_dir/identity" > "$tmp_dir/restore.out" || { cat "$tmp_dir/restore.out" >&2; fail "restore drill failed"; }
for expected in 'financial_events 3' 'postings 4' 'accounts 2' 'loan_documents 1' 'off-site restore drill passed'; do
  grep -q "$expected" "$tmp_dir/restore.out" || { cat "$tmp_dir/restore.out" >&2; fail "restore output lacks: $expected"; }
done

echo "AC6: files saved on Windows (CRLF, copied screen line) still restore"
{ printf 'Public key: %s\r\n' "$recipient"; sed 's/$/\r/' "$tmp_dir/identity"; } > "$tmp_dir/identity-windows"
sed 's/$/\r/' "$tmp_dir/owner.env" > "$tmp_dir/owner-windows.env"
GRACE_RESTORE_NETWORK="$network" sh "$repo_root/scripts/restore-offsite.sh" \
  "$tmp_dir/owner-windows.env" "$tmp_dir/identity-windows" > "$tmp_dir/restore-windows.out" \
  || { cat "$tmp_dir/restore-windows.out" >&2; fail "restore from Windows files failed"; }
grep -q 'off-site restore drill passed' "$tmp_dir/restore-windows.out" || fail "Windows restore not reported"

echo "off-site backup checks passed"
