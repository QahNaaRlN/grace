#!/bin/sh
# Reproduce the temporary-server race without Docker, a bucket or credentials.
set -eu
repo_root="$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
mkdir "$tmp_dir/bin" "$tmp_dir/state"
cat > "$tmp_dir/owner.env" <<'ENV'
GRACE_OFFSITE_ENDPOINT='http://fixture.invalid'
GRACE_OFFSITE_REGION='fixture'
GRACE_OFFSITE_BUCKET='fixture'
GRACE_OFFSITE_ACCESS_KEY_ID='fixture'
GRACE_OFFSITE_SECRET_ACCESS_KEY='fixture'
ENV
printf '%s\n' 'AGE-SECRET-KEY-FIXTURE' > "$tmp_dir/identity"
cat > "$tmp_dir/bin/sleep" <<'MOCK'
#!/bin/sh
exit 0
MOCK
cat > "$tmp_dir/bin/docker" <<'MOCK'
#!/bin/sh
set -eu
case "$1" in
 run)
  case " $* " in *' POSTGRES_DB=grace '*) echo fixture-postgres;; *) echo fixture-tools;; esac
  exit 0;;
 rm) printf '%s\n' "$*" >> "$READINESS_STATE/cleanup"; exit 0;;
 exec) shift;;
 *) exit 1;;
esac
while [ "$1" != fixture-postgres ] && [ "$1" != fixture-tools ]; do shift; done
container="$1"; shift
if [ "$container" = fixture-tools ]; then
 case "$*" in *'cat > /work/identity'*) cat >/dev/null;; 'cat /work/backup.dump') printf 'synthetic-dump';; esac
 exit 0
fi
# Unix-socket pg_isready accepts during init, before the database exists.
if [ "$1" = pg_isready ]; then exit 0; fi
if [ "$1" = psql ]; then
 case " $* " in
  *' SELECT 1 '*)
   case " $* " in *' -h 127.0.0.1 '*) :;; *) exit 1;; esac
   count=0; [ ! -f "$READINESS_STATE/probes" ] || count="$(cat "$READINESS_STATE/probes")"
   count=$((count + 1)); printf '%s\n' "$count" > "$READINESS_STATE/probes"
   if [ "$READINESS_MODE" = timeout ] || [ "$count" -lt 3 ]; then exit 1; fi
   touch "$READINESS_STATE/ready"; exit 0;;
 esac
 [ -f "$READINESS_STATE/ready" ] || { echo 'FATAL: database "grace" does not exist' >&2; exit 1; }
 case " $* " in *' CREATE ROLE '*) touch "$READINESS_STATE/role";; *) echo 'accounts 2';; esac
 exit 0
fi
if [ "$1" = pg_restore ]; then
 [ -f "$READINESS_STATE/ready" ] && [ -f "$READINESS_STATE/role" ] || exit 1
 cat >/dev/null; touch "$READINESS_STATE/restored"; exit 0
fi
exit 1
MOCK
chmod +x "$tmp_dir/bin/docker" "$tmp_dir/bin/sleep"
export READINESS_STATE="$tmp_dir/state" READINESS_MODE=ready
PATH="$tmp_dir/bin:$PATH" sh "$repo_root/scripts/restore-offsite.sh" "$tmp_dir/owner.env" "$tmp_dir/identity" daily-fixture.dump.age > "$tmp_dir/success.log" 2>&1 || { cat "$tmp_dir/success.log"; exit 1; }
test -f "$READINESS_STATE/restored"
grep -q 'off-site restore drill passed' "$tmp_dir/success.log"
test "$(cat "$READINESS_STATE/probes")" = 3
rm "$READINESS_STATE/ready" "$READINESS_STATE/role" "$READINESS_STATE/restored" "$READINESS_STATE/probes"
export READINESS_MODE=timeout
if PATH="$tmp_dir/bin:$PATH" sh "$repo_root/scripts/restore-offsite.sh" "$tmp_dir/owner.env" "$tmp_dir/identity" daily-fixture.dump.age > "$tmp_dir/timeout.log" 2>&1; then
 echo 'Restore incorrectly succeeded without a ready database' >&2; exit 1
fi
test "$(cat "$READINESS_STATE/probes")" = 60
test ! -f "$READINESS_STATE/role"
test ! -f "$READINESS_STATE/restored"
grep -q 'PostgreSQL did not start' "$tmp_dir/timeout.log"
grep -q fixture-tools "$READINESS_STATE/cleanup"
grep -q fixture-postgres "$READINESS_STATE/cleanup"
echo 'restore readiness regression checks passed'
