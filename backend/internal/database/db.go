package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/models"
)

type DB struct {
	conn *sql.DB
	cfg  *config.Config
}

func New(cfg *config.Config) (*DB, error) {
	dir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", cfg.DBPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
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
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			max_proxies INTEGER NOT NULL DEFAULT 5,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS proxies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			port INTEGER NOT NULL UNIQUE,
			domain TEXT NOT NULL,
			secret TEXT NOT NULL,
			container_id TEXT NOT NULL DEFAULT '',
			container_name TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'stopped',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS payments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			plan_id TEXT NOT NULL,
			external_id TEXT NOT NULL UNIQUE,
			amount TEXT NOT NULL DEFAULT '0',
			status TEXT NOT NULL DEFAULT 'pending',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			plan_id TEXT NOT NULL,
			payment_id INTEGER NOT NULL DEFAULT 0,
			starts_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS referral_codes (
			user_id INTEGER PRIMARY KEY,
			code TEXT NOT NULL UNIQUE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS referrals (
			referrer_id INTEGER NOT NULL,
			referred_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (referrer_id, referred_id),
			FOREIGN KEY (referrer_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (referred_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS referral_bonuses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			referrer_id INTEGER NOT NULL,
			referred_user_id INTEGER NOT NULL,
			payment_id INTEGER NOT NULL,
			bonus_days INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (referrer_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (referred_user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
			return err
		}
	}

	// SOCKS5 columns migration (idempotent)
	for _, alter := range []string{
		"ALTER TABLE proxies ADD COLUMN socks5_port INTEGER DEFAULT 0",
		"ALTER TABLE proxies ADD COLUMN socks5_user TEXT DEFAULT ''",
		"ALTER TABLE proxies ADD COLUMN socks5_pass TEXT DEFAULT ''",
		"ALTER TABLE proxies ADD COLUMN socks5_container_id TEXT DEFAULT ''",
		"ALTER TABLE proxies ADD COLUMN socks5_container_name TEXT DEFAULT ''",
	} {
		if _, err := db.conn.Exec(alter); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}

	return db.ensureAdmin()
}

func (db *DB) ensureAdmin() error {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users WHERE role = ?", models.RoleAdmin).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if len(db.cfg.AdminPassword) < 6 {
		return fmt.Errorf("ADMIN_PASSWORD must be set and at least 6 characters for initial admin creation")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(db.cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	adminUsername := db.cfg.AdminUsername
	if adminUsername == "" {
		adminUsername = "admin"
	}

	_, err = db.conn.Exec(
		"INSERT INTO users (username, password_hash, role, max_proxies) VALUES (?, ?, ?, ?)",
		adminUsername, string(hash), models.RoleAdmin, 100,
	)
	return err
}

// --- User queries ---

func (db *DB) CreateUser(username, password string, referrerID *int64) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	res, err := db.conn.Exec(
		"INSERT INTO users (username, password_hash, role, max_proxies) VALUES (?, ?, ?, ?)",
		username, string(hash), models.RoleUser, db.cfg.DefaultMaxProxies,
	)
	if err != nil {
		return nil, err
	}

	id, _ := res.LastInsertId()
	user := &models.User{
		ID:         id,
		Username:   username,
		Role:       models.RoleUser,
		MaxProxies: db.cfg.DefaultMaxProxies,
		CreatedAt:  time.Now(),
	}

	if referrerID != nil && *referrerID > 0 && *referrerID != id {
		_ = db.CreateReferral(*referrerID, id)
	}

	return user, nil
}

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	u := &models.User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password_hash, role, max_proxies, created_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MaxProxies, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) GetUserByID(id int64) (*models.User, error) {
	u := &models.User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password_hash, role, max_proxies, created_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MaxProxies, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) ListUsers() ([]models.User, error) {
	rows, err := db.conn.Query("SELECT id, username, role, max_proxies, created_at FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.MaxProxies, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *DB) UpdateUser(id int64, role models.Role, maxProxies int) error {
	_, err := db.conn.Exec("UPDATE users SET role = ?, max_proxies = ? WHERE id = ?", role, maxProxies, id)
	return err
}

func (db *DB) DeleteUser(id int64) error {
	_, err := db.conn.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// --- Proxy queries ---

func (db *DB) CreateProxy(p *models.Proxy) error {
	res, err := db.conn.Exec(
		`INSERT INTO proxies (user_id, port, domain, secret, container_id, container_name, status,
			socks5_port, socks5_user, socks5_pass, socks5_container_id, socks5_container_name)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.UserID, p.Port, p.Domain, p.Secret, p.ContainerID, p.ContainerName, p.Status,
		p.Socks5Port, p.Socks5User, p.Socks5Pass, p.Socks5ContainerID, p.Socks5ContainerName,
	)
	if err != nil {
		return err
	}
	p.ID, _ = res.LastInsertId()
	p.CreatedAt = time.Now()
	return nil
}

func (db *DB) GetProxy(id int64) (*models.Proxy, error) {
	p := &models.Proxy{}
	err := db.conn.QueryRow(
		`SELECT id, user_id, port, domain, secret, container_id, container_name, status, created_at,
			COALESCE(socks5_port, 0), COALESCE(socks5_user, ''), COALESCE(socks5_pass, ''),
			COALESCE(socks5_container_id, ''), COALESCE(socks5_container_name, '')
		 FROM proxies WHERE id = ?`, id,
	).Scan(&p.ID, &p.UserID, &p.Port, &p.Domain, &p.Secret, &p.ContainerID, &p.ContainerName, &p.Status, &p.CreatedAt,
		&p.Socks5Port, &p.Socks5User, &p.Socks5Pass, &p.Socks5ContainerID, &p.Socks5ContainerName)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) ListProxiesByUser(userID int64) ([]models.Proxy, error) {
	rows, err := db.conn.Query(
		`SELECT id, user_id, port, domain, secret, container_id, container_name, status, created_at,
			COALESCE(socks5_port, 0), COALESCE(socks5_user, ''), COALESCE(socks5_pass, ''),
			COALESCE(socks5_container_id, ''), COALESCE(socks5_container_name, '')
		 FROM proxies WHERE user_id = ? ORDER BY id`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []models.Proxy
	for rows.Next() {
		var p models.Proxy
		if err := rows.Scan(&p.ID, &p.UserID, &p.Port, &p.Domain, &p.Secret, &p.ContainerID, &p.ContainerName, &p.Status, &p.CreatedAt,
			&p.Socks5Port, &p.Socks5User, &p.Socks5Pass, &p.Socks5ContainerID, &p.Socks5ContainerName); err != nil {
			return nil, err
		}
		proxies = append(proxies, p)
	}
	return proxies, nil
}

func (db *DB) ListAllProxies() ([]models.Proxy, error) {
	rows, err := db.conn.Query(
		`SELECT id, user_id, port, domain, secret, container_id, container_name, status, created_at,
			COALESCE(socks5_port, 0), COALESCE(socks5_user, ''), COALESCE(socks5_pass, ''),
			COALESCE(socks5_container_id, ''), COALESCE(socks5_container_name, '')
		 FROM proxies ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []models.Proxy
	for rows.Next() {
		var p models.Proxy
		if err := rows.Scan(&p.ID, &p.UserID, &p.Port, &p.Domain, &p.Secret, &p.ContainerID, &p.ContainerName, &p.Status, &p.CreatedAt,
			&p.Socks5Port, &p.Socks5User, &p.Socks5Pass, &p.Socks5ContainerID, &p.Socks5ContainerName); err != nil {
			return nil, err
		}
		proxies = append(proxies, p)
	}
	return proxies, nil
}

func (db *DB) UpdateProxyStatus(id int64, status models.ProxyStatus, containerID string) error {
	_, err := db.conn.Exec("UPDATE proxies SET status = ?, container_id = ? WHERE id = ?", status, containerID, id)
	return err
}

func (db *DB) DeleteProxy(id int64) error {
	_, err := db.conn.Exec("DELETE FROM proxies WHERE id = ?", id)
	return err
}

func (db *DB) CountProxiesByUser(userID int64) (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM proxies WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

func (db *DB) IsPortUsed(port int) (bool, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM proxies WHERE port = ? OR socks5_port = ?", port, port).Scan(&count)
	return count > 0, err
}

func (db *DB) GetUsedPorts() (map[int]bool, error) {
	rows, err := db.conn.Query("SELECT port FROM proxies UNION SELECT socks5_port FROM proxies WHERE socks5_port > 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ports := make(map[int]bool)
	for rows.Next() {
		var port int
		if err := rows.Scan(&port); err != nil {
			return nil, err
		}
		ports[port] = true
	}
	return ports, nil
}

// --- Payment queries ---

func (db *DB) CreatePayment(p *models.Payment) error {
	res, err := db.conn.Exec(
		`INSERT INTO payments (user_id, plan_id, external_id, amount, status) VALUES (?, ?, ?, ?, ?)`,
		p.UserID, p.PlanID, p.ExternalID, p.Amount, p.Status,
	)
	if err != nil {
		return err
	}
	p.ID, _ = res.LastInsertId()
	p.CreatedAt = time.Now()
	return nil
}

func (db *DB) GetPaymentByExternalID(externalID string) (*models.Payment, error) {
	p := &models.Payment{}
	err := db.conn.QueryRow(
		`SELECT id, user_id, plan_id, external_id, amount, status, created_at FROM payments WHERE external_id = ?`,
		externalID,
	).Scan(&p.ID, &p.UserID, &p.PlanID, &p.ExternalID, &p.Amount, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) UpdatePaymentStatus(externalID, status string) error {
	_, err := db.conn.Exec("UPDATE payments SET status = ? WHERE external_id = ?", status, externalID)
	return err
}

// --- Subscription queries ---

func (db *DB) CreateSubscription(s *models.Subscription) error {
	res, err := db.conn.Exec(
		`INSERT INTO subscriptions (user_id, plan_id, payment_id, starts_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		s.UserID, s.PlanID, s.PaymentID, s.StartsAt, s.ExpiresAt,
	)
	if err != nil {
		return err
	}
	s.ID, _ = res.LastInsertId()
	s.CreatedAt = time.Now()
	return nil
}

func (db *DB) GetActiveSubscription(userID int64) (*models.Subscription, error) {
	s := &models.Subscription{}
	err := db.conn.QueryRow(
		`SELECT id, user_id, plan_id, payment_id, starts_at, expires_at, created_at
		 FROM subscriptions WHERE user_id = ? AND expires_at > datetime('now') ORDER BY expires_at DESC LIMIT 1`,
		userID,
	).Scan(&s.ID, &s.UserID, &s.PlanID, &s.PaymentID, &s.StartsAt, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
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
	err := db.conn.QueryRow("SELECT code FROM referral_codes WHERE user_id = ?", userID).Scan(&code)
	if err == nil {
		return code, nil
	}
	code, err = generateReferralCode()
	if err != nil {
		return "", err
	}
	for i := 0; i < 5; i++ {
		_, err = db.conn.Exec("INSERT INTO referral_codes (user_id, code) VALUES (?, ?)", userID, code)
		if err == nil {
			return code, nil
		}
		code, _ = generateReferralCode()
	}
	return "", fmt.Errorf("failed to generate unique referral code")
}

func (db *DB) GetUserIDByReferralCode(code string) (int64, error) {
	var userID int64
	err := db.conn.QueryRow("SELECT user_id FROM referral_codes WHERE code = ?", code).Scan(&userID)
	return userID, err
}

func (db *DB) CreateReferral(referrerID, referredID int64) error {
	_, err := db.conn.Exec(
		"INSERT OR IGNORE INTO referrals (referrer_id, referred_id, created_at) VALUES (?, ?, datetime('now'))",
		referrerID, referredID,
	)
	return err
}

func (db *DB) CountReferredBy(referrerID int64) (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM referrals WHERE referrer_id = ?", referrerID).Scan(&count)
	return count, err
}

func (db *DB) SumBonusDaysReceived(referrerID int64) (int, error) {
	var sum sql.NullInt64
	err := db.conn.QueryRow("SELECT COALESCE(SUM(bonus_days), 0) FROM referral_bonuses WHERE referrer_id = ?", referrerID).Scan(&sum)
	if err != nil || !sum.Valid {
		return 0, err
	}
	return int(sum.Int64), nil
}

func (db *DB) GetReferrerByReferred(referredID int64) (int64, error) {
	var referrerID int64
	err := db.conn.QueryRow("SELECT referrer_id FROM referrals WHERE referred_id = ?", referredID).Scan(&referrerID)
	return referrerID, err
}

func (db *DB) ReferralBonusExistsForPayment(paymentID int64) (bool, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM referral_bonuses WHERE payment_id = ?", paymentID).Scan(&count)
	return count > 0, err
}

func (db *DB) CreateReferralBonus(referrerID, referredUserID int64, paymentID int64, bonusDays int) error {
	_, err := db.conn.Exec(
		`INSERT INTO referral_bonuses (referrer_id, referred_user_id, payment_id, bonus_days, created_at)
		 VALUES (?, ?, ?, ?, datetime('now'))`,
		referrerID, referredUserID, paymentID, bonusDays,
	)
	return err
}

func (db *DB) ExtendSubscription(userID int64, days int) error {
	sub, err := db.GetActiveSubscription(userID)
	if err != nil || sub == nil {
		// No active subscription - create a new one starting now
		now := time.Now()
		expiresAt := now.AddDate(0, 0, days)
		_, err = db.conn.Exec(
			`INSERT INTO subscriptions (user_id, plan_id, payment_id, starts_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
			userID, "referral_bonus", 0, now, expiresAt,
		)
		return err
	}
	modifier := fmt.Sprintf("+%d days", days)
	_, err = db.conn.Exec(
		"UPDATE subscriptions SET expires_at = datetime(expires_at, ?) WHERE id = ?",
		modifier, sub.ID,
	)
	return err
}
