package xui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientWithAPIToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/p/login" {
			t.Error("login must not be called when API token is set")
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"success":true,"obj":{"id":36,"port":443}}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "p", "", "", "tok", 36)
	if err != nil {
		t.Fatal(err)
	}
	if c.inbound.Port != 443 {
		t.Fatalf("inbound not loaded: %+v", c.inbound)
	}
}

func TestInboundSettingsStringOrObject(t *testing.T) {
	for _, raw := range []string{
		`{"streamSettings":"{\"network\":\"tcp\"}"}`, // v2: JSON in a string
		`{"streamSettings":{"network":"tcp"}}`,       // v3: plain object
	} {
		var in Inbound
		if err := json.Unmarshal([]byte(raw), &in); err != nil {
			t.Fatal(err)
		}
		var ss streamSettings
		if err := json.Unmarshal([]byte(in.StreamSettings), &ss); err != nil || ss.Network != "tcp" {
			t.Fatalf("%s: got %+v, %v", raw, ss, err)
		}
	}
}
