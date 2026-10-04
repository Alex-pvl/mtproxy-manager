package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/models"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	conn *sql.DB
	cfg  *config.Config
}

func New(cfg *config.Config) (*DB, error) {
	conn, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db := &DB{conn: conn, cfg: cfg}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	for _, m := range []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			max_proxies INTEGER NOT NULL DEFAULT 5,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_id BIGINT DEFAULT 0",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS hide_sub_banner BOOLEAN NOT NULL DEFAULT FALSE",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id) WHERE telegram_id > 0",

		`CREATE TABLE IF NOT EXISTS proxies (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			vless_uuid TEXT NOT NULL DEFAULT '',
			vless_email TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		"ALTER TABLE proxies ADD COLUMN IF NOT EXISTS vless_uuid TEXT DEFAULT ''",
		"ALTER TABLE proxies ADD COLUMN IF NOT EXISTS vless_email TEXT DEFAULT ''",
		// Legacy MTProxy/SOCKS5 era: drop records without a VLESS client, name the
		// old VLESS clients the way they were created, then drop the dead columns.
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'proxies' AND column_name = 'port') THEN
				DELETE FROM proxies WHERE COALESCE(vless_uuid, '') = '';
				UPDATE proxies SET vless_email = 'proxy-' || port || '-user-' || user_id WHERE COALESCE(vless_email, '') = '';
				ALTER TABLE proxies
					DROP COLUMN port, DROP COLUMN domain, DROP COLUMN secret,
					DROP COLUMN container_id, DROP COLUMN container_name, DROP COLUMN status,
					DROP COLUMN IF EXISTS socks5_port, DROP COLUMN IF EXISTS socks5_user,
					DROP COLUMN IF EXISTS socks5_pass, DROP COLUMN IF EXISTS socks5_container_id,
					DROP COLUMN IF EXISTS socks5_container_name;
			END IF;
		END $$`,

		`CREATE TABLE IF NOT EXISTS payments (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			plan_id TEXT NOT NULL,
			external_id TEXT NOT NULL UNIQUE,
			amount TEXT NOT NULL DEFAULT '0',
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			plan_id TEXT NOT NULL,
			payment_id BIGINT NOT NULL DEFAULT 0,
			starts_at TIMESTAMPTZ NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		"ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS expiry_reminder_7d_sent BOOLEAN NOT NULL DEFAULT FALSE",
		"ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS expiry_reminder_1d_sent BOOLEAN NOT NULL DEFAULT FALSE",

		`CREATE TABLE IF NOT EXISTS referral_codes (
			user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			code TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS referrals (
			referrer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			referred_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (referrer_id, referred_id)
		)`,
		`CREATE TABLE IF NOT EXISTS referral_bonuses (
			id BIGSERIAL PRIMARY KEY,
			referrer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			referred_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			payment_id BIGINT NOT NULL,
			bonus_days INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	} {
		if _, err := db.conn.Exec(m); err != nil {
			return fmt.Errorf("%w\n%s", err, m)
		}
	}
	return db.ensureAdmin()
}

func (db *DB) ensureAdmin() error {
	adminUsername := strings.TrimSpace(db.cfg.AdminUsername)
	if adminUsername == "" {
		adminUsername = "admin"
	}

	passwordHash := ""
	if strings.TrimSpace(db.cfg.AdminPassword) != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(db.cfg.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		passwordHash = string(hash)
	}

	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users WHERE role = $1", models.RoleAdmin).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		if passwordHash != "" {
			_, _ = db.conn.Exec(
				"UPDATE users SET password_hash = $1 WHERE username = $2 AND role = $3 AND password_hash = ''",
				passwordHash, adminUsername, models.RoleAdmin,
			)
		}
		return nil
	}

	// Allow bootstrapping admin either via Telegram ID or via username/password only.
	// If both are empty, skip creating admin user.
	if db.cfg.AdminTelegramID == 0 && passwordHash == "" {
		return nil
	}

	_, err = db.conn.Exec(
		"INSERT INTO users (username, password_hash, role, max_proxies, telegram_id) VALUES ($1, $2, $3, $4, $5)",
		adminUsername, passwordHash, models.RoleAdmin, 100, db.cfg.AdminTelegramID,
	)
	return err
}

// --- User queries ---

func (db *DB) GetUserByTelegramID(telegramID int64) (*models.User, error) {
	u := &models.User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password_hash, role, max_proxies, COALESCE(telegram_id, 0), hide_sub_banner, created_at FROM users WHERE telegram_id = $1",
		telegramID,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MaxProxies, &u.TelegramID, &u.HideSubBanner, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) CreateUserByTelegram(telegramID int64, username string, referrerID *int64) (*models.User, error) {
	var id int64
	err := db.conn.QueryRow(
		"INSERT INTO users (username, password_hash, role, max_proxies, telegram_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		username, "", models.RoleUser, db.cfg.DefaultMaxProxies, telegramID,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:         id,
		Username:   username,
		Role:       models.RoleUser,
		MaxProxies: db.cfg.DefaultMaxProxies,
		TelegramID: telegramID,
		CreatedAt:  time.Now(),
	}

	if referrerID != nil && *referrerID > 0 && *referrerID != id {
		_ = db.CreateReferral(*referrerID, id)
	}

	return user, nil
}

func (db *DB) CreateUser(username, passwordHash string) (*models.User, error) {
	var id int64
	err := db.conn.QueryRow(
		"INSERT INTO users (username, password_hash, role, max_proxies, telegram_id) VALUES ($1, $2, $3, $4, 0) RETURNING id",
		username, passwordHash, models.RoleUser, db.cfg.DefaultMaxProxies,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         models.RoleUser,
		MaxProxies:   db.cfg.DefaultMaxProxies,
		CreatedAt:    time.Now(),
	}, nil
}

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	u := &models.User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password_hash, role, max_proxies, COALESCE(telegram_id, 0), hide_sub_banner, created_at FROM users WHERE username = $1",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MaxProxies, &u.TelegramID, &u.HideSubBanner, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) GetUserByID(id int64) (*models.User, error) {
	u := &models.User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password_hash, role, max_proxies, COALESCE(telegram_id, 0), hide_sub_banner, created_at FROM users WHERE id = $1",
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MaxProxies, &u.TelegramID, &u.HideSubBanner, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) ListUsers() ([]models.User, error) {
	rows, err := db.conn.Query("SELECT id, username, role, max_proxies, hide_sub_banner, created_at FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.MaxProxies, &u.HideSubBanner, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *DB) UpdateUser(id int64, role models.Role, maxProxies int, hideSubBanner bool) error {
	_, err := db.conn.Exec("UPDATE users SET role = $1, max_proxies = $2, hide_sub_banner = $3 WHERE id = $4", role, maxProxies, hideSubBanner, id)
	return err
}

func (db *DB) DeleteUser(id int64) error {
	_, err := db.conn.Exec("DELETE FROM users WHERE id = $1", id)
	return err
}

// --- Proxy queries ---

const proxyCols = "id, user_id, vless_uuid, vless_email, created_at"

func scanProxies(rows *sql.Rows) ([]models.Proxy, error) {
	defer rows.Close()
	proxies := []models.Proxy{}
	for rows.Next() {
		var p models.Proxy
		if err := rows.Scan(&p.ID, &p.UserID, &p.VlessUUID, &p.VlessEmail, &p.CreatedAt); err != nil {
			return nil, err
		}
		proxies = append(proxies, p)
	}
	return proxies, rows.Err()
}

func (db *DB) CreateProxy(p *models.Proxy) error {
	return db.conn.QueryRow(
		"INSERT INTO proxies (user_id, vless_uuid, vless_email) VALUES ($1, $2, $3) RETURNING id, created_at",
		p.UserID, p.VlessUUID, p.VlessEmail,
	).Scan(&p.ID, &p.CreatedAt)
}

func (db *DB) GetProxy(id int64) (*models.Proxy, error) {
	p := &models.Proxy{}
	err := db.conn.QueryRow("SELECT "+proxyCols+" FROM proxies WHERE id = $1", id).
		Scan(&p.ID, &p.UserID, &p.VlessUUID, &p.VlessEmail, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) ListProxiesByUser(userID int64) ([]models.Proxy, error) {
	rows, err := db.conn.Query("SELECT "+proxyCols+" FROM proxies WHERE user_id = $1 ORDER BY id", userID)
	if err != nil {
		return nil, err
	}
	return scanProxies(rows)
}

func (db *DB) ListAllProxies() ([]models.Proxy, error) {
	rows, err := db.conn.Query("SELECT " + proxyCols + " FROM proxies ORDER BY id")
	if err != nil {
		return nil, err
	}
	return scanProxies(rows)
}

func (db *DB) DeleteProxy(id int64) error {
	_, err := db.conn.Exec("DELETE FROM proxies WHERE id = $1", id)
	return err
}

func (db *DB) CountProxiesByUser(userID int64) (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM proxies WHERE user_id = $1", userID).Scan(&count)
	return count, err
}

// --- Payment queries ---

func (db *DB) CreatePayment(p *models.Payment) error {
	return db.conn.QueryRow(
		`INSERT INTO payments (user_id, plan_id, external_id, amount, status)
		 VALUES ($1, $2, $3, $4, 'pending') RETURNING id, status, created_at`,
		p.UserID, p.PlanID, p.ExternalID, p.Amount,
	).Scan(&p.ID, &p.Status, &p.CreatedAt)
}

func (db *DB) GetPaymentByExternalID(externalID string) (*models.Payment, error) {
	p := &models.Payment{}
	err := db.conn.QueryRow(
		"SELECT id, user_id, plan_id, external_id, amount, status, created_at FROM payments WHERE external_id = $1",
		externalID,
	).Scan(&p.ID, &p.UserID, &p.PlanID, &p.ExternalID, &p.Amount, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// MarkPaymentPaid flips a pending payment to paid and returns it. It returns
// (nil, nil) if the payment was already paid or canceled, so concurrent
// webhooks/polls fulfil a payment exactly once.
func (db *DB) MarkPaymentPaid(externalID string) (*models.Payment, error) {
	p := &models.Payment{}
	err := db.conn.QueryRow(
		`UPDATE payments SET status = 'paid' WHERE external_id = $1 AND status = 'pending'
		 RETURNING id, user_id, plan_id, external_id, amount, status, created_at`,
		externalID,
	).Scan(&p.ID, &p.UserID, &p.PlanID, &p.ExternalID, &p.Amount, &p.Status, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) CancelPayment(externalID string) error {
	_, err := db.conn.Exec("UPDATE payments SET status = 'canceled' WHERE external_id = $1 AND status = 'pending'", externalID)
	return err
}

// ReopenPayment returns a canceled payment to pending, for providers that report late payments.
func (db *DB) ReopenPayment(externalID string) error {
	_, err := db.conn.Exec("UPDATE payments SET status = 'pending' WHERE external_id = $1 AND status = 'canceled'", externalID)
	return err
}

func (db *DB) GetPendingPaymentsByUser(userID int64) ([]models.Payment, error) {
	rows, err := db.conn.Query(
		`SELECT id, user_id, plan_id, external_id, amount, status, created_at
		 FROM payments WHERE user_id = $1 AND status = 'pending' ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Payment
	for rows.Next() {
		var p models.Payment
		if err := rows.Scan(&p.ID, &p.UserID, &p.PlanID, &p.ExternalID, &p.Amount, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// --- Subscription queries ---

func (db *DB) CreateSubscription(s *models.Subscription) error {
	err := db.conn.QueryRow(
		`INSERT INTO subscriptions (user_id, plan_id, payment_id, starts_at, expires_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		s.UserID, s.PlanID, s.PaymentID, s.StartsAt, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt)
	return err
}

func (db *DB) GetActiveSubscription(userID int64) (*models.Subscription, error) {
	s := &models.Subscription{}
	err := db.conn.QueryRow(
		`SELECT id, user_id, plan_id, payment_id, starts_at, expires_at, created_at
		 FROM subscriptions WHERE user_id = $1 AND expires_at > NOW() ORDER BY expires_at DESC LIMIT 1`,
		userID,
	).Scan(&s.ID, &s.UserID, &s.PlanID, &s.PaymentID, &s.StartsAt, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// SubscriptionExpiryReminderRow is the user's current active subscription (latest expires_at) due for a calendar-day reminder.
type SubscriptionExpiryReminderRow struct {
	ID         int64
	UserID     int64
	PlanID     string
	ExpiresAt  time.Time
	TelegramID int64
}

// ListSubscriptionsExpiryCalendarDays lists active subscriptions where UTC calendar days until expiry equals daysLeft (e.g. 7 or 1), reminder not yet sent, user has telegram_id.
func (db *DB) ListSubscriptionsExpiryCalendarDays(daysLeft int, reminder7d bool) ([]SubscriptionExpiryReminderRow, error) {
	var sentCol string
	if reminder7d {
		sentCol = "expiry_reminder_7d_sent"
	} else {
		sentCol = "expiry_reminder_1d_sent"
	}
	q := fmt.Sprintf(`
		SELECT s.id, s.user_id, s.plan_id, s.expires_at, u.telegram_id
		FROM subscriptions s
		INNER JOIN (
			SELECT user_id, MAX(expires_at) AS max_exp
			FROM subscriptions
			WHERE expires_at > NOW()
			GROUP BY user_id
		) latest ON latest.user_id = s.user_id AND latest.max_exp = s.expires_at
		INNER JOIN users u ON u.id = s.user_id AND COALESCE(u.telegram_id, 0) > 0
		WHERE NOT s.%s
		  AND (s.expires_at AT TIME ZONE 'UTC')::date - (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::date = $1`,
		sentCol)
	rows, err := db.conn.Query(q, daysLeft)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubscriptionExpiryReminderRow
	for rows.Next() {
		var r SubscriptionExpiryReminderRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.PlanID, &r.ExpiresAt, &r.TelegramID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (db *DB) MarkSubscriptionExpiryReminder7dSent(subscriptionID int64) error {
	_, err := db.conn.Exec(`UPDATE subscriptions SET expiry_reminder_7d_sent = TRUE WHERE id = $1`, subscriptionID)
	return err
}

func (db *DB) MarkSubscriptionExpiryReminder1dSent(subscriptionID int64) error {
	_, err := db.conn.Exec(`UPDATE subscriptions SET expiry_reminder_1d_sent = TRUE WHERE id = $1`, subscriptionID)
	return err
}

// --- Referral queries ---

func generateReferralCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:8], nil
}

func (db *DB) GetOrCreateReferralCode(userID int64) (string, error) {
	var code string
	err := db.conn.QueryRow("SELECT code FROM referral_codes WHERE user_id = $1", userID).Scan(&code)
	if err == nil {
		return code, nil
	}
	code, err = generateReferralCode()
	if err != nil {
		return "", err
	}
	for i := 0; i < 5; i++ {
		_, err = db.conn.Exec("INSERT INTO referral_codes (user_id, code) VALUES ($1, $2)", userID, code)
		if err == nil {
			return code, nil
		}
		code, _ = generateReferralCode()
	}
	return "", fmt.Errorf("failed to generate unique referral code")
}

func (db *DB) GetUserIDByReferralCode(code string) (int64, error) {
	var userID int64
	err := db.conn.QueryRow("SELECT user_id FROM referral_codes WHERE code = $1", code).Scan(&userID)
	return userID, err
}

func (db *DB) CreateReferral(referrerID, referredID int64) error {
	_, err := db.conn.Exec(
		"INSERT INTO referrals (referrer_id, referred_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT DO NOTHING",
		referrerID, referredID,
	)
	return err
}

func (db *DB) CountReferredBy(referrerID int64) (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM referrals WHERE referrer_id = $1", referrerID).Scan(&count)
	return count, err
}

func (db *DB) SumBonusDaysReceived(referrerID int64) (int, error) {
	var sum sql.NullInt64
	err := db.conn.QueryRow("SELECT COALESCE(SUM(bonus_days), 0) FROM referral_bonuses WHERE referrer_id = $1", referrerID).Scan(&sum)
	if err != nil || !sum.Valid {
		return 0, err
	}
	return int(sum.Int64), nil
}

func (db *DB) GetReferrerByReferred(referredID int64) (int64, error) {
	var referrerID int64
	err := db.conn.QueryRow("SELECT referrer_id FROM referrals WHERE referred_id = $1", referredID).Scan(&referrerID)
	return referrerID, err
}

func (db *DB) ReferralBonusExistsForPayment(paymentID int64) (bool, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM referral_bonuses WHERE payment_id = $1", paymentID).Scan(&count)
	return count > 0, err
}

func (db *DB) CreateReferralBonus(referrerID, referredUserID int64, paymentID int64, bonusDays int) error {
	_, err := db.conn.Exec(
		`INSERT INTO referral_bonuses (referrer_id, referred_user_id, payment_id, bonus_days, created_at)
		 VALUES ($1, $2, $3, $4, NOW())`,
		referrerID, referredUserID, paymentID, bonusDays,
	)
	return err
}

func (db *DB) ExtendSubscription(userID int64, days int) error {
	sub, err := db.GetActiveSubscription(userID)
	if err != nil || sub == nil {
		now := time.Now()
		expiresAt := now.AddDate(0, 0, days)
		_, err = db.conn.Exec(
			`INSERT INTO subscriptions (user_id, plan_id, payment_id, starts_at, expires_at) VALUES ($1, $2, $3, $4, $5)`,
			userID, "referral_bonus", 0, now, expiresAt,
		)
		return err
	}
	_, err = db.conn.Exec(
		`UPDATE subscriptions SET
			expires_at = expires_at + ($1 * INTERVAL '1 day'),
			expiry_reminder_7d_sent = FALSE,
			expiry_reminder_1d_sent = FALSE
		WHERE id = $2`,
		days, sub.ID,
	)
	return err
}
