package handlers

import (
	"testing"

	"mtproxy-manager/internal/models"
)

func TestNextClientName(t *testing.T) {
	existing := []models.Proxy{{VlessEmail: "staytg.org-bob-1"}, {VlessEmail: "staytg.org-bob-3"}}
	if got := nextClientName("bob", existing); got != "staytg.org-bob-2" {
		t.Fatalf("got %q", got)
	}
	if got := nextClientName("bob", nil); got != "staytg.org-bob-1" {
		t.Fatalf("got %q", got)
	}
}
