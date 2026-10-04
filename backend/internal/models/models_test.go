package models

import (
	"testing"
	"time"
)

func TestSBPPromo(t *testing.T) {
	during, after := SBPPromoUntil.Add(-time.Second), SBPPromoUntil
	p := GetPlan("month_1")
	if p.SBPAmount(during) != "160.00" || p.SBPAmount(after) != "200.00" {
		t.Fatalf("SBPAmount: during=%s after=%s", p.SBPAmount(during), p.SBPAmount(after))
	}
	if a := ActivePlans(during)[0]; a.SBPPriceLabel != "160 ₽" || a.OriginalPriceLabel != "200 ₽" || a.DiscountPercent != 20 || a.Price != "200.00" {
		t.Fatalf("during promo: %+v", a)
	}
	if a := ActivePlans(after)[0]; a.SBPPriceLabel != "" || a.DiscountPercent != 0 || a.OriginalPriceLabel != "" {
		t.Fatalf("after promo: %+v", a)
	}
	if Plans[0].SBPPrice == "" {
		t.Fatal("ActivePlans mutated Plans")
	}
}
