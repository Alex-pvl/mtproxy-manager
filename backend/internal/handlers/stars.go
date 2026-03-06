package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"mtproxy-manager/internal/models"
)

const telegramBotAPI = "https://api.telegram.org/bot"

// CreateStarsPayment creates a Telegram Stars (XTR) invoice link via Bot API.
func (h *PaymentHandler) CreateStarsPayment(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if h.cfg.TelegramBotToken == "" {
		writeError(w, http.StatusServiceUnavailable, "stars payments not configured")
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
	if plan.StarsPrice == 0 {
		writeError(w, http.StatusBadRequest, "this plan does not support Stars payment")
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"user_id": strconv.FormatInt(claims.UserID, 10),
		"plan_id": plan.ID,
	})

	invoiceReq := map[string]interface{}{
		"title":       fmt.Sprintf("Подписка Stay VPN — %s", plan.Name),
		"description": fmt.Sprintf("VPN-доступ на %d дней. MTProxy, SOCKS5, VLESS.", plan.DurationDays),
		"payload":     string(payload),
		"currency":    "XTR",
		"prices": []map[string]interface{}{
			{"label": plan.Name, "amount": plan.StarsPrice},
		},
	}

	body, _ := json.Marshal(invoiceReq)
	url := fmt.Sprintf("%s%s/createInvoiceLink", telegramBotAPI, h.cfg.TelegramBotToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("Stars createInvoiceLink error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create stars invoice")
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var tgResp struct {
		OK     bool   `json:"ok"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal(respBody, &tgResp); err != nil || !tgResp.OK {
		log.Printf("Stars createInvoiceLink failed: %s", string(respBody))
		writeError(w, http.StatusInternalServerError, "failed to create stars invoice")
		return
	}

	payment := &models.Payment{
		UserID:     claims.UserID,
		PlanID:     plan.ID,
		ExternalID: fmt.Sprintf("stars_%d_%d", claims.UserID, time.Now().UnixMilli()),
		Amount:     strconv.Itoa(plan.StarsPrice),
		Status:     "pending",
	}
	if err := h.db.CreatePayment(payment); err != nil {
		log.Printf("Failed to save stars payment: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"invoice_link": tgResp.Result,
	})
}

// CreateTonPayment returns the TON wallet address and amount for a direct TON transfer.
func (h *PaymentHandler) CreateTonPayment(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if h.cfg.TonWalletAddress == "" {
		writeError(w, http.StatusServiceUnavailable, "ton payments not configured")
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
	if plan.TonAmount == "" {
		writeError(w, http.StatusBadRequest, "this plan does not support TON payment")
		return
	}

	// Encode comment as base64 for TON transaction payload
	comment := fmt.Sprintf("stay_%d_%s", claims.UserID, plan.ID)

	payment := &models.Payment{
		UserID:     claims.UserID,
		PlanID:     plan.ID,
		ExternalID: comment,
		Amount:     plan.TonAmount,
		Status:     "pending",
	}
	if err := h.db.CreatePayment(payment); err != nil {
		log.Printf("Failed to save TON payment: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"address": h.cfg.TonWalletAddress,
		"amount":  plan.TonAmount,
		"comment": comment,
	})
}

// BotWebhook handles Telegram Bot API webhook updates (successful_payment, pre_checkout_query).
func (h *PaymentHandler) BotWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var update struct {
		UpdateID         int64 `json:"update_id"`
		PreCheckoutQuery *struct {
			ID               string `json:"id"`
			From             struct{ ID int64 `json:"id"` } `json:"from"`
			Currency         string `json:"currency"`
			TotalAmount      int    `json:"total_amount"`
			InvoicePayload   string `json:"invoice_payload"`
		} `json:"pre_checkout_query"`
		Message *struct {
			From             struct{ ID int64 `json:"id"` } `json:"from"`
			SuccessfulPayment *struct {
				Currency                string `json:"currency"`
				TotalAmount             int    `json:"total_amount"`
				InvoicePayload          string `json:"invoice_payload"`
				TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
			} `json:"successful_payment"`
		} `json:"message"`
	}

	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("BotWebhook parse error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Answer pre_checkout_query immediately (required within 10 seconds)
	if update.PreCheckoutQuery != nil {
		h.answerPreCheckoutQuery(update.PreCheckoutQuery.ID, true, "")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle successful Stars payment
	if update.Message != nil && update.Message.SuccessfulPayment != nil {
		sp := update.Message.SuccessfulPayment
		if sp.Currency != "XTR" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var meta struct {
			UserID string `json:"user_id"`
			PlanID string `json:"plan_id"`
		}
		if err := json.Unmarshal([]byte(sp.InvoicePayload), &meta); err != nil {
			log.Printf("BotWebhook Stars payload parse error: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}

		userID, _ := strconv.ParseInt(meta.UserID, 10, 64)
		plan := models.GetPlan(meta.PlanID)
		if plan == nil || userID == 0 {
			log.Printf("BotWebhook: invalid plan=%s or user=%d", meta.PlanID, userID)
			w.WriteHeader(http.StatusOK)
			return
		}

		if err := h.activateSubscription(userID, plan, sp.TelegramPaymentChargeID); err != nil {
			log.Printf("BotWebhook: failed to activate subscription for user %d: %v", userID, err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PaymentHandler) answerPreCheckoutQuery(queryID string, ok bool, errorMsg string) {
	payload := map[string]interface{}{
		"pre_checkout_query_id": queryID,
		"ok":                    ok,
	}
	if !ok && errorMsg != "" {
		payload["error_message"] = errorMsg
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s%s/answerPreCheckoutQuery", telegramBotAPI, h.cfg.TelegramBotToken)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("answerPreCheckoutQuery error: %v", err)
		return
	}
	defer resp.Body.Close()
}

func (h *PaymentHandler) activateSubscription(userID int64, plan *models.Plan, externalID string) error {
	h.db.UpdatePaymentStatus(externalID, "paid")

	existing, _ := h.db.GetActiveSubscription(userID)
	startsAt := time.Now()
	if existing != nil && existing.ExpiresAt.After(startsAt) {
		startsAt = existing.ExpiresAt
	}
	expiresAt := startsAt.AddDate(0, 0, plan.DurationDays)

	var paymentID int64
	if payment, err := h.db.GetPaymentByExternalID(externalID); err == nil && payment != nil {
		paymentID = payment.ID
	}

	sub := &models.Subscription{
		UserID:    userID,
		PlanID:    plan.ID,
		PaymentID: paymentID,
		StartsAt:  startsAt,
		ExpiresAt: expiresAt,
	}
	if err := h.db.CreateSubscription(sub); err != nil {
		return err
	}

	log.Printf("Subscription activated: user=%d plan=%s expires=%s", userID, plan.ID, expiresAt.Format(time.RFC3339))

	if user, err := h.db.GetUserByID(userID); err == nil {
		_ = h.db.UpdateUser(userID, user.Role, plan.MaxProxies)
	}
	h.syncVlessExpiry(userID, expiresAt)

	// Referral bonus
	if referrerID, err := h.db.GetReferrerByReferred(userID); err == nil && referrerID > 0 {
		exists, _ := h.db.ReferralBonusExistsForPayment(paymentID)
		if !exists {
			bonusDays := int(float64(plan.DurationDays) * 0.15)
			if bonusDays > 0 {
				if err := h.db.CreateReferralBonus(referrerID, userID, paymentID, bonusDays); err == nil {
					_ = h.db.ExtendSubscription(referrerID, bonusDays)
					if referrerSub, err := h.db.GetActiveSubscription(referrerID); err == nil && referrerSub != nil {
						h.syncVlessExpiry(referrerID, referrerSub.ExpiresAt)
					}
				}
			}
		}
	}

	return nil
}

// CheckPendingPayments polls CryptoPay for pending invoices and activates subscriptions.
func (h *PaymentHandler) CheckPendingPayments(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if h.cfg.CryptoBotToken == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"updated": false})
		return
	}

	pending, err := h.db.GetPendingPaymentsByUser(claims.UserID)
	if err != nil || len(pending) == 0 {
		writeJSON(w, http.StatusOK, map[string]bool{"updated": false})
		return
	}

	// Build list of external IDs to check (CryptoPay invoice IDs only)
	invoiceIDs := make([]string, 0, len(pending))
	for _, p := range pending {
		// Skip Stars and TON payments (not CryptoPay)
		if len(p.ExternalID) > 0 && p.ExternalID[0] != 's' {
			invoiceIDs = append(invoiceIDs, p.ExternalID)
		}
	}

	if len(invoiceIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]bool{"updated": false})
		return
	}

	updated := false
	for _, p := range pending {
		if p.Status != "pending" {
			continue
		}
		plan := models.GetPlan(p.PlanID)
		if plan == nil {
			continue
		}
		// Re-check via CryptoPay API
		if err := h.checkCryptoPayInvoice(p.ExternalID, p.UserID, plan); err == nil {
			updated = true
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"updated": updated})
}

func (h *PaymentHandler) checkCryptoPayInvoice(invoiceID string, userID int64, plan *models.Plan) error {
	url := fmt.Sprintf("%s/getInvoices?invoice_ids=%s&status=paid", cryptoPayAPI, invoiceID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Crypto-Pay-API-Token", h.cfg.CryptoBotToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Items []struct {
				InvoiceID int64  `json:"invoice_id"`
				Status    string `json:"status"`
			} `json:"items"`
		} `json:"result"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil || !result.OK {
		return fmt.Errorf("cryptopay check failed")
	}

	for _, item := range result.Result.Items {
		if item.Status == "paid" {
			extID := strconv.FormatInt(item.InvoiceID, 10)
			return h.activateSubscription(userID, plan, extID)
		}
	}

	return fmt.Errorf("not paid yet")
}

