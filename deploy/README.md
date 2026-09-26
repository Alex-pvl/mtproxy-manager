# Деплой Stay VPN на хост

Бэкенд (Go, порт 3000) + PostgreSQL + фронтенд (статика через Nginx).
VPN — клиенты в панели 3x-ui v3 на отдельном сервере.

## Требования

- Linux (Ubuntu 22.04 / Debian 12)
- Go 1.24+
- Node.js 22+
- PostgreSQL 15+
- Nginx
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

Создай `.env` в корне репозитория:

```env
JWT_SECRET=ваш-секретный-ключ-минимум-32-символа
DATABASE_URL=postgres://mtproxy:yourpassword@localhost:5432/mtproxy?sslmode=disable
SERVER_PORT=3000
BASE_URL=https://staytg.org
DEFAULT_MAX_PROXIES=5
METRICS_ADDR=127.0.0.1:9464   # Prometheus, см. ops/README.md

# Админ: логин+пароль и/или Telegram ID
ADMIN_USERNAME=admin
ADMIN_PASSWORD=
ADMIN_TELEGRAM_ID=

# Telegram-бот и вход через Telegram (@BotFather → Bot Settings → Web Login)
TG_BOT_TOKEN=
TG_BOT_USERNAME=           # без @
TG_PAY_URL=https://t.me/staytg_bot/pay
TG_CLIENT_ID=
TG_CLIENT_SECRET=
VITE_TG_CLIENT_ID=         # = TG_CLIENT_ID, встраивается во фронт при сборке
# Секрет вебхука бота: тот же, что secret_token в setWebhook (см. ниже)
TG_WEBHOOK_SECRET=

# Оплата
CRYPTOBOT_TOKEN=
DIGITALPAY_API_KEY=
DIGITALPAY_BASE_URL=https://digitalpay.cc
DIGITALPAY_SBP_BACK_URL=
TON_WALLET_ADDRESS=

# Панель 3x-ui v3
XUI_URL=https://tagwaiter.ru:<порт-панели>
XUI_PATH_PREFIX=<base-path-панели>
XUI_API_TOKEN=             # Настройки → Безопасность → API Token
XUI_INBOUND_ID=1
XUI_SUB_URL=https://tagwaiter.ru:2096/<путь-подписки>/
```

Значения с пробелами бери в кавычки, иначе `source .env` сломается.

### Вебхук бота

Бот принимает оплату звёздами через `/api/webhook/bot`. Чтобы никто, кроме
Telegram, не мог туда постучаться, задай `TG_WEBHOOK_SECRET` и передай его
при регистрации вебхука:

```bash
curl "https://api.telegram.org/bot$TG_BOT_TOKEN/setWebhook" \
  -d url=https://staytg.org/api/webhook/bot -d secret_token=$TG_WEBHOOK_SECRET
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
cd backend && CGO_ENABLED=0 go build -o mtproxy-manager ./cmd/server

# Установка
rm -rf /opt/mtproxy-manager
sudo mkdir -p /opt/mtproxy-manager
sudo cp mtproxy-manager /opt/mtproxy-manager/
sudo cp ../.env /opt/mtproxy-manager/.env
sudo systemctl restart mtproxy-manager
sudo systemctl status mtproxy-manager
```

### Systemd-сервис

```bash
sudo cp deploy/mtproxy-manager.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable mtproxy-manager


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

> **Важно:** переменная `VITE_TG_CLIENT_ID` должна быть задана при сборке —
> Vite встраивает её в бандл на этапе `npm run build`.

```bash
# Загрузить переменные из .env (для VITE_TG_CLIENT_ID)
set -a && source ../.env && set +a

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
git fetch && git pull
set -a && source .env && set +a
cd frontend && npm ci && npm run build
sudo rsync -a --delete dist/ /var/www/staytg.org/dist/
sudo systemctl reload nginx
```

> Или используйте `deploy/deploy.sh` — он загружает `.env` и собирает всё автоматически.

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

# Полная пересборка и деплой (бэкенд + фронтенд)
./deploy/deploy.sh
sudo systemctl restart mtproxy-manager
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
