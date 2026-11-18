//go:build integration

package notes

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPostgresRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	store, err := NewPostgres(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	created, err := store.Create(ctx, "integration-note", "первая версия")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_ = store.Delete(cleanupCtx, created.ID)
	})

	updated, err := store.Update(ctx, created.ID, "integration-note", "вторая версия")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Body != "вторая версия" || updated.UpdatedAt.Before(created.UpdatedAt) {
		t.Fatalf("update did not apply: %+v", updated)
	}

	items, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("created note is missing from the list")
	}

	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(ctx, created.ID, "integration-note", "gone"); err != ErrNotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}
