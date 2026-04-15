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
	"strings"
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
	Source string `json:"source,omitempty"` // "tg" | "web"
}

func normalizePaymentSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "tg", "telegram", "miniapp":
		return "tg"
	default:
		return "web"
	}
}

func (h *PaymentHandler) paymentReturnURL(source string) string {
	if normalizePaymentSource(source) == "tg" {
		botUsername := strings.TrimPrefix(strings.TrimSpace(h.cfg.TelegramBotUsername), "@")
		if botUsername != "" {
			return fmt.Sprintf("https://t.me/%s?startapp=payment_success", botUsername)
		}
		if h.cfg.TelegramPayURL != "" {
			if strings.Contains(h.cfg.TelegramPayURL, "?") {
				return h.cfg.TelegramPayURL + "&startapp=payment_success"
			}
			return h.cfg.TelegramPayURL + "?startapp=payment_success"
		}
	}
	return strings.TrimRight(h.cfg.BaseURL, "/") + "/pricing?payment=1"
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
	source := normalizePaymentSource(req.Source)

	payload, _ := json.Marshal(map[string]string{
		"user_id": strconv.FormatInt(claims.UserID, 10),
		"plan_id": plan.ID,
		"source":  source,
	})

	returnURL := h.paymentReturnURL(source)

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
	if err := h.activateSubscription(userID, plan, externalID); err != nil {
		log.Printf("Failed to create subscription: %v", err)
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
