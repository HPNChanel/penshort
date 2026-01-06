package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateDestination(t *testing.T) {
	svc := &LinkService{}

	longDest := "https://example.com/" + strings.Repeat("a", maxDestinationLength)

	tests := []struct {
		name    string
		dest    string
		wantErr error
	}{
		{"empty", "", ErrInvalidDestination},
		{"invalid_scheme", "ftp://example.com", ErrInvalidDestination},
		{"missing_host", "https://", ErrInvalidDestination},
		{"too_long", longDest, ErrURLTooLong},
		{"valid", "https://example.com/path", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := svc.validateDestination(test.dest)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestCreateLinkValidationErrors(t *testing.T) {
	svc := &LinkService{}

	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)

	tests := []struct {
		name    string
		input   CreateLinkInput
		wantErr error
	}{
		{
			name: "invalid_alias",
			input: CreateLinkInput{
				Destination: "https://example.com",
				Alias:       "!!",
			},
			wantErr: ErrInvalidAlias,
		},
		{
			name: "invalid_redirect_type",
			input: CreateLinkInput{
				Destination:  "https://example.com",
				Alias:        "valid-alias",
				RedirectType: 307,
			},
			wantErr: ErrInvalidRedirectType,
		},
		{
			name: "expires_in_past",
			input: CreateLinkInput{
				Destination: "https://example.com",
				Alias:       "valid-alias",
				ExpiresAt:   &past,
			},
			wantErr: ErrExpiresInPast,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := svc.CreateLink(context.Background(), test.input)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}
