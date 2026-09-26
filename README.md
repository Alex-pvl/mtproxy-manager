# Stay VPN

Telegram Mini App and website that sells VPN subscriptions (1/3/6/12 months).

- **backend/** — Go API: auth (Telegram OIDC, Mini App initData, login/password),
  payments (CryptoBot, SBP via DigitalPay, TON, Telegram Stars), subscriptions,
  referrals, admin. VPN configs are clients in a 3x-ui v3 panel; users get a
  subscription URL that stops working when the paid period ends.
- **frontend/** — React + Vite SPA served by nginx.

Deployment and configuration: [deploy/README.md](deploy/README.md).

Tests: `cd backend && go test ./...`. The DB test runs only with
`TEST_DATABASE_URL` pointing at a throwaway Postgres (it wipes the schema).
