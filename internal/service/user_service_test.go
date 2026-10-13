package service

import (
	"context"
	"errors"
	"testing"

	"movie-reservation-system/internal/model"
	"movie-reservation-system/internal/util"
)

func TestUserService_GetByID(t *testing.T) {
	repo := newFakeUserRepo()
	stored := &model.User{Name: "Alice", Email: "alice@example.com", Role: model.RoleUser}
	if err := repo.Create(context.Background(), stored); err != nil {
		t.Fatalf("seeding the fake repo failed: %v", err)
	}
	svc := NewUserService(repo)

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(context.Background(), stored.ID)
		if err != nil {
			t.Fatalf("GetByID returned error: %v", err)
		}
		if got.Email != stored.Email {
			t.Errorf("Email = %q, want %q", got.Email, stored.Email)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), 9999)
		if !errors.Is(err, util.ErrNotFound) {
			t.Fatalf("err = %v, want util.ErrNotFound", err)
		}
	})
}
