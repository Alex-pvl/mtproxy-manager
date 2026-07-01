-- Выставляет пользователю подписку на :months месяцев (от текущего момента).
--
-- Запуск:
--   psql "$DATABASE_URL" -v uid=42 -v months=3 -f deploy/grant_subscription.sql
--
-- uid    — users.id
-- months — срок подписки в месяцах
--
-- Что делает: добавляет строку в subscriptions (активной считается запись с
-- наибольшим expires_at в будущем) и поднимает users.max_proxies под срок —
-- ровно как это делает бэкенд при оплате. Старые подписки не трогает.

\set ON_ERROR_STOP on

BEGIN;

INSERT INTO subscriptions (user_id, plan_id, payment_id, starts_at, expires_at)
SELECT
  :uid,
  CASE
    WHEN :months <= 1 THEN 'month_1'
    WHEN :months <= 3 THEN 'month_3'
    WHEN :months <= 6 THEN 'month_6'
    ELSE 'year_1'
  END,
  0,                                        -- payment_id = 0: ручная выдача
  NOW(),
  NOW() + make_interval(months => :months::int);

UPDATE users
SET max_proxies = CASE
    WHEN :months <= 1 THEN 1
    WHEN :months <= 3 THEN 3
    WHEN :months <= 6 THEN 5
    ELSE 10
  END
WHERE id = :uid;

COMMIT;

-- Проверка результата:
SELECT u.id, u.username, u.max_proxies, s.plan_id, s.starts_at, s.expires_at
FROM users u
JOIN subscriptions s ON s.user_id = u.id
WHERE u.id = :uid
ORDER BY s.expires_at DESC
LIMIT 1;
