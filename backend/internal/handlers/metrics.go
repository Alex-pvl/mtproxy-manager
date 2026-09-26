package handlers

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	paymentsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "stay_payments_created_total",
		Help: "Pending payments created, by provider.",
	}, []string{"provider"})
	paymentsFulfilled = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "stay_payments_fulfilled_total",
		Help: "Payments confirmed and turned into a subscription, by provider.",
	}, []string{"provider"})
	paymentFulfillErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "stay_payment_fulfill_errors_total",
		Help: "Confirmed payments that failed to activate a subscription (needs manual fix).",
	}, []string{"provider"})
	telegramErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "stay_telegram_api_errors_total",
		Help: "Failed Telegram Bot API calls, by method.",
	}, []string{"method"})
)

// providerOf maps an external payment id to its provider by prefix.
func providerOf(externalID string) string {
	switch {
	case strings.HasPrefix(externalID, sbpPrefix):
		return "sbp"
	case strings.HasPrefix(externalID, tonPrefix):
		return "ton"
	case strings.HasPrefix(externalID, starsPrefix):
		return "stars"
	default:
		return "cryptobot"
	}
}
