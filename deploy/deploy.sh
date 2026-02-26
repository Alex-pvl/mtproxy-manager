#!/bin/bash
# MTProxy Manager — деплой на хост (без Docker)
# Запуск: ./deploy/deploy.sh [путь к репо]

set -e

REPO_DIR="${1:-$(cd "$(dirname "$0")/.." && pwd)}"
INSTALL_DIR="/opt/mtproxy-manager"
DATA_DIR="/var/lib/mtproxy-manager"

echo "==> MTProxy Manager: деплой на хост"
echo "    Репо: $REPO_DIR"
echo "    Установка: $INSTALL_DIR"
echo "    Данные: $DATA_DIR"
echo ""

# 1. Сборка фронтенда
echo "==> Сборка фронтенда..."
cd "$REPO_DIR/frontend"
npm ci
npm run build

# 2. Сборка бэкенда (нужен Go и gcc для sqlite)
echo "==> Сборка бэкенда..."
cd "$REPO_DIR/backend"
CGO_ENABLED=1 go build -o mtproxy-manager ./cmd/server

# 3. Создание директорий
echo "==> Создание директорий..."
sudo mkdir -p "$INSTALL_DIR"
sudo mkdir -p "$DATA_DIR"

# 4. Копирование файлов
echo "==> Копирование файлов..."
sudo cp "$REPO_DIR/backend/mtproxy-manager" "$INSTALL_DIR/"
sudo mkdir -p "$INSTALL_DIR/frontend"
sudo cp -r "$REPO_DIR/frontend/dist" "$INSTALL_DIR/frontend/"
if [ -f "$REPO_DIR/.env" ]; then
  sudo cp "$REPO_DIR/.env" "$INSTALL_DIR/.env"
else
  sudo cp "$REPO_DIR/.env.example" "$INSTALL_DIR/.env"
  echo "    Внимание: скопирован .env.example — отредактируйте $INSTALL_DIR/.env"
fi

# 5. Обновление DB_PATH в .env для хоста
if [ -f "$INSTALL_DIR/.env" ]; then
  if grep -q "DB_PATH=" "$INSTALL_DIR/.env"; then
    sudo sed -i "s|DB_PATH=.*|DB_PATH=$DATA_DIR/mtproxy.db|" "$INSTALL_DIR/.env"
  else
    echo "DB_PATH=$DATA_DIR/mtproxy.db" | sudo tee -a "$INSTALL_DIR/.env"
  fi
fi

# 6. Права
echo "==> Настройка прав..."
sudo chown -R root:root "$INSTALL_DIR"
sudo chmod 755 "$INSTALL_DIR/mtproxy-manager"
sudo chmod 755 "$DATA_DIR"

# 7. Systemd
echo "==> Установка systemd сервиса..."
sudo cp "$REPO_DIR/deploy/mtproxy-manager.service" /etc/systemd/system/
sudo systemctl daemon-reload

echo ""
echo "==> Готово!"
echo ""
echo "Дальнейшие шаги:"
echo "  1. Проверьте .env в $INSTALL_DIR/.env"
echo "  2. Убедитесь что DB_PATH=$DATA_DIR/mtproxy.db"
echo "  3. Запуск: sudo systemctl start mtproxy-manager"
echo "  4. Автозапуск: sudo systemctl enable mtproxy-manager"
echo "  5. Nginx: скопируйте deploy/nginx.conf в /etc/nginx/sites-available/"
echo ""
