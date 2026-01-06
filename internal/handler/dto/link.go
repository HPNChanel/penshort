// Package dto provides Data Transfer Objects for API requests and responses.
package dto

import (
	"time"

	"github.com/penshort/penshort/internal/model"
)

// CreateLinkRequest represents the request body for creating a link.
type CreateLinkRequest struct {
	Destination  string     `json:"destination"`
	Alias        string     `json:"alias,omitempty"`
	RedirectType int        `json:"redirect_type,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// UpdateLinkRequest represents the request body for updating a link.
type UpdateLinkRequest struct {
	Destination  *string    `json:"destination,omitempty"`
	RedirectType *int       `json:"redirect_type,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Enabled      *bool      `json:"enabled,omitempty"`
}

// LinkResponse represents a link in API responses.
type LinkResponse struct {
	ID           string     `json:"id"`
	ShortCode    string     `json:"short_code"`
	ShortURL     string     `json:"short_url"`
	Destination  string     `json:"destination"`
	RedirectType int        `json:"redirect_type"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Status       string     `json:"status"`
	ClickCount   int64      `json:"click_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// LinkListResponse represents a paginated list of links.
type LinkListResponse struct {
	Data       []LinkResponse `json:"data"`
	Pagination *Pagination    `json:"pagination"`
}

// Pagination provides cursor-based pagination info.
type Pagination struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// ListLinksQuery represents query parameters for listing links.
type ListLinksQuery struct {
	Cursor        string
	Limit         int
	Status        string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// ToLinkResponse converts a Link model to LinkResponse DTO.
func ToLinkResponse(link *model.Link, baseURL string) *LinkResponse {
	return &LinkResponse{
		ID:           link.ID,
		ShortCode:    link.ShortCode,
		ShortURL:     baseURL + "/" + link.ShortCode,
		Destination:  link.Destination,
		RedirectType: int(link.RedirectType),
		ExpiresAt:    link.ExpiresAt,
		Status:       string(link.Status()),
		ClickCount:   link.ClickCount,
		CreatedAt:    link.CreatedAt,
		UpdatedAt:    link.UpdatedAt,
	}
}

// ToLinkListResponse converts a slice of Link models to LinkListResponse.
func ToLinkListResponse(links []*model.Link, baseURL string, nextCursor string, hasMore bool) *LinkListResponse {
	responses := make([]LinkResponse, len(links))
	for i, link := range links {
		responses[i] = *ToLinkResponse(link, baseURL)
	}
	return &LinkListResponse{
		Data: responses,
		Pagination: &Pagination{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}
}
