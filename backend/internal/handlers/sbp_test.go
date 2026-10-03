package handlers

import "testing"

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
