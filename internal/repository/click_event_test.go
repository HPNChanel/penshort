package repository

import (
	"testing"

	"github.com/penshort/penshort/internal/model"
)

func TestAccumulateDailyStats(t *testing.T) {
	events := []*model.ClickEvent{
		{
			Referrer:    "https://example.com/page",
			CountryCode: "US",
			VisitorHash: "visitor-a",
		},
		{
			Referrer:    "",
			CountryCode: "US",
			VisitorHash: "visitor-b",
		},
		{
			Referrer:    "https://example.com/other",
			CountryCode: "VN",
			VisitorHash: "visitor-a",
		},
	}

	acc := accumulateDailyStats(events)

	if acc.totalClicks != 3 {
		t.Fatalf("expected total clicks 3, got %d", acc.totalClicks)
	}
	if acc.uniqueVisitors != 2 {
		t.Fatalf("expected unique visitors 2, got %d", acc.uniqueVisitors)
	}
	if acc.referrers["example.com"] != 2 {
		t.Fatalf("expected example.com referrers 2, got %d", acc.referrers["example.com"])
	}
	if acc.referrers["(direct)"] != 1 {
		t.Fatalf("expected direct referrers 1, got %d", acc.referrers["(direct)"])
	}
	if acc.countries["US"] != 2 {
		t.Fatalf("expected US clicks 2, got %d", acc.countries["US"])
	}
	if acc.countries["VN"] != 1 {
		t.Fatalf("expected VN clicks 1, got %d", acc.countries["VN"])
	}
}
