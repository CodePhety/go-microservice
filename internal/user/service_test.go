package user

import (
	"context"
	"log"
	"testing"

	"user-service/internal/metrics"
)

func TestServiceCRUDFlow(t *testing.T) {
	repo := NewInMemoryRepository(log.New(nil, "", 0))
	service := NewService(repo, log.New(nil, "", 0), metrics.New())

	created, err := service.Create(context.Background(), CreateUserRequest{
		Name:   "Alice Example",
		Email:  "alice@example.com",
		Active: true,
	})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	if created.ID == "" {
		t.Fatal("created user should have an id")
	}

	allUsers, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(allUsers) != 1 {
		t.Fatalf("expected 1 user, got %d", len(allUsers))
	}

	got, err := service.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %s", got.Email)
	}

	updated, err := service.Update(context.Background(), created.ID, UpdateUserRequest{
		Name:   "Alice Smith",
		Email:  "alice.smith@example.com",
		Active: false,
	})
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if updated.Name != "Alice Smith" {
		t.Fatalf("expected updated name, got %s", updated.Name)
	}

	if err := service.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete returned error: %v", err)
	}

	_, err = service.Get(context.Background(), created.ID)
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
