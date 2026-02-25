package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
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

func (db *DB) CreateUser(username, password string) (*models.User, error) {
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
	return &models.User{
		ID:         id,
		Username:   username,
		Role:       models.RoleUser,
		MaxProxies: db.cfg.DefaultMaxProxies,
		CreatedAt:  time.Now(),
	}, nil
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
		`INSERT INTO proxies (user_id, port, domain, secret, container_id, container_name, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.UserID, p.Port, p.Domain, p.Secret, p.ContainerID, p.ContainerName, p.Status,
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
		`SELECT id, user_id, port, domain, secret, container_id, container_name, status, created_at
		 FROM proxies WHERE id = ?`, id,
	).Scan(&p.ID, &p.UserID, &p.Port, &p.Domain, &p.Secret, &p.ContainerID, &p.ContainerName, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *DB) ListProxiesByUser(userID int64) ([]models.Proxy, error) {
	rows, err := db.conn.Query(
		`SELECT id, user_id, port, domain, secret, container_id, container_name, status, created_at
		 FROM proxies WHERE user_id = ? ORDER BY id`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []models.Proxy
	for rows.Next() {
		var p models.Proxy
		if err := rows.Scan(&p.ID, &p.UserID, &p.Port, &p.Domain, &p.Secret, &p.ContainerID, &p.ContainerName, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		proxies = append(proxies, p)
	}
	return proxies, nil
}

func (db *DB) ListAllProxies() ([]models.Proxy, error) {
	rows, err := db.conn.Query(
		`SELECT id, user_id, port, domain, secret, container_id, container_name, status, created_at
		 FROM proxies ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []models.Proxy
	for rows.Next() {
		var p models.Proxy
		if err := rows.Scan(&p.ID, &p.UserID, &p.Port, &p.Domain, &p.Secret, &p.ContainerID, &p.ContainerName, &p.Status, &p.CreatedAt); err != nil {
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
	err := db.conn.QueryRow("SELECT COUNT(*) FROM proxies WHERE port = ?", port).Scan(&count)
	return count > 0, err
}

func (db *DB) GetUsedPorts() (map[int]bool, error) {
	rows, err := db.conn.Query("SELECT port FROM proxies")
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
