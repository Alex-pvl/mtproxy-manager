package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// RollyPay payment ids are stored with this prefix.
const sbpPrefix = "sbp_"

// CreateSBPPayment creates an SBP payment via RollyPay and returns its pay page.
func (h *PaymentHandler) CreateSBPPayment(w http.ResponseWriter, r *http.Request) {
	if h.cfg.RollyPayAPIKey == "" {
		writeError(w, http.StatusServiceUnavailable, "sbp payments not configured")
		return
	}
	req, ok := readPaymentRequest(w, r)
	if !ok {
		return
	}
	back := h.returnURL(req.Source)
	body := map[string]any{
		"amount":               req.Plan.Price, // already "200.00"
		"payment_currency":     "RUB",
		"payment_method":       "sbp",
		"order_id":             fmt.Sprintf("sbp_%d_%d", req.UserID, time.Now().UnixMilli()),
		"customer_id":          fmt.Sprint(req.UserID),
		"description":          "Stay VPN: " + req.Plan.Name,
		"success_redirect_url": back,
		"fail_redirect_url":    back,
		"test":                 h.cfg.RollyPayTest,
	}
	var data struct {
		PaymentID string `json:"payment_id"`
		PayURL    string `json:"pay_url"`
	}
	if err := h.rollyPay(http.MethodPost, "/api/v1/payments", body, &data); err != nil || data.PaymentID == "" || data.PayURL == "" {
		log.Printf("rollypay create: id=%q err=%v", data.PaymentID, err)
		writeError(w, http.StatusBadGateway, "payment service unavailable")
		return
	}
	if !h.savePayment(w, req, sbpPrefix+data.PaymentID, req.Plan.Price) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"payment_url": data.PayURL})
}

// RollyPayWebhook handles signed status callbacks. The callback URL is set in
// the RollyPay terminal settings: <BASE_URL>/api/payments/sbp/webhook.
func (h *PaymentHandler) RollyPayWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !rollyPaySigned(body, r.Header.Get("X-Timestamp"), r.Header.Get("X-Signature"), h.cfg.RollyPaySigningSecret) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	var cb struct {
		PaymentID string `json:"payment_id"`
		Status    string `json:"status"`
		Test      bool   `json:"test"`
	}
	if err := json.Unmarshal(body, &cb); err != nil || cb.PaymentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if cb.Test && !h.cfg.RollyPayTest {
		w.WriteHeader(http.StatusOK) // sandbox event on a live setup: never credit it
		return
	}
	externalID := sbpPrefix + cb.PaymentID
	if _, err := h.db.GetPaymentByExternalID(externalID); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := h.applySBPStatus(externalID, cb.Status); err != nil {
		log.Printf("rollypay webhook %s %s: %v", externalID, cb.Status, err)
		w.WriteHeader(http.StatusInternalServerError) // RollyPay retries
		return
	}
	w.WriteHeader(http.StatusOK)
}

// applySBPStatus moves our payment to match RollyPay's status.
// ponytail: chargeback/refunded are only logged; revoke the subscription if they start happening.
func (h *PaymentHandler) applySBPStatus(externalID, status string) error {
	switch status {
	case "paid":
		// A payment can be paid after it expired, so reopen it first.
		if err := h.db.ReopenPayment(externalID); err != nil {
			return err
		}
		return h.fulfill(externalID)
	case "canceled", "expired":
		return h.db.CancelPayment(externalID)
	case "chargeback", "refunded":
		log.Printf("rollypay: payment %s is %s, subscription NOT revoked", externalID, status)
	}
	return nil
}

// sbpPaid polls RollyPay for the payment status, for when a webhook was missed.
func (h *PaymentHandler) sbpPaid(externalID string) (bool, error) {
	id := strings.TrimPrefix(externalID, sbpPrefix)
	if h.cfg.RollyPayAPIKey == "" || !strings.HasPrefix(id, "pay_") {
		return false, nil // legacy DigitalPay payment, nothing to ask
	}
	var data struct {
		Status string `json:"status"`
	}
	if err := h.rollyPay(http.MethodGet, "/api/v1/payments/"+id, nil, &data); err != nil {
		return false, err
	}
	if data.Status == "paid" {
		return true, nil
	}
	return false, h.applySBPStatus(externalID, data.Status)
}

// rollyPay calls the RollyPay API with the terminal's API key.
func (h *PaymentHandler) rollyPay(method, path string, in, out any) error {
	var reqBody io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(h.cfg.RollyPayBaseURL, "/")+path, reqBody)
	if err != nil {
		return err
	}
	nonce := make([]byte, 16)
	_, _ = rand.Read(nonce)
	req.Header.Set("X-API-Key", h.cfg.RollyPayAPIKey)
	req.Header.Set("X-Nonce", hex.EncodeToString(nonce))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return fmt.Errorf("rollypay %s %s: HTTP %d: %s", method, path, resp.StatusCode, e.Error)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// rollyPaySigned checks X-Signature = hex(HMAC-SHA256(secret, timestamp + "." + body)).
func rollyPaySigned(body []byte, timestamp, signature, secret string) bool {
	if secret == "" || timestamp == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "."))
	mac.Write(body)
	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(strings.ToLower(signature)))
}
