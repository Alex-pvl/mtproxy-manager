package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func signInitData(vals url.Values, botToken string) string {
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte("auth_date=" + vals.Get("auth_date") + "\nuser=" + vals.Get("user")))
	vals.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return vals.Encode()
}

func TestValidateInitData(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	vals := url.Values{"auth_date": {strconv.FormatInt(now.Unix(), 10)}, "user": {`{"id":42,"username":"bob"}`}}
	data := signInitData(vals, "tok")

	u, err := validateInitData(data, "tok", now)
	if err != nil || u.ID != 42 || u.Username != "bob" {
		t.Fatalf("valid: %+v %v", u, err)
	}
	if _, err := validateInitData(data, "other-token", now); err == nil {
		t.Fatal("wrong token accepted")
	}
	if _, err := validateInitData(data, "tok", now.Add(25*time.Hour)); err == nil {
		t.Fatal("stale init_data accepted")
	}
}
