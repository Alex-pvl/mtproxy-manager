package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mtproxy-manager/internal/config"
)

func TestRollyPaySigned(t *testing.T) {
	body := []byte(`{"payment_id":"pay_1","status":"paid"}`)
	// echo -n '1700000000.{"payment_id":"pay_1","status":"paid"}' | openssl dgst -sha256 -hmac secret
	sig := "4eaedcf075a6405975edf7fc9606d821a0b3a2c219beba356bc2cd03bd6ab24e"
	if !rollyPaySigned(body, "1700000000", sig, "secret") {
		t.Fatal("valid signature rejected")
	}
	if rollyPaySigned(body, "1700000001", sig, "secret") || rollyPaySigned(body, "1700000000", sig, "other") || rollyPaySigned(body, "1700000000", sig, "") {
		t.Fatal("bad signature accepted")
	}
}

func TestSBPTerminalRouting(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		w.Write([]byte(`{"status":"paid"}`))
	}))
	defer srv.Close()
	h := &PaymentHandler{cfg: &config.Config{RollyPayBaseURL: srv.URL, RollyPayAPIKey: "site", RollyPayTGAPIKey: "bot"}}

	if h.sbpTerminalFor("tg").prefix != sbpTGPrefix || h.sbpTerminalFor("web").prefix != sbpPrefix {
		t.Fatal("wrong terminal for new payment")
	}
	for id, key := range map[string]string{"sbp_pay_1": "site", "sbp_tg_pay_1": "bot"} {
		if paid, err := h.sbpPaid(id); !paid || err != nil || gotKey != key {
			t.Fatalf("%s: paid=%v err=%v key=%q, want key %q", id, paid, err, gotKey, key)
		}
	}
	// Bot terminal removed from config: its old payments are skipped, not sent to the site terminal.
	h.cfg.RollyPayTGAPIKey, gotKey = "", ""
	if paid, _ := h.sbpPaid("sbp_tg_pay_1"); paid || gotKey != "" {
		t.Fatal("bot payment polled through site terminal")
	}
}
