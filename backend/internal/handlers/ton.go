package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"mtproxy-manager/internal/models"
)

// TON payments are matched by transfer comment, which is the external id.
const (
	tonPrefix  = "stay_"
	tonAPIBase = "https://tonapi.io/v2"
)

// CreateTonPayment returns where and how much to send; the wallet signs the transfer.
func (h *PaymentHandler) CreateTonPayment(w http.ResponseWriter, r *http.Request) {
	if h.cfg.TonWalletAddress == "" {
		writeError(w, http.StatusServiceUnavailable, "ton payments not configured")
		return
	}
	req, ok := readPaymentRequest(w, r)
	if !ok {
		return
	}
	if req.Plan.TonAmount == "" {
		writeError(w, http.StatusBadRequest, "this plan does not support TON payment")
		return
	}
	comment := fmt.Sprintf("%s%d_%s_%d", tonPrefix, req.UserID, req.Plan.ID, time.Now().Unix())
	if !h.savePayment(w, req, comment, req.Plan.TonAmount) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"address": h.cfg.TonWalletAddress,
		"amount":  req.Plan.TonAmount,
		"comment": comment,
	})
}

// tonPaid looks for an incoming transfer with the payment's comment and enough value.
func (h *PaymentHandler) tonPaid(p models.Payment) (bool, error) {
	if h.cfg.TonWalletAddress == "" {
		return false, nil
	}
	want, err := strconv.ParseInt(p.Amount, 10, 64)
	if err != nil {
		return false, fmt.Errorf("bad TON amount %q", p.Amount)
	}
	resp, err := httpClient.Get(tonAPIBase + "/accounts/" + h.cfg.TonWalletAddress + "/events?limit=30")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var data struct {
		Events []struct {
			Actions []struct {
				TonTransfer *struct {
					Amount  int64  `json:"amount"`
					Comment string `json:"comment"`
				} `json:"TonTransfer"`
			} `json:"actions"`
		} `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, fmt.Errorf("tonapi: HTTP %d: %w", resp.StatusCode, err)
	}
	for _, ev := range data.Events {
		for _, a := range ev.Actions {
			if t := a.TonTransfer; t != nil && t.Comment == p.ExternalID && t.Amount >= want {
				return true, nil
			}
		}
	}
	return false, nil
}
