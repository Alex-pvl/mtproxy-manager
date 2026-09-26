#!/usr/bin/env bash
# Dumps the Stay database and mirrors the newest $BACKUP_KEEP dumps to the
# 3x-ui server. Reports status to node_exporter's textfile collector.
#
# Config (env, see stay-backup.env.example):
#   BACKUP_REMOTE   rsync destination, e.g. stay-backup@tagwaiter.ru:   (required)
#   BACKUP_SSH_KEY  private key for that user        (default /root/.ssh/stay_backup)
#   BACKUP_DIR      local dump directory             (default /var/backups/stay)
#   BACKUP_KEEP     dumps to keep, local and remote  (default 8 ≈ 4 months)
#   APP_ENV_FILE    app .env with DATABASE_URL       (default /opt/mtproxy-manager/.env)
#   TEXTFILE_DIR    node_exporter textfile dir       (default /var/lib/node_exporter/textfile)
set -euo pipefail

: "${BACKUP_REMOTE:?BACKUP_REMOTE is required}"
BACKUP_SSH_KEY=${BACKUP_SSH_KEY:-/root/.ssh/stay_backup}
BACKUP_DIR=${BACKUP_DIR:-/var/backups/stay}
BACKUP_KEEP=${BACKUP_KEEP:-8}
APP_ENV_FILE=${APP_ENV_FILE:-/opt/mtproxy-manager/.env}
TEXTFILE_DIR=${TEXTFILE_DIR:-/var/lib/node_exporter/textfile}

# Written atomically so node_exporter never reads a half-written file.
write_metrics() { # $1 = 1|0 success, $2 = dump size bytes
  mkdir -p "$TEXTFILE_DIR"
  local tmp="$TEXTFILE_DIR/stay_backup.prom.$$"
  {
    echo "# HELP stay_backup_last_run_success 1 if the last backup run succeeded."
    echo "# TYPE stay_backup_last_run_success gauge"
    echo "stay_backup_last_run_success $1"
    if [ "$1" = 1 ]; then
      echo "# HELP stay_backup_last_success_timestamp_seconds Unix time of the last successful backup."
      echo "# TYPE stay_backup_last_success_timestamp_seconds gauge"
      echo "stay_backup_last_success_timestamp_seconds $(date +%s)"
      echo "# HELP stay_backup_last_size_bytes Size of the last dump."
      echo "# TYPE stay_backup_last_size_bytes gauge"
      echo "stay_backup_last_size_bytes $2"
    elif [ -f "$TEXTFILE_DIR/stay_backup.prom" ]; then
      # keep the last success timestamp/size so the "missing" alert stays accurate
      grep -E '^stay_backup_last_(success_timestamp_seconds|size_bytes) ' "$TEXTFILE_DIR/stay_backup.prom" || true
    fi
  } > "$tmp"
  mv "$tmp" "$TEXTFILE_DIR/stay_backup.prom"
}
trap 'write_metrics 0 0; echo "backup FAILED" >&2' ERR

# Read only DATABASE_URL instead of sourcing the whole app .env.
DATABASE_URL=$(grep -E '^DATABASE_URL=' "$APP_ENV_FILE" | tail -n1 | cut -d= -f2- | sed -E 's/^["'\'']|["'\'']$//g')
[ -n "$DATABASE_URL" ] || { echo "DATABASE_URL not found in $APP_ENV_FILE" >&2; false; }

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
file="$BACKUP_DIR/stay-$(date -u +%Y%m%d-%H%M%S).dump"

pg_dump --format=custom --no-owner --no-privileges --dbname="$DATABASE_URL" --file="$file.part"
pg_restore --list "$file.part" > /dev/null # the dump must be readable before it replaces anything
mv "$file.part" "$file"
size=$(stat -c %s "$file" 2>/dev/null || stat -f %z "$file")
echo "dumped $file ($size bytes)"

# Local retention; rsync --delete then applies the same set remotely.
ls -1t "$BACKUP_DIR"/stay-*.dump | tail -n +"$((BACKUP_KEEP + 1))" | xargs -r rm -f --
rm -f "$BACKUP_DIR"/*.part

rsync -a --delete --include='stay-*.dump' --exclude='*' \
  -e "ssh -i $BACKUP_SSH_KEY -o BatchMode=yes -o StrictHostKeyChecking=accept-new" \
  "$BACKUP_DIR/" "$BACKUP_REMOTE"
echo "synced to $BACKUP_REMOTE"

write_metrics 1 "$size"
