#!/usr/bin/env bash
set -euo pipefail
# Disable bash history expansion so "!" in message text does not break.
set +H

# Рассылка всем пользователям: сервис снова доступен.
#
# Получатели по умолчанию берутся из БД (все users с telegram_id).
# Можно передать файл со списком chat_id первым аргументом.
#
# Использование:
#   ./deploy/send_service_restored_broadcast.sh                 # всем из БД
#   ./deploy/send_service_restored_broadcast.sh ids.txt         # из файла
#
# TG_BOT_TOKEN / DATABASE_URL берутся из /opt/mtproxy-manager/.env
# (переопредели через ENV_FILE=/path/.env).

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

ENV_FILE="${ENV_FILE:-/opt/mtproxy-manager/.env}"
if [[ -f "$ENV_FILE" ]]; then
  set -a; source "$ENV_FILE"; set +a
fi

if [[ -z "${TG_BOT_TOKEN:-}" ]]; then
  echo "TG_BOT_TOKEN is not set"
  exit 1
fi

# Источник ID: файл-аргумент или БД.
IDS_FILE="${1:-}"
if [[ -n "$IDS_FILE" ]]; then
  if [[ ! -f "$IDS_FILE" ]]; then
    echo "IDs file not found: $IDS_FILE"
    exit 1
  fi
  ids_cmd() { cat "$IDS_FILE"; }
else
  : "${DATABASE_URL:?DATABASE_URL не задан — укажи ENV_FILE=/path/.env или передай файл с ID}"
  ids_cmd() {
    psql "$DATABASE_URL" -tAc \
      "SELECT telegram_id FROM users WHERE id = 1;"
  }
fi

MESSAGE="$(cat <<'EOF'
✅ Stay снова доступен!

Технические работы завершены — VPN снова работает в штатном режиме.

Откройте приложение, чтобы создать или обновить конфигурацию.
EOF
)"

API_URL="https://api.telegram.org/bot${TG_BOT_TOKEN}/sendMessage"
OPEN_STAY_URL="${OPEN_STAY_URL:-${TELEGRAM_PAY_URL:-${BASE_URL:-https://t.me/staytg_bot/pay}}}"

if [[ ! "$OPEN_STAY_URL" =~ ^https?:// ]]; then
  echo "OPEN_STAY_URL must start with http:// or https://"
  exit 1
fi

sent=0
failed=0

while IFS= read -r line || [[ -n "$line" ]]; do
  chat_id="$(echo "$line" | tr -d '[:space:]')"
  if [[ -z "$chat_id" ]] || [[ "$chat_id" == \#* ]]; then
    continue
  fi

  if curl -sS --fail --location -X POST "$API_URL" \
    --header "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "chat_id=$chat_id" \
    --data-urlencode "text=$MESSAGE" \
    --data-urlencode "parse_mode=HTML" \
    --data "disable_web_page_preview=true" \
    --data-urlencode "reply_markup={\"inline_keyboard\":[[{\"text\":\"Открыть Stay\",\"url\":\"$OPEN_STAY_URL\"}]]}" >/dev/null; then
    sent=$((sent + 1))
    echo "sent: $chat_id"
  else
    failed=$((failed + 1))
    echo "failed: $chat_id"
  fi

  # ~20 msg/s — под лимит Telegram (30/s).
  sleep 0.05
done < <(ids_cmd)

echo "done. sent=$sent failed=$failed"
