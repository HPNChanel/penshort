package analytics

import (
	"testing"
	"time"
)

func TestValidateClickEventPayload(t *testing.T) {
	valid := ClickEventPayload{
		ShortCode:   "abc123",
		LinkID:      "link-1",
		Referrer:    "https://example.com/path",
		UserAgent:   "TestAgent/1.0",
		VisitorHash: "0123456789abcdef",
		CountryCode: "US",
		ClickedAt:   time.Now().UnixMilli(),
	}

	if err := ValidateClickEventPayload(valid); err != nil {
		t.Fatalf("expected valid payload, got %v", err)
	}

	cases := []struct {
		name    string
		payload ClickEventPayload
	}{
		{"missing_short_code", ClickEventPayload{LinkID: "link", VisitorHash: "0123456789abcdef", ClickedAt: 1}},
		{"short_code_too_short", ClickEventPayload{ShortCode: "ab", LinkID: "link", VisitorHash: "0123456789abcdef", ClickedAt: 1}},
		{"missing_link_id", ClickEventPayload{ShortCode: "abc", VisitorHash: "0123456789abcdef", ClickedAt: 1}},
		{"missing_visitor_hash", ClickEventPayload{ShortCode: "abc", LinkID: "link", ClickedAt: 1}},
		{"invalid_visitor_hash", ClickEventPayload{ShortCode: "abc", LinkID: "link", VisitorHash: "not-hex", ClickedAt: 1}},
		{"invalid_country_code", ClickEventPayload{ShortCode: "abc", LinkID: "link", VisitorHash: "0123456789abcdef", CountryCode: "USA", ClickedAt: 1}},
		{"missing_clicked_at", ClickEventPayload{ShortCode: "abc", LinkID: "link", VisitorHash: "0123456789abcdef"}},
	}

	for _, tc := range cases {
		if err := ValidateClickEventPayload(tc.payload); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}
