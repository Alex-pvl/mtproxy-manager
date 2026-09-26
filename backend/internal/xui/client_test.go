package xui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSubscriptionLinkAndSubID(t *testing.T) {
	var addBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/addClient") {
			b, _ := io.ReadAll(r.Body)
			addBody = string(b)
		}
		w.Write([]byte(`{"success":true,"obj":{"id":1,"port":443}}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "p", "u", "pw", "https://vpn.example:2096/sub/", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddClient("abc", "e@x", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(addBody, `\"subId\":\"abc\"`) {
		t.Fatalf("subId not sent: %s", addBody)
	}
	if got := c.UserLink("abc", "r"); got != "https://vpn.example:2096/sub/abc" {
		t.Fatalf("UserLink = %q", got)
	}
	c.subURL = ""
	if got := c.UserLink("abc", "r"); !strings.HasPrefix(got, "vless://abc@") {
		t.Fatalf("fallback = %q", got)
	}
}
