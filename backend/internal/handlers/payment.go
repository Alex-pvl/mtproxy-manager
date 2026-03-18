package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/models"
	"mtproxy-manager/internal/xui"
)

const cryptoPayAPI = "https://pay.crypt.bot/api"

type PaymentHandler struct {
	db        *database.DB
	cfg       *config.Config
	xuiClient *xui.Client // nil if x-ui integration is disabled
}

func NewPaymentHandler(db *database.DB, cfg *config.Config, xuiClient *xui.Client) *PaymentHandler {
	return &PaymentHandler{db: db, cfg: cfg, xuiClient: xuiClient}
}

func (h *PaymentHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, models.Plans)
}

type createPaymentRequest struct {
	PlanID string `json:"plan_id"`
}

func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	plan := models.GetPlan(req.PlanID)
	if plan == nil {
		writeError(w, http.StatusBadRequest, "invalid plan")
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"user_id": strconv.FormatInt(claims.UserID, 10),
		"plan_id": plan.ID,
	})

	returnURL := h.cfg.BaseURL + "/pricing"

	invoiceReq := map[string]interface{}{
		"currency_type": "fiat",
		"fiat":          "RUB",
		"amount":        plan.Price,
		"description":   fmt.Sprintf("Подписка MTProxy — %s", plan.Name),
		"payload":       string(payload),
		"paid_btn_name": "callback",
		"paid_btn_url":  returnURL,
	}

	body, _ := json.Marshal(invoiceReq)

	httpReq, _ := http.NewRequest("POST", cryptoPayAPI+"/createInvoice", bytes.NewReader(body))
	httpReq.Header.Set("Crypto-Pay-API-Token", h.cfg.CryptoBotToken)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("CryptoPay request error: %v", err)
		writeError(w, http.StatusInternalServerError, "payment service unavailable")
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var cryptoResp struct {
		OK     bool `json:"ok"`
		Result struct {
			InvoiceID     int64  `json:"invoice_id"`
			BotInvoiceURL string `json:"bot_invoice_url"`
			Status        string `json:"status"`
		} `json:"result"`
		Error struct {
			Code int    `json:"code"`
			Name string `json:"name"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &cryptoResp); err != nil || !cryptoResp.OK {
		log.Printf("CryptoPay error: ok=%v, err=%v, body=%s", cryptoResp.OK, err, string(respBody))
		writeError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	payment := &models.Payment{
		UserID:     claims.UserID,
		PlanID:     plan.ID,
		ExternalID: strconv.FormatInt(cryptoResp.Result.InvoiceID, 10),
		Amount:     plan.Price,
		Status:     "pending",
	}
	if err := h.db.CreatePayment(payment); err != nil {
		log.Printf("Failed to save payment: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"payment_url": cryptoResp.Result.BotInvoiceURL,
	})
}

func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("crypto-pay-api-signature")
	if !h.verifySignature(body, signature) {
		log.Printf("Webhook signature verification failed")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var update struct {
		UpdateType string `json:"update_type"`
		Payload    struct {
			InvoiceID int64  `json:"invoice_id"`
			Status    string `json:"status"`
			Payload   string `json:"payload"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("Webhook parse error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if update.UpdateType != "invoice_paid" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var meta struct {
		UserID string `json:"user_id"`
		PlanID string `json:"plan_id"`
	}
	if err := json.Unmarshal([]byte(update.Payload.Payload), &meta); err != nil {
		log.Printf("Webhook payload parse error: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	userID, _ := strconv.ParseInt(meta.UserID, 10, 64)
	planID := meta.PlanID

	plan := models.GetPlan(planID)
	if plan == nil || userID == 0 {
		log.Printf("Invalid plan_id=%s or user_id=%d in webhook", planID, userID)
		w.WriteHeader(http.StatusOK)
		return
	}

	externalID := strconv.FormatInt(update.Payload.InvoiceID, 10)
	h.db.UpdatePaymentStatus(externalID, "paid")

	existing, _ := h.db.GetActiveSubscription(userID)

	startsAt := time.Now()
	if existing != nil && existing.ExpiresAt.After(startsAt) {
		startsAt = existing.ExpiresAt
	}
	expiresAt := startsAt.AddDate(0, 0, plan.DurationDays)

	var paymentID int64
	payment, _ := h.db.GetPaymentByExternalID(externalID)
	if payment != nil {
		paymentID = payment.ID
	}

	sub := &models.Subscription{
		UserID:    userID,
		PlanID:    planID,
		PaymentID: paymentID,
		StartsAt:  startsAt,
		ExpiresAt: expiresAt,
	}
	if err := h.db.CreateSubscription(sub); err != nil {
		log.Printf("Failed to create subscription: %v", err)
	} else {
		log.Printf("Subscription created for user %d, plan %s, expires %s", userID, planID, expiresAt.Format(time.RFC3339))
		// Update user's max_proxies to match the plan
		if user, err := h.db.GetUserByID(userID); err == nil {
			_ = h.db.UpdateUser(userID, user.Role, plan.MaxProxies)
		}
		// Sync new expiry to all existing VLESS clients for this user
		h.syncVlessExpiry(userID, expiresAt)

		// Referral bonus: 15% of subscription days to referrer (once per payment)
		if referrerID, err := h.db.GetReferrerByReferred(userID); err == nil && referrerID > 0 {
			exists, _ := h.db.ReferralBonusExistsForPayment(paymentID)
			if !exists {
				bonusDays := int(float64(plan.DurationDays) * 0.15)
				if bonusDays > 0 {
					if err := h.db.CreateReferralBonus(referrerID, userID, paymentID, bonusDays); err != nil {
						log.Printf("Failed to create referral bonus: %v", err)
					} else if err := h.db.ExtendSubscription(referrerID, bonusDays); err != nil {
						log.Printf("Failed to extend referrer subscription: %v", err)
					} else {
						log.Printf("Referral bonus: %d days added to user %d for referred user %d", bonusDays, referrerID, userID)
						// Sync new expiry for referrer's VLESS clients too
						if referrerSub, err := h.db.GetActiveSubscription(referrerID); err == nil && referrerSub != nil {
							h.syncVlessExpiry(referrerID, referrerSub.ExpiresAt)
						}
					}
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PaymentHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sub, err := h.db.GetActiveSubscription(claims.UserID)
	if err != nil || sub == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"active": false,
		})
		return
	}

	plan := models.GetPlan(sub.PlanID)
	planName := sub.PlanID
	if plan != nil {
		planName = plan.Name
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"active":     true,
		"plan_name":  planName,
		"expires_at": sub.ExpiresAt,
	})
}

// syncVlessExpiry updates the expiry time of all VLESS clients belonging to userID
// in the x-ui panel to match their subscription expiry. Called after any subscription change.
func (h *PaymentHandler) syncVlessExpiry(userID int64, expiresAt time.Time) {
	if h.xuiClient == nil {
		return
	}
	proxies, err := h.db.ListProxiesWithVlessByUser(userID)
	if err != nil {
		log.Printf("syncVlessExpiry: list proxies for user %d: %v", userID, err)
		return
	}
	for _, p := range proxies {
		email := vlessEmail(p.Port, p.UserID)
		if err := h.xuiClient.UpdateClientExpiry(p.VlessUUID, email, expiresAt); err != nil {
			log.Printf("syncVlessExpiry: update uuid=%s user=%d: %v", p.VlessUUID, userID, err)
		}
	}
	if len(proxies) > 0 {
		log.Printf("syncVlessExpiry: updated %d VLESS client(s) for user %d, expires %s",
			len(proxies), userID, expiresAt.Format(time.RFC3339))
	}
}

func (h *PaymentHandler) verifySignature(body []byte, signature string) bool {
	if signature == "" || h.cfg.CryptoBotToken == "" {
		return false
	}
	secret := sha256.Sum256([]byte(h.cfg.CryptoBotToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

