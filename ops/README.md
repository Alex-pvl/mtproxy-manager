# Эксплуатация: мониторинг, алерты, бэкапы

Всё ставится на сервер приложения (там, где бэкенд и Postgres).
Сервисы мониторинга слушают только `127.0.0.1`.

## 1. Метрики бэкенда

Бэкенд отдаёт Prometheus-метрики на `METRICS_ADDR` (по умолчанию
`127.0.0.1:9464/metrics`, наружу не торчит). Ничего настраивать не нужно,
достаточно обновить бэкенд.

## 2. Prometheus + Grafana

Нужен Docker с compose-плагином.

```bash
cd ops/monitoring
echo "GRAFANA_ADMIN_PASSWORD=$(openssl rand -base64 18)" > .env && chmod 600 .env
sudo mkdir -p /var/lib/node_exporter/textfile
docker compose up -d
```

Уведомления пока не отправляются: сработавшие алерты видно в Prometheus
(`ssh -L 9090:127.0.0.1:9090 ...` → http://localhost:9090/alerts) и в Grafana.

Grafana — через SSH-туннель, логин `admin`, пароль из `ops/monitoring/.env`:

```bash
ssh -L 3001:127.0.0.1:3001 root@<сервер-приложения>
# открыть http://localhost:3001 → Dashboards → Stay → «Stay — обзор»
```

Что алертит (`alerts.yml`): бэкенд не отвечает, оплата прошла, но подписка
не активировалась, 5xx > 5%, медленный API, 3x-ui недоступна или отвечает
ошибками, ошибки Telegram Bot API, сайт или сервер 3x-ui недоступен, TLS
истекает < 14 дней, бэкап упал или его не было > 17 дней, диск/память/нагрузка.

## 3. Бэкапы базы на сервер 3x-ui

Раз в две недели (1-го и 15-го в 03:30) `pg_dump` в custom-формате,
проверка дампа, хранение 8 последних (≈ 4 месяца) локально в
`/var/backups/stay` и зеркально на сервере 3x-ui. SSH-ключ на той стороне
ограничен `rrsync` одной папкой — с ним нельзя ни зайти в shell, ни писать
за пределы каталога бэкапов.

**На сервере 3x-ui (tagwaiter.ru):**

```bash
# shell must be a real one: sshd runs the forced rrsync command through it;
# the key's command=/restrict options are what lock the account down
sudo useradd -m -s /bin/sh stay-backup
sudo -u stay-backup mkdir -p /home/stay-backup/dumps /home/stay-backup/.ssh
sudo chmod 700 /home/stay-backup/.ssh
which rrsync   # есть в пакете rsync (Ubuntu 22.04+/Debian 12); иначе /usr/share/doc/rsync/scripts/rrsync
```

**На сервере приложения:**

```bash
sudo apt install -y postgresql-client rsync   # версия pg_dump >= версии сервера Postgres
sudo ssh-keygen -t ed25519 -N '' -f /root/.ssh/stay_backup -C stay-backup
sudo cat /root/.ssh/stay_backup.pub
```

Эту строку — на сервер 3x-ui в `/home/stay-backup/.ssh/authorized_keys`, с префиксом:

```
command="/usr/bin/rrsync /home/stay-backup/dumps",restrict ssh-ed25519 AAAA... stay-backup
```

(потом `chown stay-backup: /home/stay-backup/.ssh/authorized_keys && chmod 600 ...`).

Дальше на сервере приложения:

```bash
sudo install -m 755 ops/backup/stay-backup.sh /usr/local/bin/stay-backup.sh
sudo install -m 600 ops/backup/stay-backup.env.example /etc/stay-backup.env   # поправь при необходимости
sudo install -m 644 ops/backup/stay-backup.service ops/backup/stay-backup.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now stay-backup.timer
sudo systemctl start stay-backup.service && journalctl -u stay-backup -n 20   # первый бэкап сразу
```

Первый запуск руками — чтобы сразу убедиться, что всё работает (иначе через час загорится алерт «бэкапа нет»).

**Восстановление:**

```bash
# с сервера 3x-ui (scp через rrsync не работает, только rsync)
rsync -e "ssh -i /root/.ssh/stay_backup" stay-backup@tagwaiter.ru:stay-XXXX.dump /tmp/
sudo systemctl stop mtproxy-manager
pg_restore --clean --if-exists --no-owner --dbname="$DATABASE_URL" /tmp/stay-XXXX.dump
sudo systemctl start mtproxy-manager
```

Клиенты в 3x-ui восстанавливать отдельно не нужно, пока жива сама панель: в базе
лежат их UUID, имена и сроки.
