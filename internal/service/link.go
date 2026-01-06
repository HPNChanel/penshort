// Package service provides business logic for the application.
package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/penshort/penshort/internal/cache"
	"github.com/penshort/penshort/internal/metrics"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/repository"
)

// Service errors.
var (
	ErrInvalidDestination = errors.New("invalid destination URL")
	ErrInvalidAlias       = errors.New("invalid alias format")
	ErrAliasExists        = errors.New("alias already exists")
	ErrLinkNotFound       = errors.New("link not found")
	ErrLinkExpired        = errors.New("link is expired")
	ErrLinkDisabled       = errors.New("link is disabled")
	ErrExpiresInPast      = errors.New("expires_at must be in the future")
	ErrInvalidRedirectType = errors.New("invalid redirect type")
	ErrURLTooLong         = errors.New("destination URL too long")
)

// Alias validation regex: 3-50 chars, alphanumeric + hyphen.
var aliasRegex = regexp.MustCompile(`^[a-zA-Z0-9-]{3,50}$`)

const (
	maxDestinationLength = 2048
	aliasLength          = 7
	aliasAlphabet        = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	maxAliasRetries      = 3
)

// LinkService handles link business logic.
type LinkService struct {
	repo    *repository.Repository
	cache   *cache.Cache
	baseURL string
	metrics metrics.Recorder
}

// NewLinkService creates a new LinkService.
func NewLinkService(repo *repository.Repository, cache *cache.Cache, baseURL string, recorder metrics.Recorder) *LinkService {
	if recorder == nil {
		recorder = metrics.NewNoop()
	}
	return &LinkService{
		repo:    repo,
		cache:   cache,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		metrics: recorder,
	}
}

// CreateLinkInput defines input for creating a link.
type CreateLinkInput struct {
	Destination  string
	Alias        string
	RedirectType int
	ExpiresAt    *time.Time
	OwnerID      string
}

// CreateLink creates a new short link.
func (s *LinkService) CreateLink(ctx context.Context, input CreateLinkInput) (*model.Link, error) {
	// Validate destination URL
	if err := s.validateDestination(input.Destination); err != nil {
		return nil, err
	}

	// Validate and normalize redirect type
	redirectType := model.RedirectTemporary // Default 302
	if input.RedirectType != 0 {
		redirectType = model.RedirectType(input.RedirectType)
		if !redirectType.IsValid() {
			return nil, ErrInvalidRedirectType
		}
	}

	// Validate expiry
	if input.ExpiresAt != nil && input.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpiresInPast
	}

	// Handle alias
	alias := input.Alias
	if alias != "" {
		// Custom alias: validate format
		if !aliasRegex.MatchString(alias) {
			return nil, ErrInvalidAlias
		}
	} else {
		// Auto-generate alias
		var err error
		alias, err = s.generateUniqueAlias(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate alias: %w", err)
		}
	}

	// Set default owner if not provided
	ownerID := input.OwnerID
	if ownerID == "" {
		ownerID = "system" // Phase 2 default
	}

	// Create link model
	link := &model.Link{
		ID:           generateULID(),
		ShortCode:    alias,
		Destination:  input.Destination,
		RedirectType: redirectType,
		OwnerID:      ownerID,
		Enabled:      true,
		ExpiresAt:    input.ExpiresAt,
		ClickCount:   0,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Insert into database
	if err := s.repo.CreateLink(ctx, link); err != nil {
		if errors.Is(err, repository.ErrAliasExists) {
			return nil, ErrAliasExists
		}
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	s.metrics.IncLinkCreated()

	return link, nil
}

// GetLink retrieves a link by ID.
func (s *LinkService) GetLink(ctx context.Context, id string) (*model.Link, error) {
	link, err := s.repo.GetLinkByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrLinkNotFound) {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}

	return link, nil
}

// ListLinksInput defines input for listing links.
type ListLinksInput struct {
	OwnerID       string
	Cursor        string
	Limit         int
	Status        string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// ListLinksOutput defines output for listing links.
type ListLinksOutput struct {
	Links      []*model.Link
	NextCursor string
	HasMore    bool
}

// ListLinks retrieves a paginated list of links.
func (s *LinkService) ListLinks(ctx context.Context, input ListLinksInput) (*ListLinksOutput, error) {
	// Set defaults
	if input.Limit <= 0 || input.Limit > 100 {
		input.Limit = 20
	}
	if input.OwnerID == "" {
		input.OwnerID = "system"
	}

	filter := repository.LinkFilter{
		OwnerID:       input.OwnerID,
		CreatedAfter:  input.CreatedAfter,
		CreatedBefore: input.CreatedBefore,
	}

	links, nextCursor, err := s.repo.ListLinks(ctx, filter, input.Cursor, input.Limit)
	if err != nil {
		return nil, err
	}

	// Filter by computed status if specified
	if input.Status != "" {
		filtered := make([]*model.Link, 0, len(links))
		targetStatus := model.LinkStatus(input.Status)
		for _, link := range links {
			if link.Status() == targetStatus {
				filtered = append(filtered, link)
			}
		}
		links = filtered
	}

	return &ListLinksOutput{
		Links:      links,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}, nil
}

// UpdateLinkInput defines input for updating a link.
type UpdateLinkInput struct {
	ID           string
	Destination  *string
	RedirectType *int
	ExpiresAt    *time.Time
	Enabled      *bool
	ClearExpiry  bool // If true, set expires_at to nil
}

// UpdateLink updates a link's mutable fields.
func (s *LinkService) UpdateLink(ctx context.Context, input UpdateLinkInput) (*model.Link, error) {
	// Get existing link
	link, err := s.repo.GetLinkByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, repository.ErrLinkNotFound) {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}

	// Check if expired
	if link.IsExpired() {
		return nil, ErrLinkExpired
	}

	// Apply updates
	if input.Destination != nil {
		if err := s.validateDestination(*input.Destination); err != nil {
			return nil, err
		}
		link.Destination = *input.Destination
	}

	if input.RedirectType != nil {
		redirectType := model.RedirectType(*input.RedirectType)
		if !redirectType.IsValid() {
			return nil, ErrInvalidRedirectType
		}
		link.RedirectType = redirectType
	}

	if input.ClearExpiry {
		link.ExpiresAt = nil
	} else if input.ExpiresAt != nil {
		if input.ExpiresAt.Before(time.Now()) {
			return nil, ErrExpiresInPast
		}
		link.ExpiresAt = input.ExpiresAt
	}

	if input.Enabled != nil {
		link.Enabled = *input.Enabled
	}

	// Update in database
	if err := s.repo.UpdateLink(ctx, link); err != nil {
		return nil, err
	}

	s.metrics.IncLinkUpdated()

	// Invalidate cache
	if err := s.cache.DeleteLink(ctx, link.ShortCode); err != nil {
		// Log but don't fail - eventual consistency is acceptable
		_ = err
	}

	return link, nil
}

// DeleteLink soft-deletes a link.
func (s *LinkService) DeleteLink(ctx context.Context, id string) error {
	// Get link first to get short code for cache invalidation
	link, err := s.repo.GetLinkByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrLinkNotFound) {
			return ErrLinkNotFound
		}
		return err
	}

	// Soft delete in database
	if err := s.repo.DeleteLink(ctx, id); err != nil {
		return err
	}

	s.metrics.IncLinkDeleted()

	// Invalidate cache
	if err := s.cache.DeleteLink(ctx, link.ShortCode); err != nil {
		_ = err // Log but don't fail
	}

	return nil
}

// ResolveRedirect resolves a short code to its destination for redirect.
// This is the hot path - optimized for speed with cache-first lookup.
func (s *LinkService) ResolveRedirect(ctx context.Context, shortCode string) (*model.Link, bool, error) {
	start := time.Now()
	defer func() {
		s.metrics.ObserveRedirectDuration(time.Since(start))
	}()

	cacheHit := false

	// Step 1: Try cache
	cached, err := s.cache.GetLink(ctx, shortCode)
	if err == nil {
		// Cache hit - validate and return
		cacheHit = true
		s.metrics.IncRedirectCacheHit()
		link := cached.ToLink(shortCode)
		validated, err := s.validateRedirectLink(ctx, link, shortCode)
		return validated, cacheHit, err
	}

	// Step 2: Check negative cache
	if !errors.Is(err, cache.ErrCacheMiss) {
		// Redis error - fall through to DB
		// In production, log this error
	} else {
		s.metrics.IncRedirectCacheMiss()
		// Check negative cache
		isNegative, _ := s.cache.IsNegativelyCached(ctx, shortCode)
		if isNegative {
			return nil, cacheHit, ErrLinkNotFound
		}
	}

	// Step 3: DB lookup
	link, err := s.repo.GetLinkByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, repository.ErrLinkNotFound) {
			// Set negative cache
			_ = s.cache.SetNegativeCache(ctx, shortCode)
			return nil, cacheHit, ErrLinkNotFound
		}
		return nil, cacheHit, err
	}

	// Step 4: Backfill cache
	if err := s.cache.SetLink(ctx, shortCode, link); err != nil {
		// Log but don't fail
		_ = err
	}

	// Step 5: Validate and return
	validated, err := s.validateRedirectLink(ctx, link, shortCode)
	return validated, cacheHit, err
}

// IncrementClickAsync increments click counter asynchronously.
func (s *LinkService) IncrementClickAsync(ctx context.Context, shortCode string) {
	// Fire and forget - don't block redirect
	go func() {
		_ = s.cache.IncrementClicks(context.Background(), shortCode)
	}()
}

// BaseURL returns the configured base URL.
func (s *LinkService) BaseURL() string {
	return s.baseURL
}

// validateRedirectLink validates a link for redirect and handles cleanup.
func (s *LinkService) validateRedirectLink(ctx context.Context, link *model.Link, shortCode string) (*model.Link, error) {
	// Check deleted
	if link.DeletedAt != nil {
		return nil, ErrLinkNotFound
	}

	// Check disabled
	if !link.Enabled {
		return nil, ErrLinkDisabled
	}

	// Check expired
	if link.IsExpired() {
		// Evict from cache
		_ = s.cache.DeleteLink(ctx, shortCode)
		return nil, ErrLinkExpired
	}

	return link, nil
}

// validateDestination validates a destination URL.
func (s *LinkService) validateDestination(dest string) error {
	if dest == "" {
		return ErrInvalidDestination
	}

	if len(dest) > maxDestinationLength {
		return ErrURLTooLong
	}

	parsed, err := url.Parse(dest)
	if err != nil {
		return ErrInvalidDestination
	}

	// Only allow http and https schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidDestination
	}

	// Must have a host
	if parsed.Host == "" {
		return ErrInvalidDestination
	}

	return nil
}

// generateUniqueAlias generates a unique alias with collision retry.
func (s *LinkService) generateUniqueAlias(ctx context.Context) (string, error) {
	for i := 0; i < maxAliasRetries; i++ {
		alias := generateRandomAlias()
		exists, err := s.repo.ShortCodeExists(ctx, alias)
		if err != nil {
			return "", err
		}
		if !exists {
			return alias, nil
		}
	}
	return "", errors.New("failed to generate unique alias after retries")
}

// generateRandomAlias generates a random alias using crypto/rand.
func generateRandomAlias() string {
	b := make([]byte, aliasLength)
	for i := range b {
		idx, err := cryptoRandInt(len(aliasAlphabet))
		if err != nil {
			// Fallback (should never happen in practice)
			idx = 0
		}
		b[i] = aliasAlphabet[idx]
	}
	return string(b)
}

// generateULID generates a unique ID for links.
// Uses timestamp + random suffix for uniqueness.
func generateULID() string {
	timestamp := time.Now().UnixNano()
	randomPart := generateRandomAlias()
	return fmt.Sprintf("%016x%s", timestamp, randomPart)
}

// cryptoRandInt returns a cryptographically secure random integer in [0, max).
func cryptoRandInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

