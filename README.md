<div align="center">

# Stay

**Telegram Mini App и сайт для продажи подписок на защищённое подключение.**
Оплата в пару кликов, ссылка на подключение выдаётся сразу и работает ровно до конца оплаченного периода.

[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](backend/go.mod)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black)](frontend/package.json)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?logo=typescript&logoColor=white)](frontend/package.json)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-4169E1?logo=postgresql&logoColor=white)](backend/internal/database/db.go)
[![Telegram Mini App](https://img.shields.io/badge/Telegram-Mini_App-26A5E4?logo=telegram&logoColor=white)](https://core.telegram.org/bots/webapps)
[![Prometheus](https://img.shields.io/badge/Prometheus-+_Grafana-E6522C?logo=prometheus&logoColor=white)](ops/README.md)

[Возможности](#возможности) · [Скриншоты](#скриншоты) · [Как устроено](#как-устроено) · [Запуск](#запуск) · [Эксплуатация](ops/README.md)

</div>

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/home-dark.png">
    <img alt="Главная страница" src="docs/screenshots/home-light.png" width="100%">
  </picture>
</p>

## Возможности

| | |
|---|---|
| 💳 **Четыре способа оплаты** | CryptoBot, СБП (DigitalPay), TON через TON Connect, Telegram Stars |
| ⏳ **Подписки 1 / 3 / 6 / 12 месяцев** | Продление добавляет срок к текущему, а не сгорает |
| 🔗 **Ссылки-подписки** | Клиенты создаются в панели 3x-ui, ссылка перестаёт работать в день окончания подписки |
| 📱 **Telegram Mini App** | Вход по `initData`, оплата звёздами прямо в Telegram, уведомления от бота |
| 🔐 **Три способа входа** | Telegram OIDC, Mini App, логин и пароль |
| 🎁 **Рефералы** | Пригласивший получает 15% дней от каждой оплаты друга |
| 🛠 **Админка** | Пользователи, роли, лимиты устройств, все подключения |
| 📊 **Мониторинг и бэкапы** | Метрики Prometheus, дашборд Grafana, дамп базы раз в две недели |

## Скриншоты

<table>
  <tr>
    <td width="50%">
      <picture>
        <source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/pricing-dark.png">
        <img alt="Тарифы" src="docs/screenshots/pricing-light.png">
      </picture>
      <p align="center"><b>Тарифы и способы оплаты</b></p>
    </td>
    <td width="50%">
      <picture>
        <source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/services-dark.png">
        <img alt="Мои сервисы" src="docs/screenshots/services-light.png">
      </picture>
      <p align="center"><b>Подписка и ссылки на подключение</b></p>
    </td>
  </tr>
</table>

<details>
<summary><b>📱 Как это выглядит в Telegram</b></summary>
<br>
<p align="center">
  <img alt="Главная в Mini App" src="docs/screenshots/mobile-home-dark.png" width="30%">
  &nbsp;
  <img alt="Тарифы в Mini App" src="docs/screenshots/mobile-pricing-dark.png" width="30%">
  &nbsp;
  <img alt="Подключения в Mini App" src="docs/screenshots/mobile-services-dark.png" width="30%">
</p>
</details>

## Как устроено

```mermaid
flowchart LR
    U(["Пользователь<br/>сайт или Mini App"]) -->|HTTPS| N[nginx]
    N -->|статика| F["frontend<br/>React + Vite"]
    N -->|/api| B["backend<br/>Go + chi"]
    B --> DB[(PostgreSQL)]
    B -->|клиенты и сроки| X["панель 3x-ui"]
    B <-->|счета и вебхуки| P["CryptoBot · DigitalPay<br/>TON · Telegram Stars"]
    B -->|уведомления| T["Telegram Bot API"]
    U -.->|ссылка-подписка| X
    PR[Prometheus] -->|/metrics| B
    G[Grafana] --> PR
```

<details>
<summary><b>Путь оплаты</b></summary>
<br>

```mermaid
sequenceDiagram
    actor U as Пользователь
    participant S as Stay backend
    participant P as Платёжка
    participant X as 3x-ui
    U->>S: выбрал тариф и способ оплаты
    S->>P: создать счёт
    S-->>U: ссылка на оплату
    U->>P: оплатил
    P->>S: вебхук (или проверка по кнопке)
    S->>S: pending → paid (ровно один раз)
    S->>S: подписка + реферальный бонус
    S->>X: продлить срок всех подключений
    S-->>U: сообщение от бота
```
</details>

<details>
<summary><b>Структура репозитория</b></summary>
<br>

```
backend/          Go API
  cmd/server/       точка входа, роутинг
  internal/
    handlers/       HTTP: авторизация, оплаты, подключения, админка, бот
    database/       Postgres, миграции при старте
    xui/            клиент API 3x-ui v3
frontend/         React SPA (Vite, Tailwind, TON Connect)
ops/
  monitoring/       Prometheus, Grafana, экспортеры (docker compose)
  backup/           pg_dump раз в две недели + systemd timer
deploy/           инструкция по деплою на хост
docs/screenshots/ картинки для этого README
```
</details>

## Запуск

```bash
# бэкенд: нужен PostgreSQL и .env (переменные в deploy/README.md);
# 8080 — порт, на который dev-сервер фронта проксирует /api
cd backend && SERVER_PORT=8080 go run ./cmd/server

# фронт: http://localhost:5173
cd frontend && npm ci && npm run dev
```

Тесты:

```bash
cd backend && go test ./...
```

Тест базы запускается только с переменной `TEST_DATABASE_URL`, указывающей на одноразовый Postgres: он пересоздаёт схему.

Продакшн-деплой описан в [deploy/README.md](deploy/README.md), мониторинг и бэкапы в [ops/README.md](ops/README.md).
