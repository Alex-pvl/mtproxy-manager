package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mtproxy-manager/internal/models"
)

// startCryptoBotForProduct creates a CryptoBot RUB invoice for any product
// (VPN, Stars, Premium) and persists a Payment row in pending state.
func (h *PaymentHandler) startCryptoBotForProduct(userID int64, planID, amountRub, metadata, productType, source string) (*models.Payment, string, error) {
	payload, _ := json.Marshal(map[string]string{
		"user_id": strconv.FormatInt(userID, 10),
		"plan_id": planID,
		"source":  source,
	})
	returnURL := h.paymentReturnURL(source)

	description := "Stay покупка"
	switch models.ProductType(productType) {
	case models.ProductStars:
		description = "Stay — покупка Telegram Stars"
	case models.ProductPremium:
		description = "Stay — покупка Telegram Premium"
	}

	invoiceReq := map[string]interface{}{
		"currency_type": "fiat",
		"fiat":          "RUB",
		"amount":        amountRub,
		"description":   description,
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
		return nil, "", fmt.Errorf("cryptobot request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var cryptoResp struct {
		OK     bool `json:"ok"`
		Result struct {
			InvoiceID     int64  `json:"invoice_id"`
			BotInvoiceURL string `json:"bot_invoice_url"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &cryptoResp); err != nil || !cryptoResp.OK {
		return nil, "", fmt.Errorf("cryptobot create failed: %s", string(respBody))
	}

	payment := &models.Payment{
		UserID:      userID,
		PlanID:      planID,
		ExternalID:  strconv.FormatInt(cryptoResp.Result.InvoiceID, 10),
		Amount:      amountRub,
		Status:      "pending",
		ProductType: productType,
		Metadata:    metadata,
	}
	if err := h.db.CreatePayment(payment); err != nil {
		return nil, "", err
	}
	return payment, cryptoResp.Result.BotInvoiceURL, nil
}

// startSBPForProduct creates a DigitalPay SBP payment for any product.
func (h *PaymentHandler) startSBPForProduct(userID int64, planID, amountRub, metadata, productType, source string) (*models.Payment, string, error) {
	// DigitalPay expects integer amount in RUB
	amountFloat, err := strconv.ParseFloat(amountRub, 64)
	if err != nil {
		return nil, "", fmt.Errorf("invalid amount: %w", err)
	}
	amount := strconv.Itoa(int(amountFloat))

	partnerPaymentID := fmt.Sprintf("sbp_%d_%d", userID, time.Now().UnixMilli())
	callbackURL := strings.TrimRight(h.cfg.BaseURL, "/") + "/api/payments/sbp/webhook"
	backURL := h.paymentReturnURL(source)
	if source != "tg" && h.cfg.DigitalPaySBPBackURL != "" {
		backURL = h.cfg.DigitalPaySBPBackURL
	}

	q := url.Values{}
	q.Set("PartnerPaymentId", partnerPaymentID)
	q.Set("Amount", amount)
	q.Set("Currency", "RUB")
	q.Set("PaymentType", digitalPayPaymentTypeSBP)
	q.Set("CallbackUrl", callbackURL)
	q.Set("BackUrl", backURL)
	q.Set("PaymentLifeTime", "15")

	endpoint := strings.TrimRight(h.cfg.DigitalPayBaseURL, "/") + "/api/v3/fiat/payments/create?" + q.Encode()
	httpReq, _ := http.NewRequest("GET", endpoint, nil)
	httpReq.Header.Set("apikey", h.cfg.DigitalPayAPIKey)
	httpReq.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("digitalpay request: %w", err)
	}
	defer resp.Body.Close()

	var digitalResp struct {
		OK   bool `json:"ok"`
		Data struct {
			PaymentID string `json:"paymentId"`
			Creds     struct {
				PaymentURL string `json:"paymentUrl"`
				CleanURL   string `json:"cleanUrl"`
			} `json:"credentials"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&digitalResp); err != nil {
		return nil, "", fmt.Errorf("digitalpay parse: %w", err)
	}
	paymentLink := digitalResp.Data.Creds.CleanURL
	if paymentLink == "" {
		paymentLink = digitalResp.Data.Creds.PaymentURL
	}
	if !digitalResp.OK || digitalResp.Data.PaymentID == "" || paymentLink == "" {
		return nil, "", fmt.Errorf("digitalpay create failed: %s", digitalResp.Message)
	}

	externalID := "sbp_" + digitalResp.Data.PaymentID
	payment := &models.Payment{
		UserID:      userID,
		PlanID:      planID,
		ExternalID:  externalID,
		Amount:      amount,
		Status:      "pending",
		ProductType: productType,
		Metadata:    metadata,
	}
	if err := h.db.CreatePayment(payment); err != nil {
		return nil, "", err
	}
	return payment, paymentLink, nil
}
