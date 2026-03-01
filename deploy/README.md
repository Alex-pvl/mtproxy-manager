# Деплой MTProxy Manager на хост (без Docker)

Бэкенд (Go, порт 3000) + PostgreSQL + фронтенд (статика через Nginx).
Docker нужен только для MTProxy/SOCKS5-контейнеров.

## Требования

- Linux (Ubuntu 22.04 / Debian 12)
- Go 1.24+
- Node.js 22+
- PostgreSQL 15+
- Nginx
- Docker (для прокси-контейнеров)
- Certbot + Let's Encrypt (для SSL)

---

## 1. Установка PostgreSQL

```bash
sudo apt update
sudo apt install -y postgresql postgresql-contrib

# Запускаем и включаем автостарт
sudo systemctl enable --now postgresql
```

### Создание пользователя и базы данных

```bash
sudo -u postgres psql <<EOF
CREATE USER mtproxy WITH PASSWORD 'yourpassword';
CREATE DATABASE mtproxy OWNER mtproxy;
GRANT ALL PRIVILEGES ON DATABASE mtproxy TO mtproxy;
EOF
```

### Проверка подключения

```bash
psql -h localhost -U mtproxy -d mtproxy -c '\l'
```

---

## 2. Конфигурация (.env)

```bash
cp .env.example .env
nano .env
```

Обязательные параметры:

```env
JWT_SECRET=ваш-секретный-ключ-минимум-32-символа
ADMIN_USERNAME=admin
ADMIN_PASSWORD=ваш-пароль-минимум-6-символов

DATABASE_URL=postgres://mtproxy:yourpassword@localhost:5432/mtproxy?sslmode=disable

SERVER_IP=ваш-IP-сервера
SERVER_PORT=3000
BASE_URL=https://staytg.org

CRYPTOBOT_TOKEN=токен-из-@CryptoBot

PORT_MIN=8000
PORT_MAX=9999
SOCKS5_PORT_MIN=10000
SOCKS5_PORT_MAX=10999
DEFAULT_MAX_PROXIES=5
```

---

## 3. Сборка и деплой бэкенда

```bash
# Клонировать репо
git clone <repo-url> mtproxy-manager
cd mtproxy-manager

# Создать .env
cp .env.example .env
# Отредактировать .env (см. раздел выше)

# Запустить скрипт деплоя
chmod +x deploy/deploy.sh
./deploy/deploy.sh
```

Или вручную:

```bash
# Сборка (CGO не нужен — используем lib/pq)
cd backend
CGO_ENABLED=0 go build -o mtproxy-manager ./cmd/server

# Установка
sudo mkdir -p /opt/mtproxy-manager
sudo cp mtproxy-manager /opt/mtproxy-manager/
sudo cp ../.env /opt/mtproxy-manager/.env
```

### Systemd-сервис

```bash
sudo cp deploy/mtproxy-manager.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable mtproxy-manager
sudo systemctl start mtproxy-manager

# Проверка
sudo systemctl status mtproxy-manager
sudo journalctl -u mtproxy-manager -f
```

Структура на хосте:

```
/opt/mtproxy-manager/
├── mtproxy-manager   # скомпилированный бинарник
└── .env
```

---

## 4. Сборка и деплой фронтенда

```bash
cd frontend

# Установить зависимости
npm ci

# Собрать production-сборку
npm run build
# Результат: frontend/dist/
```

### Копирование в /var/www/staytg.org/

```bash
# Создать директорию
sudo mkdir -p /var/www/staytg.org

# Скопировать файлы (--delete удаляет старые файлы)
sudo rsync -a --delete dist/ /var/www/staytg.org/

# Права для Nginx
sudo chown -R www-data:www-data /var/www/staytg.org
```

При обновлении фронтенда:

```bash
cd frontend && npm ci && npm run build
sudo rsync -a --delete dist/ /var/www/staytg.org/
```

---

## 5. Настройка Nginx

### Установка Certbot и получение SSL-сертификата

```bash
sudo apt install -y certbot python3-certbot-nginx

# Получить сертификат (домен должен уже указывать на сервер)
sudo certbot certonly --nginx -d staytg.org -d www.staytg.org
```

### Конфиг Nginx

```bash
sudo cp deploy/nginx.conf /etc/nginx/sites-available/staytg.org
sudo ln -s /etc/nginx/sites-available/staytg.org /etc/nginx/sites-enabled/

# Проверка конфига
sudo nginx -t

# Перезагрузка
sudo systemctl reload nginx
```

Схема работы:

```
Клиент → Nginx (443) → /api/* → Go-бэкенд :3000
                     → /*     → /var/www/staytg.org (статика React)
```

---

## 6. Обновление

```bash
cd mtproxy-manager
git pull

# Пересборка и деплой бэкенда
./deploy/deploy.sh
sudo systemctl restart mtproxy-manager

# Пересборка фронтенда
cd frontend && npm ci && npm run build
sudo rsync -a --delete dist/ /var/www/staytg.org/
```

---

## Диагностика

```bash
# Логи бэкенда
sudo journalctl -u mtproxy-manager -f

# Проверить, что бэкенд слушает порт 3000
ss -tlnp | grep 3000

# Проверить подключение к PostgreSQL
psql "$DATABASE_URL" -c 'SELECT NOW()'

# Проверить Nginx
sudo nginx -t
sudo journalctl -u nginx -f
```
