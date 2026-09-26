package handlers

import (
	"database/sql"
	"os"
	"sync"
	"testing"

	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/models"
)

// Runs against a throwaway Postgres: TEST_DATABASE_URL=postgres://... go test ./...
// It wipes the database's public schema.
func TestLegacyMigrationAndSingleFulfilment(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	raw, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := raw.Exec(q, args...); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}

	cfg := &config.Config{DatabaseURL: dsn, DefaultMaxProxies: 5}

	// Fresh install.
	mustExec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`)
	fresh, err := database.New(cfg)
	if err != nil {
		t.Fatalf("migrate empty db: %v", err)
	}
	fresh.Close()

	// Schema as production has it from the MTProxy/SOCKS5 era.
	mustExec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`)
	mustExec(`CREATE TABLE users (id BIGSERIAL PRIMARY KEY, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user', max_proxies INTEGER NOT NULL DEFAULT 5, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		telegram_id BIGINT DEFAULT 0)`)
	mustExec(`CREATE TABLE proxies (id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		port INTEGER NOT NULL UNIQUE, domain TEXT NOT NULL, secret TEXT NOT NULL, container_id TEXT NOT NULL DEFAULT '',
		container_name TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'stopped', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		socks5_port INTEGER DEFAULT 0, socks5_user TEXT DEFAULT '', socks5_pass TEXT DEFAULT '',
		socks5_container_id TEXT DEFAULT '', socks5_container_name TEXT DEFAULT '', vless_uuid TEXT DEFAULT '')`)
	mustExec(`CREATE TABLE payments (id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		plan_id TEXT NOT NULL, external_id TEXT NOT NULL UNIQUE, amount TEXT NOT NULL DEFAULT '0', status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), product_type TEXT NOT NULL DEFAULT 'vpn', metadata JSONB NOT NULL DEFAULT '{}'::jsonb)`)
	mustExec(`INSERT INTO users (username, password_hash) VALUES ('alice', '')`)
	mustExec(`INSERT INTO proxies (user_id, port, domain, secret, vless_uuid) VALUES (1, 8001, 'd', 's', ''), (1, 8007, '', '', 'u-old')`)

	for i := 0; i < 2; i++ { // migration must be re-runnable
		db, err := database.New(cfg)
		if err != nil {
			t.Fatalf("migrate run %d: %v", i+1, err)
		}
		db.Close()
	}
	db, err := database.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	proxies, err := db.ListProxiesByUser(1)
	if err != nil || len(proxies) != 1 || proxies[0].VlessEmail != "proxy-8007-user-1" || proxies[0].VlessUUID != "u-old" {
		t.Fatalf("legacy proxies after migration: %+v %v", proxies, err)
	}
	if err := db.CreateProxy(&models.Proxy{UserID: 1, VlessUUID: "u-new", VlessEmail: "staytg.org-alice-1"}); err != nil {
		t.Fatalf("insert into migrated table: %v", err)
	}

	// Payment on the legacy payments table, fulfilled by racing webhooks.
	if err := db.CreatePayment(&models.Payment{UserID: 1, PlanID: "month_3", ExternalID: "42", Amount: "540.00"}); err != nil {
		t.Fatal(err)
	}
	h := NewPaymentHandler(db, cfg, NewVPN(db, nil), NewBot(cfg, db))
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.fulfill("42"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	var subs, maxProxies int
	raw.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE user_id = 1`).Scan(&subs)
	raw.QueryRow(`SELECT max_proxies FROM users WHERE id = 1`).Scan(&maxProxies)
	if subs != 1 || maxProxies != 3 {
		t.Fatalf("want 1 subscription and max_proxies=3, got %d and %d", subs, maxProxies)
	}
}
