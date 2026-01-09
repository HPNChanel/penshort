// Package analytics provides click event capture and processing.
package analytics

import "fmt"

const (
	minShortCodeLength = 3
	maxShortCodeLength = 50
	maxMetaLength      = 500
	visitorHashLength  = 16
)

// ValidateClickEventPayload validates click event payload fields.
func ValidateClickEventPayload(payload ClickEventPayload) error {
	if payload.ShortCode == "" {
		return fmt.Errorf("short_code is required")
	}
	if len(payload.ShortCode) < minShortCodeLength || len(payload.ShortCode) > maxShortCodeLength {
		return fmt.Errorf("short_code length out of bounds")
	}
	if payload.LinkID == "" {
		return fmt.Errorf("link_id is required")
	}
	if payload.VisitorHash == "" {
		return fmt.Errorf("visitor_hash is required")
	}
	if len(payload.VisitorHash) != visitorHashLength || !isHex(payload.VisitorHash) {
		return fmt.Errorf("visitor_hash must be %d hex chars", visitorHashLength)
	}
	if payload.CountryCode != "" && len(payload.CountryCode) != 2 {
		return fmt.Errorf("country_code must be 2 chars")
	}
	if payload.ClickedAt <= 0 {
		return fmt.Errorf("clicked_at must be set")
	}
	if len(payload.Referrer) > maxMetaLength {
		return fmt.Errorf("referrer too long")
	}
	if len(payload.UserAgent) > maxMetaLength {
		return fmt.Errorf("user_agent too long")
	}
	return nil
}

func isHex(value string) bool {
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			continue
		}
		return false
	}
	return true
}
