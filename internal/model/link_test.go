package model

import (
	"testing"
	"time"
)

func TestLink_ToCachedLink_Basic(t *testing.T) {
	t.Parallel()

	now := time.Now()
	link := &Link{
		ID:           "link-123",
		ShortCode:    "abc123",
		Destination:  "https://example.com",
		RedirectType: RedirectPermanent,
		OwnerID:      "user-1",
		Enabled:      true,
		UpdatedAt:    now,
	}

	cached := link.ToCachedLink()

	if cached.Destination != "https://example.com" {
		t.Errorf("Destination = %s, want https://example.com", cached.Destination)
	}
	if cached.RedirectType != "301" {
		t.Errorf("RedirectType = %s, want 301", cached.RedirectType)
	}
	if cached.Enabled != "1" {
		t.Errorf("Enabled = %s, want 1", cached.Enabled)
	}
	if cached.ExpiresAt != "" {
		t.Errorf("ExpiresAt should be empty, got %s", cached.ExpiresAt)
	}
	if cached.DeletedAt != "" {
		t.Errorf("DeletedAt should be empty, got %s", cached.DeletedAt)
	}
}

func TestLink_ToCachedLink_Disabled(t *testing.T) {
	t.Parallel()

	link := &Link{
		Destination:  "https://example.com",
		RedirectType: RedirectTemporary,
		Enabled:      false,
		UpdatedAt:    time.Now(),
	}

	cached := link.ToCachedLink()

	if cached.Enabled != "0" {
		t.Errorf("Enabled = %s, want 0", cached.Enabled)
	}
	if cached.RedirectType != "302" {
		t.Errorf("RedirectType = %s, want 302", cached.RedirectType)
	}
}

func TestLink_ToCachedLink_WithExpiry(t *testing.T) {
	t.Parallel()

	expiresAt := time.Unix(1700000000, 0)
	link := &Link{
		Destination:  "https://example.com",
		RedirectType: RedirectPermanent,
		Enabled:      true,
		ExpiresAt:    &expiresAt,
		UpdatedAt:    time.Now(),
	}

	cached := link.ToCachedLink()

	if cached.ExpiresAt != "1700000000" {
		t.Errorf("ExpiresAt = %s, want 1700000000", cached.ExpiresAt)
	}
}

func TestLink_ToCachedLink_NoExpiry(t *testing.T) {
	t.Parallel()

	link := &Link{
		Destination:  "https://example.com",
		RedirectType: RedirectPermanent,
		Enabled:      true,
		ExpiresAt:    nil,
		UpdatedAt:    time.Now(),
	}

	cached := link.ToCachedLink()

	if cached.ExpiresAt != "" {
		t.Errorf("ExpiresAt should be empty for nil expiry, got %s", cached.ExpiresAt)
	}
}

func TestLink_ToCachedLink_Deleted(t *testing.T) {
	t.Parallel()

	deletedAt := time.Unix(1700000000, 0)
	link := &Link{
		Destination:  "https://example.com",
		RedirectType: RedirectPermanent,
		Enabled:      true,
		DeletedAt:    &deletedAt,
		UpdatedAt:    time.Now(),
	}

	cached := link.ToCachedLink()

	if cached.DeletedAt != "1700000000" {
		t.Errorf("DeletedAt = %s, want 1700000000", cached.DeletedAt)
	}
}

func TestCachedLink_ToLink_Basic(t *testing.T) {
	t.Parallel()

	cached := &CachedLink{
		Destination:  "https://example.com",
		RedirectType: "301",
		Enabled:      "1",
		UpdatedAt:    "1700000000",
	}

	link := cached.ToLink("abc123")

	if link.ShortCode != "abc123" {
		t.Errorf("ShortCode = %s, want abc123", link.ShortCode)
	}
	if link.Destination != "https://example.com" {
		t.Errorf("Destination = %s, want https://example.com", link.Destination)
	}
	if link.RedirectType != RedirectPermanent {
		t.Errorf("RedirectType = %d, want %d", link.RedirectType, RedirectPermanent)
	}
	if !link.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestCachedLink_ToLink_302(t *testing.T) {
	t.Parallel()

	cached := &CachedLink{
		Destination:  "https://example.com",
		RedirectType: "302",
		Enabled:      "1",
		UpdatedAt:    "1700000000",
	}

	link := cached.ToLink("abc123")

	if link.RedirectType != RedirectTemporary {
		t.Errorf("RedirectType = %d, want %d", link.RedirectType, RedirectTemporary)
	}
}

func TestCachedLink_ToLink_Invalid302Default(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		redirectType string
	}{
		{"invalid string", "invalid"},
		{"empty", ""},
		{"wrong number", "307"},
		{"text", "permanent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cached := &CachedLink{
				Destination:  "https://example.com",
				RedirectType: tt.redirectType,
				Enabled:      "1",
				UpdatedAt:    "1700000000",
			}

			link := cached.ToLink("abc123")

			// Invalid redirect type should default to 302
			if link.RedirectType != RedirectTemporary {
				t.Errorf("RedirectType = %d, want %d (default 302)", link.RedirectType, RedirectTemporary)
			}
		})
	}
}

func TestCachedLink_ToLink_ParsesTimestamps(t *testing.T) {
	t.Parallel()

	cached := &CachedLink{
		Destination:  "https://example.com",
		RedirectType: "301",
		Enabled:      "1",
		ExpiresAt:    "1700000000",
		DeletedAt:    "1700000001",
		UpdatedAt:    "1700000002",
	}

	link := cached.ToLink("abc123")

	// Check ExpiresAt
	if link.ExpiresAt == nil {
		t.Fatal("ExpiresAt should not be nil")
	}
	if link.ExpiresAt.Unix() != 1700000000 {
		t.Errorf("ExpiresAt Unix = %d, want 1700000000", link.ExpiresAt.Unix())
	}

	// Check DeletedAt
	if link.DeletedAt == nil {
		t.Fatal("DeletedAt should not be nil")
	}
	if link.DeletedAt.Unix() != 1700000001 {
		t.Errorf("DeletedAt Unix = %d, want 1700000001", link.DeletedAt.Unix())
	}

	// Check UpdatedAt
	if link.UpdatedAt.Unix() != 1700000002 {
		t.Errorf("UpdatedAt Unix = %d, want 1700000002", link.UpdatedAt.Unix())
	}
}

func TestCachedLink_ToLink_InvalidTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expiresAt string
		deletedAt string
	}{
		{"invalid expiresAt", "invalid", ""},
		{"invalid deletedAt", "", "invalid"},
		{"both invalid", "not-a-number", "also-not-a-number"},
		{"empty both", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cached := &CachedLink{
				Destination:  "https://example.com",
				RedirectType: "301",
				Enabled:      "1",
				ExpiresAt:    tt.expiresAt,
				DeletedAt:    tt.deletedAt,
				UpdatedAt:    "1700000000",
			}

			// Should not panic
			link := cached.ToLink("abc123")

			// Invalid timestamps should result in nil times
			if tt.expiresAt != "" && link.ExpiresAt != nil {
				// If expiresAt was invalid, it should be nil
				if tt.expiresAt == "invalid" || tt.expiresAt == "not-a-number" {
					t.Errorf("ExpiresAt should be nil for invalid timestamp")
				}
			}
		})
	}
}

func TestLink_Status(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name   string
		link   Link
		want   LinkStatus
	}{
		{
			name:   "active - enabled, no expiry",
			link:   Link{Enabled: true, ExpiresAt: nil, DeletedAt: nil},
			want:   LinkStatusActive,
		},
		{
			name:   "active - enabled, future expiry",
			link:   Link{Enabled: true, ExpiresAt: &future, DeletedAt: nil},
			want:   LinkStatusActive,
		},
		{
			name:   "disabled",
			link:   Link{Enabled: false, ExpiresAt: nil, DeletedAt: nil},
			want:   LinkStatusDisabled,
		},
		{
			name:   "expired",
			link:   Link{Enabled: true, ExpiresAt: &past, DeletedAt: nil},
			want:   LinkStatusExpired,
		},
		{
			name:   "deleted",
			link:   Link{Enabled: true, ExpiresAt: nil, DeletedAt: &now},
			want:   LinkStatusDeleted,
		},
		{
			name:   "deleted takes precedence over disabled",
			link:   Link{Enabled: false, ExpiresAt: nil, DeletedAt: &now},
			want:   LinkStatusDeleted,
		},
		{
			name:   "deleted takes precedence over expired",
			link:   Link{Enabled: true, ExpiresAt: &past, DeletedAt: &now},
			want:   LinkStatusDeleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.link.Status()
			if got != tt.want {
				t.Errorf("Status() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLink_IsActive(t *testing.T) {
	t.Parallel()

	future := time.Now().Add(time.Hour)

	activeLink := Link{Enabled: true, ExpiresAt: &future}
	disabledLink := Link{Enabled: false}

	if !activeLink.IsActive() {
		t.Error("Expected active link to return true")
	}
	if disabledLink.IsActive() {
		t.Error("Expected disabled link to return false")
	}
}

func TestLink_IsExpired(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"nil expiry", nil, false},
		{"future expiry", &future, false},
		{"past expiry", &past, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			link := Link{ExpiresAt: tt.expiresAt}
			if got := link.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedirectType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		redirectType RedirectType
		want         bool
	}{
		{RedirectPermanent, true},
		{RedirectTemporary, true},
		{RedirectType(307), false},
		{RedirectType(0), false},
		{RedirectType(200), false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			if got := tt.redirectType.IsValid(); got != tt.want {
				t.Errorf("RedirectType(%d).IsValid() = %v, want %v", tt.redirectType, got, tt.want)
			}
		})
	}
}
