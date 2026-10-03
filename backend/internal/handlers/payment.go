package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/models"
)

// PaymentHandler sells subscription plans through CryptoBot, SBP (RollyPay),
// TON and Telegram Stars. Every provider ends in fulfill(), which activates the
// plan exactly once per payment.
type PaymentHandler struct {
	db  *database.DB
	cfg *config.Config
	vpn *VPN
	bot *Bot
}

func NewPaymentHandler(db *database.DB, cfg *config.Config, vpn *VPN, bot *Bot) *PaymentHandler {
	return &PaymentHandler{db: db, cfg: cfg, vpn: vpn, bot: bot}
}

func (h *PaymentHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, models.Plans)
}

func (h *PaymentHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, loadSubscriptionInfo(h.db, getClaims(r).UserID))
}

type paymentRequest struct {
	UserID int64
	Plan   *models.Plan
	Source string // "tg" when paying from the Mini App, else "web"
}

// readPaymentRequest parses {plan_id, source}; on failure it has written the response.
func readPaymentRequest(w http.ResponseWriter, r *http.Request) (*paymentRequest, bool) {
	var body struct {
		PlanID string `json:"plan_id"`
		Source string `json:"source"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return nil, false
	}
	plan := models.GetPlan(body.PlanID)
	if plan == nil {
		writeError(w, http.StatusBadRequest, "invalid plan")
		return nil, false
	}
	source := "web"
	switch strings.ToLower(strings.TrimSpace(body.Source)) {
	case "tg", "telegram", "miniapp":
		source = "tg"
	}
	return &paymentRequest{UserID: getClaims(r).UserID, Plan: plan, Source: source}, true
}

// savePayment records a pending payment; without it we could not credit the user.
func (h *PaymentHandler) savePayment(w http.ResponseWriter, req *paymentRequest, externalID, amount string) bool {
	err := h.db.CreatePayment(&models.Payment{UserID: req.UserID, PlanID: req.Plan.ID, ExternalID: externalID, Amount: amount})
	if err != nil {
		log.Printf("save payment %s: %v", externalID, err)
		writeError(w, http.StatusInternalServerError, "failed to create payment")
		return false
	}
	paymentsCreated.WithLabelValues(providerOf(externalID)).Inc()
	return true
}

// returnURL is where the provider sends the user after paying.
func (h *PaymentHandler) returnURL(source string) string {
	if source == "tg" {
		if bot := strings.TrimPrefix(strings.TrimSpace(h.cfg.TelegramBotUsername), "@"); bot != "" {
			return "https://t.me/" + bot + "?startapp=payment_success"
		}
		if u := h.cfg.TelegramPayURL; u != "" {
			sep := "?"
			if strings.Contains(u, "?") {
				sep = "&"
			}
			return u + sep + "startapp=payment_success"
		}
	}
	return strings.TrimRight(h.cfg.BaseURL, "/") + "/pricing?payment=1"
}

// fulfill activates the plan of a confirmed payment. Safe to call repeatedly.
func (h *PaymentHandler) fulfill(externalID string) error {
	activated, err := h.activate(externalID)
	switch {
	case err != nil:
		paymentFulfillErrors.WithLabelValues(providerOf(externalID)).Inc()
	case activated:
		paymentsFulfilled.WithLabelValues(providerOf(externalID)).Inc()
	}
	return err
}

// activate reports whether this call was the one that activated the payment.
func (h *PaymentHandler) activate(externalID string) (bool, error) {
	payment, err := h.db.MarkPaymentPaid(externalID)
	if err != nil {
		return false, fmt.Errorf("mark paid %s: %w", externalID, err)
	}
	if payment == nil {
		return false, nil // already fulfilled or canceled
	}
	plan := models.GetPlan(payment.PlanID)
	if plan == nil {
		return false, fmt.Errorf("payment %s has unknown plan %q", externalID, payment.PlanID)
	}

	startsAt := time.Now()
	if cur, _ := h.db.GetActiveSubscription(payment.UserID); cur != nil {
		startsAt = cur.ExpiresAt
	}
	sub := &models.Subscription{
		UserID:    payment.UserID,
		PlanID:    plan.ID,
		PaymentID: payment.ID,
		StartsAt:  startsAt,
		ExpiresAt: startsAt.AddDate(0, 0, plan.DurationDays),
	}
	if err := h.db.CreateSubscription(sub); err != nil {
		return false, fmt.Errorf("create subscription for payment %s: %w", externalID, err)
	}
	log.Printf("subscription activated: user=%d plan=%s expires=%s", payment.UserID, plan.ID, sub.ExpiresAt.Format(time.RFC3339))

	if user, err := h.db.GetUserByID(payment.UserID); err == nil {
		_ = h.db.UpdateUser(user.ID, user.Role, plan.MaxProxies)
		go h.bot.NotifySubscriptionPaid(user, plan)
	}
	h.vpn.SyncExpiry(payment.UserID, sub.ExpiresAt)
	h.creditReferrer(payment, plan)
	return true, nil
}

const referralBonusShare = 0.15

// creditReferrer gives whoever invited the payer 15% of the plan's days.
func (h *PaymentHandler) creditReferrer(payment *models.Payment, plan *models.Plan) {
	referrerID, err := h.db.GetReferrerByReferred(payment.UserID)
	if err != nil || referrerID == 0 {
		return
	}
	bonusDays := int(float64(plan.DurationDays) * referralBonusShare)
	if bonusDays == 0 {
		return
	}
	if exists, _ := h.db.ReferralBonusExistsForPayment(payment.ID); exists {
		return
	}
	if err := h.db.CreateReferralBonus(referrerID, payment.UserID, payment.ID, bonusDays); err != nil {
		log.Printf("referral bonus payment=%d: %v", payment.ID, err)
		return
	}
	if err := h.db.ExtendSubscription(referrerID, bonusDays); err != nil {
		log.Printf("referral extend user=%d: %v", referrerID, err)
		return
	}
	if sub, _ := h.db.GetActiveSubscription(referrerID); sub != nil {
		h.vpn.SyncExpiry(referrerID, sub.ExpiresAt)
	}
}

// CheckPendingPayments asks providers about the user's pending payments, for
// when a webhook was missed. Stars are only confirmed by the bot webhook.
func (h *PaymentHandler) CheckPendingPayments(w http.ResponseWriter, r *http.Request) {
	pending, err := h.db.GetPendingPaymentsByUser(getClaims(r).UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load payments")
		return
	}
	updated := false
	for _, p := range pending {
		var paid bool
		var err error
		switch {
		case strings.HasPrefix(p.ExternalID, tonPrefix):
			paid, err = h.tonPaid(p)
		case strings.HasPrefix(p.ExternalID, sbpPrefix):
			paid, err = h.sbpPaid(p.ExternalID)
		case strings.HasPrefix(p.ExternalID, starsPrefix):
			continue
		default:
			paid, err = h.cryptoBotPaid(p.ExternalID)
		}
		if err != nil {
			log.Printf("check payment %s: %v", p.ExternalID, err)
			continue
		}
		if paid {
			if err := h.fulfill(p.ExternalID); err != nil {
				log.Printf("fulfill %s: %v", p.ExternalID, err)
				continue
			}
			updated = true
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"updated": updated})
}
