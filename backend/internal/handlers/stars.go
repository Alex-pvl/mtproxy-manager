package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const starsPrefix = "stars_"

// CreateStarsPayment creates a Telegram Stars (XTR) invoice link for the Mini App.
func (h *PaymentHandler) CreateStarsPayment(w http.ResponseWriter, r *http.Request) {
	if !h.bot.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "stars payments not configured")
		return
	}
	req, ok := readPaymentRequest(w, r)
	if !ok {
		return
	}
	if req.Plan.StarsPrice == 0 {
		writeError(w, http.StatusBadRequest, "this plan does not support Stars payment")
		return
	}
	externalID := fmt.Sprintf("%s%d_%d", starsPrefix, req.UserID, time.Now().UnixMilli())
	payload, _ := json.Marshal(starsPayload{PaymentID: externalID})

	var link string
	err := h.bot.Call("createInvoiceLink", map[string]any{
		"title":       "Подписка Stay — " + req.Plan.Name,
		"description": fmt.Sprintf("Доступ к сервису Stay на %d дней.", req.Plan.DurationDays),
		"payload":     string(payload),
		"currency":    "XTR",
		"prices":      []map[string]any{{"label": req.Plan.Name, "amount": req.Plan.StarsPrice}},
	}, &link)
	if err != nil {
		log.Printf("stars invoice: %v", err)
		writeError(w, http.StatusBadGateway, "failed to create stars invoice")
		return
	}
	if !h.savePayment(w, req, externalID, strconv.Itoa(req.Plan.StarsPrice)) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"invoice_link": link})
}

// starsPayload is the invoice payload. Invoices created before payment_id was
// added carry only user_id/plan_id.
type starsPayload struct {
	PaymentID string `json:"payment_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	PlanID    string `json:"plan_id,omitempty"`
}

// BotWebhook handles Telegram updates: /start, Stars pre-checkout and payment.
func (h *PaymentHandler) BotWebhook(w http.ResponseWriter, r *http.Request) {
	if s := h.cfg.TelegramWebhookSecret; s != "" &&
		subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Telegram-Bot-Api-Secret-Token")), []byte(s)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var update struct {
		PreCheckoutQuery *struct {
			ID string `json:"id"`
		} `json:"pre_checkout_query"`
		Message *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			Text              string `json:"text"`
			SuccessfulPayment *struct {
				Currency       string `json:"currency"`
				TotalAmount    int    `json:"total_amount"`
				InvoicePayload string `json:"invoice_payload"`
			} `json:"successful_payment"`
		} `json:"message"`
	}
	if err := readJSON(r, &update); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Telegram retries non-2xx responses, so always answer 200 from here on.
	defer w.WriteHeader(http.StatusOK)

	switch msg := update.Message; {
	case update.PreCheckoutQuery != nil:
		if err := h.bot.Call("answerPreCheckoutQuery", map[string]any{
			"pre_checkout_query_id": update.PreCheckoutQuery.ID, "ok": true,
		}, nil); err != nil {
			log.Printf("answerPreCheckoutQuery: %v", err)
		}
	case msg == nil:
	case msg.SuccessfulPayment != nil && msg.SuccessfulPayment.Currency == "XTR":
		sp := msg.SuccessfulPayment
		if err := h.fulfillStars(sp.InvoicePayload, sp.TotalAmount); err != nil {
			log.Printf("stars payment: %v", err)
		}
	case strings.HasPrefix(strings.TrimSpace(msg.Text), "/start"):
		if err := h.bot.SendWelcome(msg.Chat.ID); err != nil {
			log.Printf("/start: %v", err)
		}
	}
}

func (h *PaymentHandler) fulfillStars(invoicePayload string, paidStars int) error {
	var pl starsPayload
	if err := json.Unmarshal([]byte(invoicePayload), &pl); err != nil {
		return fmt.Errorf("bad payload %q: %w", invoicePayload, err)
	}
	externalID := pl.PaymentID
	if externalID == "" { // legacy invoice: latest pending Stars payment for user+plan
		userID, _ := strconv.ParseInt(pl.UserID, 10, 64)
		pending, err := h.db.GetPendingPaymentsByUser(userID)
		if err != nil {
			return err
		}
		for _, p := range pending {
			if p.PlanID == pl.PlanID && strings.HasPrefix(p.ExternalID, starsPrefix) {
				externalID = p.ExternalID
				break
			}
		}
	}
	payment, err := h.db.GetPaymentByExternalID(externalID)
	if err != nil {
		return fmt.Errorf("payment %q not found: %w", externalID, err)
	}
	if want, _ := strconv.Atoi(payment.Amount); paidStars < want {
		return fmt.Errorf("payment %s: paid %d stars, want %d", externalID, paidStars, want)
	}
	return h.fulfill(externalID)
}
