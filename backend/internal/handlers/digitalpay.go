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

	"mtproxy-manager/internal/models"
)

const digitalPayPaymentTypeSBP = "ACQ_SBP"

// CreateSBPPayment creates an SBP (NSPK) payment via DigitalPay.
func (h *PaymentHandler) CreateSBPPayment(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.cfg.DigitalPayAPIKey == "" {
		writeError(w, http.StatusServiceUnavailable, "sbp payments not configured")
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

	amountFloat, err := strconv.ParseFloat(plan.Price, 64)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid plan amount")
		return
	}
	amount := strconv.Itoa(int(amountFloat))

	partnerPaymentID := fmt.Sprintf("sbp_%d_%d", claims.UserID, time.Now().UnixMilli())
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

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("DigitalPay create error: %v", err)
		writeError(w, http.StatusInternalServerError, "payment service unavailable")
		return
	}
	defer resp.Body.Close()

	var digitalResp struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
		Data    struct {
			PaymentID string `json:"paymentId"`
			Status    string `json:"status"`
			Type      string `json:"type"`
			Currency  string `json:"currency"`
			Creds     struct {
				PaymentURL string `json:"PaymentUrl"`
			} `json:"credentials"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&digitalResp); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}
	if !digitalResp.OK || digitalResp.Data.PaymentID == "" || digitalResp.Data.Creds.PaymentURL == "" {
		log.Printf("DigitalPay create failed: ok=%v message=%s", digitalResp.OK, digitalResp.Message)
		writeError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	externalID := "sbp_" + digitalResp.Data.PaymentID
	payment := &models.Payment{
		UserID:     claims.UserID,
		PlanID:     plan.ID,
		ExternalID: externalID,
		Amount:     amount,
		Status:     "pending",
	}
	if err := h.db.CreatePayment(payment); err != nil {
		log.Printf("Failed to save SBP payment: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"payment_url": digitalResp.Data.Creds.PaymentURL,
	})
}

// DigitalPayWebhook handles asynchronous callbacks from DigitalPay.
func (h *PaymentHandler) DigitalPayWebhook(w http.ResponseWriter, r *http.Request) {
	var cb struct {
		PaymentID string `json:"PaymentId"`
		Status    int    `json:"Status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if cb.PaymentID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	externalID := "sbp_" + cb.PaymentID
	payment, err := h.db.GetPaymentByExternalID(externalID)
	if err != nil || payment == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch cb.Status {
	case 1, 4, 5: // Completed, CompletedOnDispute, Payed
		if payment.Status == "paid" {
			w.WriteHeader(http.StatusOK)
			return
		}
		plan := models.GetPlan(payment.PlanID)
		if plan == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		if err := h.activateSubscription(payment.UserID, plan, payment.ExternalID); err != nil {
			log.Printf("DigitalPay webhook activate failed: %v", err)
		}
	case 3: // Canceled
		_ = h.db.UpdatePaymentStatus(externalID, "canceled")
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PaymentHandler) checkDigitalPayPayment(externalID string, userID int64, plan *models.Plan) error {
	paymentID := strings.TrimPrefix(externalID, "sbp_")
	if paymentID == "" || paymentID == externalID {
		return fmt.Errorf("invalid digitalpay external id")
	}

	q := url.Values{}
	q.Set("PaymentId", paymentID)
	endpoint := strings.TrimRight(h.cfg.DigitalPayBaseURL, "/") + "/api/v3/fiat/payments/get?" + q.Encode()

	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("apikey", h.cfg.DigitalPayAPIKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || !result.OK {
		return fmt.Errorf("digitalpay check failed")
	}

	switch result.Data.Status {
	case "Completed", "CompletedOnDispute", "Payed":
		return h.activateSubscription(userID, plan, externalID)
	case "Canceled":
		return h.db.UpdatePaymentStatus(externalID, "canceled")
	default:
		return fmt.Errorf("not paid yet")
	}
}
