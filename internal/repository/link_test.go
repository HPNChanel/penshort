package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/testutil"
)

func TestRepository_CreateAndGetLink(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t, ctx)

	link := newTestLink()
	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("create link: %v", err)
	}

	byID, err := repo.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("get link by ID: %v", err)
	}
	assertLinkEqual(t, link, byID)

	byCode, err := repo.GetLinkByShortCode(ctx, link.ShortCode)
	if err != nil {
		t.Fatalf("get link by short code: %v", err)
	}
	assertLinkEqual(t, link, byCode)

	byCodeAlias, err := repo.GetLinkByCode(ctx, link.ShortCode)
	if err != nil {
		t.Fatalf("get link by code: %v", err)
	}
	assertLinkEqual(t, link, byCodeAlias)

	exists, err := repo.ShortCodeExists(ctx, link.ShortCode)
	if err != nil {
		t.Fatalf("short code exists: %v", err)
	}
	if !exists {
		t.Fatalf("expected short code to exist")
	}

	duplicate := newTestLink()
	duplicate.ShortCode = link.ShortCode
	if err := repo.CreateLink(ctx, duplicate); !errors.Is(err, ErrAliasExists) {
		t.Fatalf("expected ErrAliasExists, got %v", err)
	}
}

func TestRepository_UpdateLink(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t, ctx)

	link := newTestLink()
	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("create link: %v", err)
	}

	updatedDest := "https://example.com/updated"
	updatedRedirect := model.RedirectPermanent
	updatedEnabled := false
	updatedExpiry := time.Now().UTC().Add(48 * time.Hour)

	link.Destination = updatedDest
	link.RedirectType = updatedRedirect
	link.Enabled = updatedEnabled
	link.ExpiresAt = &updatedExpiry

	if err := repo.UpdateLink(ctx, link); err != nil {
		t.Fatalf("update link: %v", err)
	}

	loaded, err := repo.GetLinkByID(ctx, link.ID)
	if err != nil {
		t.Fatalf("get link by ID: %v", err)
	}

	if loaded.Destination != updatedDest {
		t.Fatalf("expected destination %q, got %q", updatedDest, loaded.Destination)
	}
	if loaded.RedirectType != updatedRedirect {
		t.Fatalf("expected redirect type %d, got %d", updatedRedirect, loaded.RedirectType)
	}
	if loaded.Enabled != updatedEnabled {
		t.Fatalf("expected enabled %v, got %v", updatedEnabled, loaded.Enabled)
	}
	if loaded.ExpiresAt == nil {
		t.Fatalf("expected expires_at to be set")
	}
}

func TestRepository_DeleteLink(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t, ctx)

	link := newTestLink()
	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("create link: %v", err)
	}

	if err := repo.DeleteLink(ctx, link.ID); err != nil {
		t.Fatalf("delete link: %v", err)
	}

	if _, err := repo.GetLinkByID(ctx, link.ID); !errors.Is(err, ErrLinkNotFound) {
		t.Fatalf("expected ErrLinkNotFound, got %v", err)
	}
}

func newTestRepository(t *testing.T, ctx context.Context) *Repository {
	t.Helper()

	dbURL := testutil.RequireEnv(t, "DATABASE_URL")
	repo, err := New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create repository: %v", err)
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
		t.Fatalf("reset schema: %v", err)
	}

	return repo
}

func newTestLink() *model.Link {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)

	return &model.Link{
		ID:           fmt.Sprintf("test-%d", now.UnixNano()),
		ShortCode:    fmt.Sprintf("code-%d", now.UnixNano()),
		Destination:  "https://example.com",
		RedirectType: model.RedirectTemporary,
		OwnerID:      "system",
		Enabled:      true,
		ExpiresAt:    &expires,
		ClickCount:   0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func assertLinkEqual(t *testing.T, expected, actual *model.Link) {
	t.Helper()

	if expected.ShortCode != actual.ShortCode {
		t.Fatalf("short_code mismatch: %q vs %q", expected.ShortCode, actual.ShortCode)
	}
	if expected.Destination != actual.Destination {
		t.Fatalf("destination mismatch: %q vs %q", expected.Destination, actual.Destination)
	}
	if expected.RedirectType != actual.RedirectType {
		t.Fatalf("redirect_type mismatch: %d vs %d", expected.RedirectType, actual.RedirectType)
	}
	if expected.Enabled != actual.Enabled {
		t.Fatalf("enabled mismatch: %v vs %v", expected.Enabled, actual.Enabled)
	}
	if expected.ExpiresAt == nil || actual.ExpiresAt == nil {
		t.Fatalf("expected expires_at to be set")
	}
	if expected.ExpiresAt != nil && actual.ExpiresAt != nil {
		diff := actual.ExpiresAt.Sub(*expected.ExpiresAt)
		if diff > time.Second || diff < -time.Second {
			t.Fatalf("expires_at mismatch: %v vs %v", expected.ExpiresAt, actual.ExpiresAt)
		}
	}
}
