package database

import (
	"log"
	"math"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RegisterMetrics exposes business gauges computed by a COUNT query on each scrape.
func (db *DB) RegisterMetrics() {
	for name, q := range map[string]struct{ help, sql string }{
		"stay_users":                {"Registered users.", "SELECT COUNT(*) FROM users"},
		"stay_active_subscriptions": {"Users with a subscription that has not expired.", "SELECT COUNT(DISTINCT user_id) FROM subscriptions WHERE expires_at > NOW()"},
		"stay_connections":          {"Connections (3x-ui clients) across all users.", "SELECT COUNT(*) FROM proxies"},
		"stay_payments_pending_24h": {"Payments created in the last 24h that are still pending.", "SELECT COUNT(*) FROM payments WHERE status = 'pending' AND created_at > NOW() - INTERVAL '24 hours'"},
	} {
		promauto.NewGaugeFunc(prometheus.GaugeOpts{Name: name, Help: q.help}, func() float64 {
			var n float64
			if err := db.conn.QueryRow(q.sql).Scan(&n); err != nil {
				log.Printf("metric %s: %v", name, err)
				return math.NaN()
			}
			return n
		})
	}
}
