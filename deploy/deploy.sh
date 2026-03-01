#!/bin/bash
# MTProxy Manager — деплой бэкенда на хост
# Запуск: ./deploy/deploy.sh [путь к репо]

set -e

REPO_DIR="${1:-$(cd "$(dirname "$0")/.." && pwd)}"
INSTALL_DIR="/opt/mtproxy-manager"

echo "==> MTProxy Manager: деплой бэкенда"
echo "    Репо: $REPO_DIR"
echo "    Установка: $INSTALL_DIR"
echo ""

# 1. Сборка бэкенда (CGO не нужен — используется lib/pq)
echo "==> Сборка бэкенда..."
cd "$REPO_DIR/backend"
CGO_ENABLED=0 go build -o mtproxy-manager ./cmd/server

# 2. Создание директории
echo "==> Создание директорий..."
sudo mkdir -p "$INSTALL_DIR"

# 3. Копирование бинарника и .env
echo "==> Копирование файлов..."
sudo cp "$REPO_DIR/backend/mtproxy-manager" "$INSTALL_DIR/"
if [ -f "$REPO_DIR/.env" ]; then
  sudo cp "$REPO_DIR/.env" "$INSTALL_DIR/.env"
else
  sudo cp "$REPO_DIR/.env.example" "$INSTALL_DIR/.env"
  echo "    Внимание: скопирован .env.example — отредактируйте $INSTALL_DIR/.env"
fi

# 4. Права
echo "==> Настройка прав..."
sudo chown -R root:root "$INSTALL_DIR"
sudo chmod 755 "$INSTALL_DIR/mtproxy-manager"

# 5. Systemd
echo "==> Установка systemd сервиса..."
sudo cp "$REPO_DIR/deploy/mtproxy-manager.service" /etc/systemd/system/
sudo systemctl daemon-reload

echo ""
echo "==> Готово!"
echo ""
echo "Дальнейшие шаги:"
echo "  1. Проверьте .env: $INSTALL_DIR/.env"
echo "     Обязательно: JWT_SECRET, ADMIN_PASSWORD, DATABASE_URL, BASE_URL"
echo "  2. Запуск:    sudo systemctl start mtproxy-manager"
echo "  3. Автозапуск: sudo systemctl enable mtproxy-manager"
echo "  4. Nginx: скопируйте deploy/nginx.conf в /etc/nginx/sites-available/staytg.org"
echo ""
echo "Сборка фронтенда выполняется отдельно:"
echo "  cd frontend && npm ci && npm run build"
echo "  sudo rsync -a --delete dist/ /var/www/staytg.org/"
echo ""
