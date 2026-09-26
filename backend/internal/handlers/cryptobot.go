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
)

const cryptoPayAPI = "https://pay.crypt.bot/api"

// CreatePayment creates a CryptoBot invoice in RUB; its invoice_id is the external id.
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	if h.cfg.CryptoBotToken == "" {
		writeError(w, http.StatusServiceUnavailable, "cryptobot payments not configured")
		return
	}
	req, ok := readPaymentRequest(w, r)
	if !ok {
		return
	}
	var invoice struct {
		InvoiceID     int64  `json:"invoice_id"`
		BotInvoiceURL string `json:"bot_invoice_url"`
	}
	err := h.cryptoPay(http.MethodPost, "/createInvoice", map[string]any{
		"currency_type": "fiat",
		"fiat":          "RUB",
		"amount":        req.Plan.Price,
		"description":   "Подписка Stay — " + req.Plan.Name,
		"paid_btn_name": "callback",
		"paid_btn_url":  h.returnURL(req.Source),
	}, &invoice)
	if err != nil {
		log.Printf("cryptobot create invoice: %v", err)
		writeError(w, http.StatusBadGateway, "payment service unavailable")
		return
	}
	if !h.savePayment(w, req, strconv.FormatInt(invoice.InvoiceID, 10), req.Plan.Price) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"payment_url": invoice.BotInvoiceURL})
}

// Webhook receives CryptoBot updates, signed with the API token.
func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !validCryptoPaySignature(h.cfg.CryptoBotToken, body, r.Header.Get("crypto-pay-api-signature")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var update struct {
		UpdateType string `json:"update_type"`
		Payload    struct {
			InvoiceID int64 `json:"invoice_id"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &update); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if update.UpdateType == "invoice_paid" {
		if err := h.fulfill(strconv.FormatInt(update.Payload.InvoiceID, 10)); err != nil {
			log.Printf("cryptobot webhook: %v", err)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func validCryptoPaySignature(token string, body []byte, signature string) bool {
	if token == "" || signature == "" {
		return false
	}
	secret := sha256.Sum256([]byte(token))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write(body)
	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(signature))
}

func (h *PaymentHandler) cryptoBotPaid(invoiceID string) (bool, error) {
	if h.cfg.CryptoBotToken == "" {
		return false, nil
	}
	var result struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := h.cryptoPay(http.MethodGet, "/getInvoices?invoice_ids="+invoiceID, nil, &result); err != nil {
		return false, err
	}
	return len(result.Items) > 0 && result.Items[0].Status == "paid", nil
}

// cryptoPay calls the Crypto Pay API and decodes its "result" into out.
func (h *PaymentHandler) cryptoPay(method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, cryptoPayAPI+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Crypto-Pay-API-Token", h.cfg.CryptoBotToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var envelope struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil || !envelope.OK {
		return fmt.Errorf("crypto pay %s: HTTP %d: %s", path, resp.StatusCode, truncateBody(respBody))
	}
	return json.Unmarshal(envelope.Result, out)
}

func truncateBody(b []byte) string {
	if len(b) > 300 {
		return string(b[:300]) + "…"
	}
	return string(b)
}
