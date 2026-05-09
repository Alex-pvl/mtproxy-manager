package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mtproxy-manager/internal/fragment"
	"mtproxy-manager/internal/models"

	"github.com/go-chi/chi/v5"
)

// productMetadata is stored as JSON in Payment.Metadata for non-VPN products.
type productMetadata struct {
	Recipient string `json:"recipient"`
	Quantity  int    `json:"quantity"`
}

// ─── Quote endpoints ─────────────────────────────────────────────────────────

type quoteResponse struct {
	Type        string `json:"type"`
	Quantity    int    `json:"quantity"`
	TonCostNano int64  `json:"ton_cost_nano"`
	TonAmount   string `json:"ton_amount"` // for TON wallet payment (with markup applied)
	PriceRUB    string `json:"price_rub"`
	PriceUSD    string `json:"price_usd"`
	StarsPrice  int    `json:"stars_price,omitempty"` // not used; kept for symmetry with Plan
	MarkupPct   int    `json:"markup_pct"`
}

// applyMarkupNano applies markupPct to nano amount.
func applyMarkupNano(nano int64, markupPct int) int64 {
	return nano + nano*int64(markupPct)/100
}

func formatRUB(rub float64) string {
	return fmt.Sprintf("%.0f ₽", math.Ceil(rub))
}

func formatUSD(usd float64) string {
	return fmt.Sprintf("~$%.2f", usd)
}

func (h *PaymentHandler) buildQuote(productType string, quantity int) (*quoteResponse, error) {
	if h.fragment == nil || !h.fragment.Available() {
		return nil, errors.New("fragment worker not configured")
	}
	q, err := h.fragment.Quote(productType, quantity)
	if err != nil {
		return nil, err
	}

	markupPct := h.cfg.StarsMarkupPct
	if productType == string(models.ProductPremium) {
		markupPct = h.cfg.PremiumMarkupPct
	}
	finalNano := applyMarkupNano(q.TonCostNano, markupPct)
	finalTon := float64(finalNano) / 1_000_000_000
	priceRub := finalTon * q.TonRubRate
	priceUsd := finalTon * q.TonUsdRate

	return &quoteResponse{
		Type:        productType,
		Quantity:    quantity,
		TonCostNano: finalNano,
		TonAmount:   strconv.FormatInt(finalNano, 10),
		PriceRUB:    formatRUB(priceRub),
		PriceUSD:    formatUSD(priceUsd),
		MarkupPct:   markupPct,
	}, nil
}

func (h *PaymentHandler) getQuoteFromQuery(r *http.Request, productType string) (*quoteResponse, int, string) {
	qty, err := strconv.Atoi(r.URL.Query().Get("quantity"))
	if err != nil || qty <= 0 {
		return nil, http.StatusBadRequest, "invalid quantity"
	}
	if !validateQuantity(productType, qty) {
		return nil, http.StatusBadRequest, "unsupported quantity"
	}
	q, err := h.buildQuote(productType, qty)
	if err != nil {
		return nil, http.StatusServiceUnavailable, err.Error()
	}
	return q, http.StatusOK, ""
}

func validateQuantity(productType string, qty int) bool {
	switch productType {
	case string(models.ProductStars):
		return qty >= 50 && qty <= 1_000_000
	case string(models.ProductPremium):
		for _, m := range models.PremiumMonthsOptions {
			if m == qty {
				return true
			}
		}
	}
	return false
}

func (h *PaymentHandler) GetStarsQuote(w http.ResponseWriter, r *http.Request) {
	q, status, errMsg := h.getQuoteFromQuery(r, string(models.ProductStars))
	if errMsg != "" {
		writeError(w, status, errMsg)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (h *PaymentHandler) GetPremiumQuote(w http.ResponseWriter, r *http.Request) {
	q, status, errMsg := h.getQuoteFromQuery(r, string(models.ProductPremium))
	if errMsg != "" {
		writeError(w, status, errMsg)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

// ─── Username check ──────────────────────────────────────────────────────────

func (h *PaymentHandler) CheckUsername(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("u")), "@")
	if username == "" {
		writeError(w, http.StatusBadRequest, "username required")
		return
	}
	if h.fragment == nil || !h.fragment.Available() {
		writeError(w, http.StatusServiceUnavailable, "fragment worker not configured")
		return
	}
	res, err := h.fragment.CheckUsername(username)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ─── Order creation (Stars/Premium) ──────────────────────────────────────────

type createProductOrderRequest struct {
	Type      string `json:"type"`     // "stars" | "premium"
	Recipient string `json:"recipient"` // username (no @)
	Quantity  int    `json:"quantity"`
	Method    string `json:"method"` // "sbp" | "cryptobot" | "ton"
	Source    string `json:"source,omitempty"`
}

type createProductOrderResponse struct {
	OrderID    int64  `json:"order_id"`
	PriceRUB   string `json:"price_rub"`
	PriceUSD   string `json:"price_usd"`
	PaymentURL string `json:"payment_url,omitempty"`
	// TON-specific:
	Address string `json:"address,omitempty"`
	Amount  string `json:"amount,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// CreateProductOrder creates a Stars/Premium order: builds a Payment with
// product_type/metadata, then routes to the chosen payment method.
func (h *PaymentHandler) CreateProductOrder(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.fragment == nil || !h.fragment.Available() {
		writeError(w, http.StatusServiceUnavailable, "fragment worker not configured")
		return
	}

	var req createProductOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Type != string(models.ProductStars) && req.Type != string(models.ProductPremium) {
		writeError(w, http.StatusBadRequest, "invalid product type")
		return
	}
	if !validateQuantity(req.Type, req.Quantity) {
		writeError(w, http.StatusBadRequest, "unsupported quantity")
		return
	}
	req.Recipient = strings.TrimPrefix(strings.TrimSpace(req.Recipient), "@")
	if req.Recipient == "" {
		writeError(w, http.StatusBadRequest, "recipient required")
		return
	}

	// Pre-flight: check wallet balance covers the cost.
	q, err := h.fragment.Quote(req.Type, req.Quantity)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to quote: "+err.Error())
		return
	}
	balance, err := h.fragment.Balance()
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to read balance: "+err.Error())
		return
	}
	// Block if balance < cost + safety floor.
	if balance.TonBalanceNano < q.TonCostNano+h.cfg.MinTonBalanceNano {
		go h.notifyAdminLowBalance(balance.TonBalanceNano, q.TonCostNano)
		writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable, please try again later")
		return
	}

	// Validate recipient via Fragment. Skip silently if check is unavailable.
	if check, err := h.fragment.CheckUsername(req.Recipient); err == nil && !check.OK {
		reason := check.Reason
		if reason == "" {
			reason = "recipient cannot accept this product"
		}
		writeError(w, http.StatusBadRequest, reason)
		return
	}

	source := normalizePaymentSource(req.Source)

	// Compute final amount using markup.
	markupPct := h.cfg.StarsMarkupPct
	if req.Type == string(models.ProductPremium) {
		markupPct = h.cfg.PremiumMarkupPct
	}
	finalNano := applyMarkupNano(q.TonCostNano, markupPct)
	finalTon := float64(finalNano) / 1_000_000_000
	priceRub := math.Ceil(finalTon * q.TonRubRate)
	priceUsd := finalTon * q.TonUsdRate
	amountRub := strconv.FormatFloat(priceRub, 'f', 2, 64)

	planID := fmt.Sprintf("%s_%d", req.Type, req.Quantity)
	metaJSON, _ := json.Marshal(productMetadata{Recipient: req.Recipient, Quantity: req.Quantity})

	// Route to selected payment method.
	switch strings.ToLower(req.Method) {
	case "sbp":
		if h.cfg.DigitalPayAPIKey == "" {
			writeError(w, http.StatusServiceUnavailable, "sbp not configured")
			return
		}
		payment, payURL, err := h.startSBPForProduct(claims.UserID, planID, amountRub, string(metaJSON), req.Type, source)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		orderID, err := h.attachFragmentOrder(payment, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create order")
			return
		}
		writeJSON(w, http.StatusOK, createProductOrderResponse{
			OrderID:    orderID,
			PriceRUB:   formatRUB(priceRub),
			PriceUSD:   formatUSD(priceUsd),
			PaymentURL: payURL,
		})
		return

	case "cryptobot":
		if h.cfg.CryptoBotToken == "" {
			writeError(w, http.StatusServiceUnavailable, "cryptobot not configured")
			return
		}
		payment, payURL, err := h.startCryptoBotForProduct(claims.UserID, planID, amountRub, string(metaJSON), req.Type, source)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		orderID, err := h.attachFragmentOrder(payment, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create order")
			return
		}
		writeJSON(w, http.StatusOK, createProductOrderResponse{
			OrderID:    orderID,
			PriceRUB:   formatRUB(priceRub),
			PriceUSD:   formatUSD(priceUsd),
			PaymentURL: payURL,
		})
		return

	case "ton":
		if h.cfg.TonWalletAddress == "" {
			writeError(w, http.StatusServiceUnavailable, "ton not configured")
			return
		}
		comment := fmt.Sprintf("stay_%d_%s", claims.UserID, planID)
		payment := &models.Payment{
			UserID:      claims.UserID,
			PlanID:      planID,
			ExternalID:  comment,
			Amount:      strconv.FormatInt(finalNano, 10),
			Status:      "pending",
			ProductType: req.Type,
			Metadata:    string(metaJSON),
		}
		if err := h.db.CreatePayment(payment); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create payment")
			return
		}
		orderID, err := h.attachFragmentOrder(payment, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create order")
			return
		}
		writeJSON(w, http.StatusOK, createProductOrderResponse{
			OrderID:  orderID,
			PriceRUB: formatRUB(priceRub),
			PriceUSD: formatUSD(priceUsd),
			Address:  h.cfg.TonWalletAddress,
			Amount:   strconv.FormatInt(finalNano, 10),
			Comment:  comment,
		})
		return

	case "stars":
		writeError(w, http.StatusBadRequest, "Stars cannot be used to buy Stars/Premium")
		return
	}

	writeError(w, http.StatusBadRequest, "invalid payment method")
}

func (h *PaymentHandler) attachFragmentOrder(payment *models.Payment, req createProductOrderRequest) (int64, error) {
	order := &models.FragmentOrder{
		PaymentID:         payment.ID,
		UserID:            payment.UserID,
		Type:              req.Type,
		RecipientUsername: req.Recipient,
		Quantity:          req.Quantity,
		Status:            "pending",
	}
	if err := h.db.CreateFragmentOrder(order); err != nil {
		log.Printf("CreateFragmentOrder: %v", err)
		return 0, err
	}
	return order.ID, nil
}

// ─── Order status (frontend polls this after payment) ────────────────────────

type orderStatusResponse struct {
	OrderID  int64  `json:"order_id"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
	TxHash   string `json:"tx_hash,omitempty"`
	Error    string `json:"error,omitempty"`
}

func (h *PaymentHandler) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	order, err := h.db.GetFragmentOrderByID(id)
	if err != nil || order == nil || order.UserID != claims.UserID {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, orderStatusResponse{
		OrderID: order.ID, Status: order.Status, Type: order.Type,
		Quantity: order.Quantity, TxHash: order.FragmentTxHash, Error: order.Error,
	})
}

// ─── Fulfillment ─────────────────────────────────────────────────────────────

// fulfillPayment is the central dispatcher invoked once a payment is confirmed
// by any backend (CryptoPay/SBP/TON/Stars). It branches by product_type.
func (h *PaymentHandler) fulfillPayment(externalID string) error {
	payment, err := h.db.GetPaymentByExternalID(externalID)
	if err != nil || payment == nil {
		return fmt.Errorf("payment not found: %s", externalID)
	}
	if payment.Status != "paid" {
		_ = h.db.UpdatePaymentStatus(externalID, "paid")
	}
	switch models.ProductType(payment.ProductType) {
	case models.ProductStars, models.ProductPremium:
		return h.fulfillFragmentOrder(payment)
	default: // VPN
		plan := models.GetPlan(payment.PlanID)
		if plan == nil {
			return fmt.Errorf("invalid VPN plan %s for payment %s", payment.PlanID, externalID)
		}
		return h.activateSubscription(payment.UserID, plan, externalID)
	}
}

func (h *PaymentHandler) fulfillFragmentOrder(payment *models.Payment) error {
	order, err := h.db.GetFragmentOrderByPaymentID(payment.ID)
	if err != nil || order == nil {
		return fmt.Errorf("fragment order not found for payment %d", payment.ID)
	}
	if order.Status == "delivered" {
		return nil
	}
	_ = h.db.UpdateFragmentOrderStatus(order.ID, "processing", order.FragmentTxHash, "")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("fulfillFragmentOrder panic order=%d: %v", order.ID, r)
			}
		}()
		res, err := h.fragment.Purchase(fragment.PurchaseRequest{
			OrderID:   order.ID,
			Type:      order.Type,
			Recipient: order.RecipientUsername,
			Quantity:  order.Quantity,
		})
		if err != nil {
			log.Printf("fragment purchase order=%d: %v", order.ID, err)
			_ = h.db.UpdateFragmentOrderStatus(order.ID, "failed", "", err.Error())
			h.notifyFragmentResult(order, "failed", err.Error())
			return
		}
		status := res.Status
		if status == "" {
			status = "delivered"
		}
		_ = h.db.UpdateFragmentOrderStatus(order.ID, status, res.TxHash, res.Error)
		h.notifyFragmentResult(order, status, res.Error)
	}()
	return nil
}

func (h *PaymentHandler) notifyFragmentResult(order *models.FragmentOrder, status, errMsg string) {
	if h.cfg.TelegramBotToken == "" {
		return
	}
	user, err := h.db.GetUserByID(order.UserID)
	if err != nil || user == nil || user.TelegramID == 0 {
		return
	}
	var text string
	switch status {
	case "delivered":
		if order.Type == string(models.ProductStars) {
			text = fmt.Sprintf("✅ Готово! %d ⭐ отправлено пользователю @%s.", order.Quantity, order.RecipientUsername)
		} else {
			text = fmt.Sprintf("✅ Готово! Telegram Premium на %d мес. отправлен пользователю @%s.", order.Quantity, order.RecipientUsername)
		}
	case "processing":
		text = "⏳ Платёж принят, заказ выполняется. Уведомим, как только доставим."
	default:
		text = fmt.Sprintf("⚠️ Не удалось доставить заказ. Свяжитесь с поддержкой.\nКод заказа: %d\n%s", order.ID, errMsg)
	}
	_ = h.telegramPostSendMessage(map[string]interface{}{
		"chat_id":                  user.TelegramID,
		"text":                     text,
		"disable_web_page_preview": true,
	})
}

func (h *PaymentHandler) notifyAdminLowBalance(balanceNano, neededNano int64) {
	if h.cfg.TelegramBotToken == "" || h.cfg.AdminTelegramID == 0 {
		return
	}
	balTon := float64(balanceNano) / 1_000_000_000
	needTon := float64(neededNano) / 1_000_000_000
	text := fmt.Sprintf("⚠️ Stay: TON баланс = %.4f TON, требуется %.4f TON. Покупка отклонена. Пополни Fragment-кошелёк.", balTon, needTon)
	_ = h.telegramPostSendMessage(map[string]interface{}{
		"chat_id": h.cfg.AdminTelegramID,
		"text":    text,
	})
}

// ─── Pending order recovery loop ─────────────────────────────────────────────

// RetryPendingFragmentOrders polls the worker for orders stuck in "processing".
// Called periodically from main.go to recover from worker restarts.
func (h *PaymentHandler) RetryPendingFragmentOrders() {
	if h.fragment == nil || !h.fragment.Available() {
		return
	}
	orders, err := h.db.ListPendingFragmentOrders(50)
	if err != nil {
		log.Printf("RetryPendingFragmentOrders list: %v", err)
		return
	}
	for _, o := range orders {
		// Skip very fresh orders — let the initial goroutine work.
		if time.Since(o.UpdatedAt) < 2*time.Minute {
			continue
		}
		res, err := h.fragment.OrderStatus(o.ID)
		if err != nil {
			continue
		}
		if res.Status == "delivered" || res.Status == "failed" {
			_ = h.db.UpdateFragmentOrderStatus(o.ID, res.Status, res.TxHash, res.Error)
			h.notifyFragmentResult(o, res.Status, res.Error)
		}
	}
}
