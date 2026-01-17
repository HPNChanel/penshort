//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/testutil"
)

// ============================================================================
// Link Repository Integration Tests
// ============================================================================

func TestIntegrationLinkRepository_CreateLink(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	shortCode := testutil.UniqueShortCode("create")
	link := testutil.NewTestLink(t, shortCode)

	err := repo.CreateLink(ctx, link)
	if err != nil {
		t.Fatalf("CreateLink failed: %v", err)
	}

	// Verify link exists in DB
	retrieved, err := repo.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("GetLinkByID failed: %v", err)
	}

	if retrieved.ShortCode != shortCode {
		t.Errorf("ShortCode mismatch: got %q, want %q", retrieved.ShortCode, shortCode)
	}
	if retrieved.Destination != link.Destination {
		t.Errorf("Destination mismatch: got %q, want %q", retrieved.Destination, link.Destination)
	}
	if retrieved.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestIntegrationLinkRepository_CreateLink_DuplicateAlias(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	shortCode := testutil.UniqueShortCode("dup")
	link1 := testutil.NewTestLink(t, shortCode)
	link2 := testutil.NewTestLink(t, shortCode)
	link2.ID = testutil.UniqueID("link") // Different ID, same short_code

	if err := repo.CreateLink(ctx, link1); err != nil {
		t.Fatalf("CreateLink (first) failed: %v", err)
	}

	err := repo.CreateLink(ctx, link2)
	if !errors.Is(err, ErrAliasExists) {
		t.Errorf("Expected ErrAliasExists, got: %v", err)
	}
}

func TestIntegrationLinkRepository_GetByID(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	shortCode := testutil.UniqueShortCode("getid")
	link := testutil.NewTestLink(t, shortCode)

	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink failed: %v", err)
	}

	retrieved, err := repo.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("GetLinkByID failed: %v", err)
	}

	if retrieved.ID != link.ID {
		t.Errorf("ID mismatch: got %q, want %q", retrieved.ID, link.ID)
	}
}

func TestIntegrationLinkRepository_GetByID_NotFound(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	_, err := repo.GetLinkByID(ctx, "nonexistent-id")
	if !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("Expected ErrLinkNotFound, got: %v", err)
	}
}

func TestIntegrationLinkRepository_GetByShortCode(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	shortCode := testutil.UniqueShortCode("getcode")
	link := testutil.NewTestLink(t, shortCode)

	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink failed: %v", err)
	}

	retrieved, err := repo.GetLinkByShortCode(ctx, shortCode)
	if err != nil {
		t.Fatalf("GetLinkByShortCode failed: %v", err)
	}

	if retrieved.ShortCode != shortCode {
		t.Errorf("ShortCode mismatch: got %q, want %q", retrieved.ShortCode, shortCode)
	}
}

func TestIntegrationLinkRepository_GetByShortCode_NotFound(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	_, err := repo.GetLinkByShortCode(ctx, "nonexistent-code")
	if !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("Expected ErrLinkNotFound, got: %v", err)
	}
}

func TestIntegrationLinkRepository_UpdateLink(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	shortCode := testutil.UniqueShortCode("update")
	link := testutil.NewTestLink(t, shortCode)

	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink failed: %v", err)
	}

	// Update destination
	newDestination := "https://updated.example.com/new-path"
	link.Destination = newDestination

	if err := repo.UpdateLink(ctx, link); err != nil {
		t.Fatalf("UpdateLink failed: %v", err)
	}

	retrieved, err := repo.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("GetLinkByID failed: %v", err)
	}

	if retrieved.Destination != newDestination {
		t.Errorf("Destination not updated: got %q, want %q", retrieved.Destination, newDestination)
	}
	if !retrieved.UpdatedAt.After(link.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

func TestIntegrationLinkRepository_DeleteLink_SoftDelete(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	shortCode := testutil.UniqueShortCode("delete")
	link := testutil.NewTestLink(t, shortCode)

	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink failed: %v", err)
	}

	if err := repo.DeleteLink(ctx, link.ID); err != nil {
		t.Fatalf("DeleteLink failed: %v", err)
	}

	// Link should not be found by short code (soft deleted)
	_, err := repo.GetLinkByShortCode(ctx, shortCode)
	if !errors.Is(err, ErrLinkNotFound) {
		t.Errorf("Expected ErrLinkNotFound after soft delete, got: %v", err)
	}

	// But can still be retrieved by ID (for admin purposes)
	retrieved, err := repo.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("GetLinkByID failed after soft delete: %v", err)
	}
	if retrieved.DeletedAt == nil {
		t.Error("DeletedAt should be set after soft delete")
	}
}

func TestIntegrationLinkRepository_ListLinks_Pagination(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	// Create 5 links
	ownerID := "pagination-test-user"
	for i := 0; i < 5; i++ {
		shortCode := testutil.UniqueShortCode("page")
		link := testutil.NewTestLink(t, shortCode)
		link.OwnerID = ownerID
		if err := repo.CreateLink(ctx, link); err != nil {
			t.Fatalf("CreateLink failed: %v", err)
		}
		time.Sleep(1 * time.Millisecond) // Ensure different created_at
	}

	// Fetch first page
	filter := LinkFilter{OwnerID: ownerID}
	links, nextCursor, err := repo.ListLinks(ctx, filter, "", 2)
	if err != nil {
		t.Fatalf("ListLinks failed: %v", err)
	}

	if len(links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(links))
	}
	if nextCursor == "" {
		t.Error("Expected nextCursor for more pages")
	}

	// Fetch second page
	links2, nextCursor2, err := repo.ListLinks(ctx, filter, nextCursor, 2)
	if err != nil {
		t.Fatalf("ListLinks (page 2) failed: %v", err)
	}

	if len(links2) != 2 {
		t.Errorf("Expected 2 links on page 2, got %d", len(links2))
	}

	// IDs should not overlap
	for _, l1 := range links {
		for _, l2 := range links2 {
			if l1.ID == l2.ID {
				t.Errorf("Duplicate link ID across pages: %s", l1.ID)
			}
		}
	}

	// Fetch third page (should have 1 link)
	links3, _, err := repo.ListLinks(ctx, filter, nextCursor2, 2)
	if err != nil {
		t.Fatalf("ListLinks (page 3) failed: %v", err)
	}

	if len(links3) != 1 {
		t.Errorf("Expected 1 link on page 3, got %d", len(links3))
	}
}

func TestIntegrationLinkRepository_IncrementClickCount(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	shortCode := testutil.UniqueShortCode("clicks")
	link := testutil.NewTestLink(t, shortCode)

	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink failed: %v", err)
	}

	// Increment by 5
	if err := repo.IncrementClickCount(ctx, link.ID, 5); err != nil {
		t.Fatalf("IncrementClickCount failed: %v", err)
	}

	retrieved, err := repo.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("GetLinkByID failed: %v", err)
	}

	if retrieved.ClickCount != 5 {
		t.Errorf("ClickCount mismatch: got %d, want 5", retrieved.ClickCount)
	}

	// Increment again by 3
	if err := repo.IncrementClickCount(ctx, link.ID, 3); err != nil {
		t.Fatalf("IncrementClickCount (2) failed: %v", err)
	}

	retrieved2, _ := repo.GetLinkByID(ctx, link.ID)
	if retrieved2.ClickCount != 8 {
		t.Errorf("ClickCount after second increment: got %d, want 8", retrieved2.ClickCount)
	}
}

func TestIntegrationLinkRepository_ShortCodeExists(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	shortCode := testutil.UniqueShortCode("exists")
	link := testutil.NewTestLink(t, shortCode)

	// Before creation
	exists, err := repo.ShortCodeExists(ctx, shortCode)
	if err != nil {
		t.Fatalf("ShortCodeExists failed: %v", err)
	}
	if exists {
		t.Error("ShortCode should not exist before creation")
	}

	// After creation
	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink failed: %v", err)
	}

	exists, err = repo.ShortCodeExists(ctx, shortCode)
	if err != nil {
		t.Fatalf("ShortCodeExists (after create) failed: %v", err)
	}
	if !exists {
		t.Error("ShortCode should exist after creation")
	}

	// After soft delete
	if err := repo.DeleteLink(ctx, link.ID); err != nil {
		t.Fatalf("DeleteLink failed: %v", err)
	}

	exists, err = repo.ShortCodeExists(ctx, shortCode)
	if err != nil {
		t.Fatalf("ShortCodeExists (after delete) failed: %v", err)
	}
	if exists {
		t.Error("ShortCode should not exist after soft delete")
	}
}

func TestIntegrationLinkRepository_ExpiredLinkBehavior(t *testing.T) {
	ctx, repo := newLinkTestEnv(t)

	// Create expired link
	shortCode := testutil.UniqueShortCode("expired")
	expiredAt := time.Now().Add(-1 * time.Hour)
	link := testutil.NewTestLinkWithExpiry(t, shortCode, expiredAt)

	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("CreateLink failed: %v", err)
	}

	// Link can still be retrieved
	retrieved, err := repo.GetLinkByShortCode(ctx, shortCode)
	if err != nil {
		t.Fatalf("GetLinkByShortCode failed: %v", err)
	}

	// But it should be expired
	if !retrieved.IsExpired() {
		t.Error("Link should be expired")
	}
	if retrieved.Status() != model.LinkStatusExpired {
		t.Errorf("Expected status %q, got %q", model.LinkStatusExpired, retrieved.Status())
	}
}

// ============================================================================
// Test Environment Setup
// ============================================================================

func newLinkTestEnv(t *testing.T) (context.Context, *Repository) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	ctx := context.Background()
	dbURL := testutil.RequireEnv(t, "DATABASE_URL")

	repo, err := New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(repo.Close)

	unlock, err := testutil.AcquireDBLock(ctx, repo.Pool())
	if err != nil {
		t.Fatalf("acquire db lock: %v", err)
	}
	t.Cleanup(func() {
		_ = unlock()
	})

	if err := testutil.ResetLinksSchema(ctx, repo.Pool()); err != nil {
		t.Fatalf("reset links schema: %v", err)
	}

	return ctx, repo
}
