# Деплой MTProxy Manager на хост (без Docker)

Бэкенд, фронтенд и SQLite работают на хосте. Docker нужен только для контейнеров MTProxy (mtg).

## Требования

- Linux (Ubuntu/Debian)
- Go 1.24+
- Node.js 22+
- Docker (для прокси-контейнеров)
- Nginx (для HTTPS)

## Быстрый деплой

```bash
# 1. Клонировать репо
git clone <repo-url> mtproxy-manager
cd mtproxy-manager

# 2. Создать .env из примера
cp .env.example .env
# Отредактировать .env — обязательно JWT_SECRET, ADMIN_PASSWORD, CRYPTOBOT_TOKEN, BASE_URL

# 3. Запустить деплой
chmod +x deploy/deploy.sh
./deploy/deploy.sh
```

## Ручная установка

### 1. Сборка

```bash
# Фронтенд
cd frontend && npm ci && npm run build && cd ..

# Бэкенд (нужен gcc для sqlite)
cd backend && CGO_ENABLED=1 go build -o mtproxy-manager ./cmd/server && cd ..
```

### 2. Установка

```bash
sudo mkdir -p /opt/mtproxy-manager /var/lib/mtproxy-manager
sudo cp backend/mtproxy-manager /opt/mtproxy-manager/
sudo cp -r frontend/dist /opt/mtproxy-manager/frontend/
sudo cp .env /opt/mtproxy-manager/
```

### 3. .env для хоста

В `/opt/mtproxy-manager/.env`:

```
DB_PATH=/var/lib/mtproxy-manager/mtproxy.db
SERVER_PORT=8080
# остальные переменные как в .env.example
```

### 4. Systemd

```bash
sudo cp deploy/mtproxy-manager.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable mtproxy-manager
sudo systemctl start mtproxy-manager
```

### 5. Nginx

```bash
sudo cp deploy/nginx.conf /etc/nginx/sites-available/mtproxy-manager
# Отредактировать server_name и пути к SSL-сертификатам
sudo ln -s /etc/nginx/sites-available/mtproxy-manager /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

## Обновление

```bash
cd mtproxy-manager
git pull
./deploy/deploy.sh
sudo systemctl restart mtproxy-manager
```

## Структура на хосте

```
/opt/mtproxy-manager/
├── mtproxy-manager      # бинарник
├── frontend/dist/       # статика SPA
└── .env

/var/lib/mtproxy-manager/
└── mtproxy.db           # SQLite
```

## Nginx

Конфиг не меняется по сравнению с Docker-вариантом: всё так же проксируется на `127.0.0.1:8080`. Бэкенд сам отдаёт API и статику фронта.
