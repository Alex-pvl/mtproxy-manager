#!/bin/bash
# Удаляет старые записи proxies без VLESS (созданные до появления VPN),
# из-за которых у пользователей в списке висят пустые карточки.
#
# Использование:
#   ./deploy/cleanup_empty_vpn.sh          # dry-run: показать, сколько удалится
#   ./deploy/cleanup_empty_vpn.sh --yes    # удалить
#
# DATABASE_URL берётся из /opt/mtproxy-manager/.env (переопредели через ENV_FILE=/path/.env).
set -euo pipefail

ENV_FILE="${ENV_FILE:-/opt/mtproxy-manager/.env}"
if [ -f "$ENV_FILE" ]; then
  set -a; source "$ENV_FILE"; set +a
fi
: "${DATABASE_URL:?DATABASE_URL не задан — укажи ENV_FILE=/path/.env или экспортируй переменную}"

# Пустая карточка = запись без VLESS-конфига (старый mtproxy/socks5 или неудавшийся x-ui).
WHERE="vless_uuid IS NULL OR vless_uuid = ''"

count=$(psql "$DATABASE_URL" -tAc "SELECT count(*) FROM proxies WHERE $WHERE;")
echo "Пустых VPN-записей к удалению: $count"

if [ "${1:-}" != "--yes" ]; then
  echo "Это dry-run. Для удаления запусти: $0 --yes"
  exit 0
fi

psql "$DATABASE_URL" -c "DELETE FROM proxies WHERE $WHERE;"
echo "Удалено: $count"
