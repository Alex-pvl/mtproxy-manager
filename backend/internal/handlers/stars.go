package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mtproxy-manager/internal/models"
)

const telegramBotAPI = "https://api.telegram.org/bot"

const botWelcomeText = `👋 Добро пожаловать в Stay!

🛡️ Получите быстрые и стабильные MTProto/SOCKS5 прокси и VPN в Telegram.

💪 Мгновенная настройка • Высокоскоростные серверы • 100% бесперебойная работа`

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
			ID   string `json:"id"`
			From struct {
				ID int64 `json:"id"`
			} `json:"from"`
			Currency       string `json:"currency"`
			TotalAmount    int    `json:"total_amount"`
			InvoicePayload string `json:"invoice_payload"`
		} `json:"pre_checkout_query"`
		Message *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			From struct {
				ID int64 `json:"id"`
			} `json:"from"`
			Text              string `json:"text"`
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

	// /start — приветствие и ссылка на тарифы
	if update.Message != nil && update.Message.Text != "" && h.cfg.TelegramBotToken != "" {
		text := strings.TrimSpace(update.Message.Text)
		if strings.HasPrefix(text, "/start") {
			payURL := h.cfg.TelegramPayURL
			if payURL == "" {
				payURL = "https://t.me/staytg_bot/pay"
			}
			if err := h.sendTelegramMessage(update.Message.Chat.ID, botWelcomeText, payURL); err != nil {
				log.Printf("BotWebhook /start sendMessage: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
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

		_ = userID
		_ = plan
		// Stars XTR external_id was created earlier as "stars_<userID>_<ms>" — fulfillPayment will look it up.
		// However the webhook only delivers TelegramPaymentChargeID; we look up via payload metadata instead.
		// For backwards compatibility, find the latest pending stars payment of this user+plan and fulfill.
		if pending, err := h.db.GetPendingPaymentsByUser(userID); err == nil {
			for _, p := range pending {
				if p.PlanID == plan.ID && strings.HasPrefix(p.ExternalID, "stars_") {
					if err := h.fulfillPayment(p.ExternalID); err != nil {
						log.Printf("BotWebhook: fulfill stars payment %s: %v", p.ExternalID, err)
					}
					break
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PaymentHandler) telegramPostSendMessage(payload map[string]interface{}) error {
	if h.cfg.TelegramBotToken == "" {
		return fmt.Errorf("telegram bot token empty")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	apiURL := fmt.Sprintf("%s%s/sendMessage", telegramBotAPI, h.cfg.TelegramBotToken)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var tgResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respBody, &tgResp); err != nil {
		return fmt.Errorf("sendMessage parse: %w; body=%s", err, string(respBody))
	}
	if !tgResp.OK {
		return fmt.Errorf("sendMessage: %s", tgResp.Description)
	}
	return nil
}

func (h *PaymentHandler) sendTelegramMessage(chatID int64, text string, payURL string) error {
	payload := map[string]interface{}{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": false,
	}
	if payURL != "" {
		payload["reply_markup"] = map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{{"text": "Открыть Stay", "url": payURL}},
			},
		}
	}
	return h.telegramPostSendMessage(payload)
}

// telegramReplyMarkupOpenStay returns inline_keyboard for opening the Mini App (web_app) or TG_PAY_URL fallback.
func (h *PaymentHandler) telegramReplyMarkupOpenStay() map[string]interface{} {
	miniURL := strings.TrimSpace(strings.TrimRight(h.cfg.BaseURL, "/"))
	payURL := strings.TrimSpace(h.cfg.TelegramPayURL)
	if payURL == "" {
		payURL = "https://t.me/staytg_bot/pay"
	}
	if strings.HasPrefix(miniURL, "https://") {
		return map[string]interface{}{
			"inline_keyboard": [][]map[string]interface{}{
				{
					{
						"text":    "Открыть Stay",
						"web_app": map[string]string{"url": miniURL},
					},
				},
			},
		}
	}
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]interface{}{
			{{"text": "Открыть Stay", "url": payURL}},
		},
	}
}

func formatSubscriptionExpiryForUser(expiresAt time.Time) string {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.UTC
	}
	t := expiresAt.In(loc)
	return t.Format("02.01.2006 15:04") + " (МСК)"
}

// notifyTelegramSubscriptionPaid sends a DM after a successful purchase (CryptoBot, SBP, TON, Stars).
func (h *PaymentHandler) notifyTelegramSubscriptionPaid(userID int64, plan *models.Plan) {
	if h.cfg.TelegramBotToken == "" || plan == nil {
		return
	}
	user, err := h.db.GetUserByID(userID)
	if err != nil || user == nil || user.TelegramID == 0 {
		return
	}

	chatID := user.TelegramID
	planName := plan.Name

	go func() {
		text := fmt.Sprintf(`✅ Успешная оплата!

Ваша подписка Stay на %s активирована.
Откройте приложение, чтобы пользоваться прокси и VPN.`, planName)

		payload := map[string]interface{}{
			"chat_id":                  chatID,
			"text":                     text,
			"disable_web_page_preview": true,
			"reply_markup":             h.telegramReplyMarkupOpenStay(),
		}

		if err := h.telegramPostSendMessage(payload); err != nil {
			log.Printf("notifyTelegramSubscriptionPaid user=%d: %v", userID, err)
		}
	}()
}

func (h *PaymentHandler) sendTelegramSubscriptionExpiryReminder(chatID int64, planName string, expiresAt time.Time, daysLeft int) error {
	when := formatSubscriptionExpiryForUser(expiresAt)
	var text string
	switch daysLeft {
	case 7:
		text = fmt.Sprintf(`⏳ Подписка Stay (%s) закончится через 7 дней.

Доступ активен до: %s

Продлите подписку, чтобы не потерять прокси и VPN.`, planName, when)
	case 1:
		text = fmt.Sprintf(`⚠️ Подписка Stay (%s) заканчивается через 1 день.

Доступ активен до: %s

Продлите подписку, чтобы не потерять доступ.`, planName, when)
	default:
		return fmt.Errorf("unsupported daysLeft=%d", daysLeft)
	}
	payload := map[string]interface{}{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
		"reply_markup":             h.telegramReplyMarkupOpenStay(),
	}
	return h.telegramPostSendMessage(payload)
}

// RunSubscriptionExpiryReminders sends Telegram DMs for subscriptions expiring in 7 or 1 calendar day(s) (UTC). Idempotent via DB flags.
func (h *PaymentHandler) RunSubscriptionExpiryReminders() {
	if h.cfg.TelegramBotToken == "" {
		return
	}

	process := func(daysLeft int, reminder7d bool, mark func(int64) error) {
		rows, err := h.db.ListSubscriptionsExpiryCalendarDays(daysLeft, reminder7d)
		if err != nil {
			log.Printf("RunSubscriptionExpiryReminders list days=%d: %v", daysLeft, err)
			return
		}
		for _, row := range rows {
			plan := models.GetPlan(row.PlanID)
			planName := row.PlanID
			if plan != nil {
				planName = plan.Name
			}
			if err := h.sendTelegramSubscriptionExpiryReminder(row.TelegramID, planName, row.ExpiresAt, daysLeft); err != nil {
				log.Printf("expiry reminder sub=%d user=%d days=%d: %v", row.ID, row.UserID, daysLeft, err)
				continue
			}
			if err := mark(row.ID); err != nil {
				log.Printf("expiry reminder mark sub=%d: %v", row.ID, err)
			}
		}
	}

	process(7, true, h.db.MarkSubscriptionExpiryReminder7dSent)
	process(1, false, h.db.MarkSubscriptionExpiryReminder1dSent)
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

	h.notifyTelegramSubscriptionPaid(userID, plan)

	return nil
}

const tonAPIBase = "https://tonapi.io/v2"

// CheckPendingPayments polls CryptoPay and TonAPI for pending payments and activates subscriptions.
func (h *PaymentHandler) CheckPendingPayments(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	pending, err := h.db.GetPendingPaymentsByUser(claims.UserID)
	if err != nil || len(pending) == 0 {
		writeJSON(w, http.StatusOK, map[string]bool{"updated": false})
		return
	}

	updated := false
	for _, p := range pending {
		if p.Status != "pending" {
			continue
		}
		// VPN payments need plan.TonAmount for TON polling. Non-VPN payments
		// store the nano amount directly in payment.Amount.
		var tonWantNano string
		switch models.ProductType(p.ProductType) {
		case models.ProductStars, models.ProductPremium:
			tonWantNano = p.Amount
		default:
			plan := models.GetPlan(p.PlanID)
			if plan == nil {
				continue
			}
			tonWantNano = plan.TonAmount
		}

		// TON payments: external_id = "stay_..."
		if strings.HasPrefix(p.ExternalID, "stay_") {
			if h.cfg.TonWalletAddress != "" && tonWantNano != "" {
				if err := h.checkTonPaymentByAmount(p.ExternalID, tonWantNano); err == nil {
					updated = true
				}
			}
			continue
		}
		// Stars: external_id starts with "stars_" — handled by BotWebhook.
		if strings.HasPrefix(p.ExternalID, "stars_") {
			continue
		}
		// DigitalPay SBP
		if strings.HasPrefix(p.ExternalID, "sbp_") {
			if h.cfg.DigitalPayAPIKey != "" {
				if err := h.checkDigitalPayPaymentByID(p.ExternalID); err == nil {
					updated = true
				}
			}
			continue
		}
		// CryptoPay
		if h.cfg.CryptoBotToken != "" {
			if err := h.checkCryptoPayInvoiceByID(p.ExternalID); err == nil {
				updated = true
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"updated": updated})
}

func (h *PaymentHandler) checkTonPaymentByAmount(comment, wantAmountNano string) error {
	url := tonAPIBase + "/accounts/" + h.cfg.TonWalletAddress + "/events?limit=30"
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data struct {
		Events []struct {
			Actions []struct {
				Type        string `json:"type"`
				TonTransfer *struct {
					Amount  int64  `json:"amount"`
					Comment string `json:"comment"`
				} `json:"TonTransfer"`
			} `json:"actions"`
		} `json:"events"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	wantAmount, _ := strconv.ParseInt(wantAmountNano, 10, 64)
	for _, ev := range data.Events {
		for _, a := range ev.Actions {
			if a.Type != "TonTransfer" || a.TonTransfer == nil {
				continue
			}
			t := a.TonTransfer
			if t.Comment == comment && t.Amount >= wantAmount {
				return h.fulfillPayment(comment)
			}
		}
	}
	return fmt.Errorf("ton payment not found")
}

func (h *PaymentHandler) checkCryptoPayInvoiceByID(invoiceID string) error {
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
			return h.fulfillPayment(extID)
		}
	}

	return fmt.Errorf("not paid yet")
}
