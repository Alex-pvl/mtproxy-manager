package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DigitalPay payment ids are stored with this prefix.
const sbpPrefix = "sbp_"

// CreateSBPPayment creates an SBP (NSPK QR) payment via DigitalPay.
func (h *PaymentHandler) CreateSBPPayment(w http.ResponseWriter, r *http.Request) {
	if h.cfg.DigitalPayAPIKey == "" {
		writeError(w, http.StatusServiceUnavailable, "sbp payments not configured")
		return
	}
	req, ok := readPaymentRequest(w, r)
	if !ok {
		return
	}
	rub, err := strconv.ParseFloat(req.Plan.Price, 64)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid plan amount")
		return
	}
	amount := strconv.Itoa(int(rub)) // DigitalPay wants whole rubles

	backURL := h.returnURL(req.Source)
	if req.Source != "tg" && h.cfg.DigitalPaySBPBackURL != "" {
		backURL = h.cfg.DigitalPaySBPBackURL
	}
	q := url.Values{
		"PartnerPaymentId": {fmt.Sprintf("sbp_%d_%d", req.UserID, time.Now().UnixMilli())},
		"Amount":           {amount},
		"Currency":         {"RUB"},
		"PaymentType":      {"ACQ_SBP"},
		"CallbackUrl":      {strings.TrimRight(h.cfg.BaseURL, "/") + "/api/payments/sbp/webhook"},
		"BackUrl":          {backURL},
		"PaymentLifeTime":  {"15"},
	}
	var data struct {
		PaymentID string `json:"paymentId"`
		Creds     struct {
			PaymentURL string `json:"paymentUrl"`
			CleanURL   string `json:"cleanUrl"` // direct NSPK link
		} `json:"credentials"`
	}
	if err := h.digitalPay("create", q, &data); err != nil || data.PaymentID == "" {
		log.Printf("digitalpay create: id=%q err=%v", data.PaymentID, err)
		writeError(w, http.StatusBadGateway, "payment service unavailable")
		return
	}
	link := data.Creds.CleanURL
	if link == "" {
		link = data.Creds.PaymentURL
	}
	if !h.savePayment(w, req, sbpPrefix+data.PaymentID, amount) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"payment_url": link})
}

// DigitalPayWebhook is unsigned, so it is only a hint: the status is always
// re-read from the DigitalPay API before crediting anything.
func (h *PaymentHandler) DigitalPayWebhook(w http.ResponseWriter, r *http.Request) {
	var cb struct {
		PaymentID string `json:"PaymentId"`
	}
	if err := readJSON(r, &cb); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	externalID := sbpPrefix + cb.PaymentID
	if cb.PaymentID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if _, err := h.db.GetPaymentByExternalID(externalID); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	paid, err := h.sbpPaid(externalID)
	if err != nil {
		log.Printf("digitalpay webhook %s: %v", externalID, err)
	} else if paid {
		if err := h.fulfill(externalID); err != nil {
			log.Printf("digitalpay webhook fulfill %s: %v", externalID, err)
		}
	}
	w.WriteHeader(http.StatusOK)
}

// sbpPaid reads the payment status from DigitalPay; canceled payments are marked as such.
func (h *PaymentHandler) sbpPaid(externalID string) (bool, error) {
	if h.cfg.DigitalPayAPIKey == "" {
		return false, nil
	}
	var data struct {
		Status string `json:"status"`
	}
	q := url.Values{"PaymentId": {strings.TrimPrefix(externalID, sbpPrefix)}}
	if err := h.digitalPay("get", q, &data); err != nil {
		return false, err
	}
	switch data.Status {
	case "Completed", "CompletedOnDispute", "Payed":
		return true, nil
	case "Canceled":
		return false, h.db.CancelPayment(externalID)
	}
	return false, nil
}

// digitalPay calls /api/v3/fiat/payments/<action> and decodes "data" into out.
func (h *PaymentHandler) digitalPay(action string, q url.Values, out any) error {
	endpoint := strings.TrimRight(h.cfg.DigitalPayBaseURL, "/") + "/api/v3/fiat/payments/" + action + "?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", h.cfg.DigitalPayAPIKey)
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var envelope struct {
		OK      bool            `json:"ok"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("digitalpay %s: HTTP %d: %w", action, resp.StatusCode, err)
	}
	if !envelope.OK {
		return fmt.Errorf("digitalpay %s: %s", action, envelope.Message)
	}
	return json.Unmarshal(envelope.Data, out)
}
