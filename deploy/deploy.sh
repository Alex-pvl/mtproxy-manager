#!/bin/bash
# MTProxy Manager — деплой бэкенда + фронтенда на хост
# Запуск: ./deploy/deploy.sh [путь к репо]

set -e

REPO_DIR="${1:-$(cd "$(dirname "$0")/.." && pwd)}"
INSTALL_DIR="/opt/mtproxy-manager"
WEB_DIR="/var/www/staytg.org"

echo "==> MTProxy Manager: полный деплой"
echo "    Репо: $REPO_DIR"
echo "    Бэкенд: $INSTALL_DIR"
echo "    Фронтенд: $WEB_DIR"
echo ""

# 0. Загрузка переменных из .env (для VITE_TG_CLIENT_ID и других)
ENV_FILE="$REPO_DIR/.env"
if [ -f "$ENV_FILE" ]; then
  set -a
  source "$ENV_FILE"
  set +a
  echo "==> Загружены переменные из .env"
fi

# 1. Сборка бэкенда (CGO не нужен — используется lib/pq)
echo "==> Сборка бэкенда..."
cd "$REPO_DIR/backend"
CGO_ENABLED=0 go build -o mtproxy-manager ./cmd/server

# 2. Сборка фронтенда (VITE_TG_CLIENT_ID берётся из окружения)
echo "==> Сборка фронтенда..."
cd "$REPO_DIR/frontend"
npm ci
npm run build

# 3. Создание директорий
echo "==> Создание директорий..."
sudo mkdir -p "$INSTALL_DIR"
sudo mkdir -p "$WEB_DIR"

# 4. Копирование бинарника и .env
echo "==> Копирование файлов бэкенда..."
sudo cp "$REPO_DIR/backend/mtproxy-manager" "$INSTALL_DIR/"
if [ -f "$REPO_DIR/.env" ]; then
  sudo cp "$REPO_DIR/.env" "$INSTALL_DIR/.env"
else
  sudo cp "$REPO_DIR/.env.example" "$INSTALL_DIR/.env"
  echo "    Внимание: скопирован .env.example — отредактируйте $INSTALL_DIR/.env"
fi

# 5. Копирование фронтенда
echo "==> Копирование файлов фронтенда..."
sudo rsync -a --delete "$REPO_DIR/frontend/dist/" "$WEB_DIR/"
sudo chown -R www-data:www-data "$WEB_DIR"

# 6. Права
echo "==> Настройка прав..."
sudo chown -R root:root "$INSTALL_DIR"
sudo chmod 755 "$INSTALL_DIR/mtproxy-manager"

# 7. Systemd
echo "==> Установка systemd сервиса..."
sudo cp "$REPO_DIR/deploy/mtproxy-manager.service" /etc/systemd/system/
sudo systemctl daemon-reload

echo ""
echo "==> Готово!"
echo ""
echo "Дальнейшие шаги:"
echo "  1. Проверьте .env: $INSTALL_DIR/.env"
echo "     Обязательно: JWT_SECRET, DATABASE_URL, BASE_URL, TG_BOT_TOKEN, VITE_TG_CLIENT_ID"
echo "  2. Запуск:    sudo systemctl restart mtproxy-manager"
echo "  3. Автозапуск: sudo systemctl enable mtproxy-manager"
echo "  4. Nginx: скопируйте deploy/nginx.conf в /etc/nginx/sites-available/staytg.org"
echo ""
