package services

import (
	"testing"

	"{{ cookiecutter.go_module }}/internal/core/repositories"

	"github.com/google/uuid"
)

type fakeUserRepository struct {
	repositories.IUserRepository
}

func TestUserServiceLogin(t *testing.T) {
	t.Parallel()

	service := NewUserService(&fakeUserRepository{})

	user, err := service.Login("token", uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Fatalf("expected nil user, got %+v", user)
	}
}
