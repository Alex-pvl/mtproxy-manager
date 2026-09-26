package xui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientV3Calls(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b, _ := io.ReadAll(r.Body)
		got = append(got, r.Method+" "+r.URL.Path+" "+string(b))
		w.Write([]byte(`{"success":true,"obj":{}}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "/p/", "tok", "https://vpn.example:2096/sub/", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddClient("u1", "a@b", time.UnixMilli(1000)); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveClient("a@b"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /p/panel/api/inbounds/get/1 ",
		`POST /p/panel/api/clients/add {"client":{"id":"u1","email":"a@b","limitIp":1,"totalGB":0,"expiryTime":1000,"enable":true,"subId":"u1"},"inboundIds":[1]}`,
		"POST /p/panel/api/clients/del/a@b ",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got:\n%s", strings.Join(got, "\n"))
	}
	if s := c.SubscriptionURL("u1"); s != "https://vpn.example:2096/sub/u1" {
		t.Fatalf("SubscriptionURL = %q", s)
	}

	if _, err := NewClient(srv.URL, "p", "wrong", "x", 1); err == nil || !strings.Contains(err.Error(), "XUI_API_TOKEN") {
		t.Fatalf("bad token should fail with a hint, got %v", err)
	}
}
