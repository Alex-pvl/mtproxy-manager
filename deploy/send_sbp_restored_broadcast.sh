#!/usr/bin/env bash
set -euo pipefail
# Disable bash history expansion so "!" in message text does not break.
set +H

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IDS_FILE="${1:-$ROOT_DIR/deploy/telegram_ids.txt}"

if [[ -z "${TG_BOT_TOKEN:-}" ]]; then
  echo "TG_BOT_TOKEN is not set"
  exit 1
fi

if [[ ! -f "$IDS_FILE" ]]; then
  echo "IDs file not found: $IDS_FILE"
  exit 1
fi

MESSAGE="$(cat <<'EOF'
🎉 СБП снова работает — технические работы платежного шлюза завершены!

🎁 В честь этого мы сделали скидку 20% на все тарифы Stay.

👇🏻 Откройте приложение и выберите подходящий план.
EOF
)"
API_URL="https://api.telegram.org/bot${TG_BOT_TOKEN}/sendMessage"
OPEN_STAY_URL="${OPEN_STAY_URL:-${TELEGRAM_PAY_URL:-${BASE_URL:-https://t.me/staytg_bot/stay}}}"

if [[ -z "$MESSAGE" ]]; then
  echo "MESSAGE is empty"
  exit 1
fi

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

  if curl -sS --fail -X POST "$API_URL" \
    --data-urlencode "chat_id=$chat_id" \
    --data-urlencode "text=$MESSAGE" \
    --data "disable_web_page_preview=true" \
    --data-urlencode "reply_markup={\"inline_keyboard\":[[{\"text\":\"Открыть Stay\",\"url\":\"$OPEN_STAY_URL\"}]]}" >/dev/null; then
    sent=$((sent + 1))
    echo "sent: $chat_id"
  else
    failed=$((failed + 1))
    echo "failed: $chat_id"
  fi

  sleep 0.05
done < "$IDS_FILE"

echo "done. sent=$sent failed=$failed"
