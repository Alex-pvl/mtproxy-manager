package handlers

import (
	"testing"

	"mtproxy-manager/internal/models"
)

func TestVlessEmailNaming(t *testing.T) {
	existing := []models.Proxy{{VlessEmail: "staytg.org-bob-1"}, {VlessEmail: "staytg.org-bob-3"}}
	if got := nextVlessEmail("bob", existing); got != "staytg.org-bob-2" {
		t.Fatalf("next = %q", got)
	}
	if got := nextVlessEmail("bob", nil); got != "staytg.org-bob-1" {
		t.Fatalf("first = %q", got)
	}
	if got := proxyEmail(models.Proxy{Port: 8007, UserID: 2}); got != "proxy-8007-user-2" {
		t.Fatalf("legacy = %q", got)
	}
}
