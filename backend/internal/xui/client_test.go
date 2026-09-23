package xui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestClientOpsUseV3Paths(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.Path)
		w.Write([]byte(`{"success":true,"obj":{"id":36}}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "p", "", "", "tok", 36)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddClient("u", "a@b", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateClientExpiry("u", "a@b", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveClient("a@b"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /p/panel/api/inbounds/get/36",
		"POST /p/panel/api/clients/add",
		"POST /p/panel/api/clients/update/a@b",
		"POST /p/panel/api/clients/del/a@b",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got:\n%s", strings.Join(got, "\n"))
	}
}
