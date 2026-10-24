package services

import (
	"errors"
	"testing"

	"{{ cookiecutter.go_module }}/internal/core/apierrors"
	"{{ cookiecutter.go_module }}/internal/core/models"
	"{{ cookiecutter.go_module }}/internal/core/vo"
	"{{ cookiecutter.go_module }}/internal/testutil"
	"{{ cookiecutter.go_module }}/pkg/cryptography"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestCreateUserHashesThePassword(t *testing.T) {
	t.Parallel()

	repo := testutil.NewFakeUserRepository()
	service := NewUserService(repo)

	user, err := service.CreateUser(&vo.CreateUserRequest{
		Firstname: "Ada",
		Lastname:  "Lovelace",
		Email:     "ada@example.com",
		Password:  "s3cret-password",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID == uuid.Nil {
		t.Error("the created user must have an id")
	}
	if user.Password == "s3cret-password" {
		t.Fatal("the password must not be stored in plain text")
	}
	if err := cryptography.VerifyPassword(user.Password, "s3cret-password"); err != nil {
		t.Fatalf("the stored hash does not verify: %v", err)
	}
}

func TestCreateUserReportsDuplicatedEmail(t *testing.T) {
	t.Parallel()

	repo := testutil.NewFakeUserRepository()
	repo.CreateErr = gorm.ErrDuplicatedKey
	service := NewUserService(repo)

	_, err := service.CreateUser(&vo.CreateUserRequest{
		Firstname: "Ada",
		Lastname:  "Lovelace",
		Email:     "ada@example.com",
		Password:  "s3cret-password",
	})
	if !errors.Is(err, apierrors.ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestLoginAcceptsCorrectPassword(t *testing.T) {
	t.Parallel()

	repo := testutil.NewFakeUserRepository()
	hashed, err := cryptography.HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	repo.Store(&models.User{
		Email:    "ada@example.com",
		Password: string(hashed),
	})
	service := NewUserService(repo)

	user, err := service.Login("ada@example.com", "s3cret-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "ada@example.com" {
		t.Errorf("email = %q", user.Email)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	t.Parallel()

	repo := testutil.NewFakeUserRepository()
	hashed, err := cryptography.HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	repo.Store(&models.User{Email: "ada@example.com", Password: string(hashed)})
	service := NewUserService(repo)

	_, err = service.Login("ada@example.com", "wrong")
	if !errors.Is(err, apierrors.ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginRejectsUnknownUser(t *testing.T) {
	t.Parallel()

	repo := testutil.NewFakeUserRepository()
	service := NewUserService(repo)

	_, err := service.Login("ghost@example.com", "s3cret-password")
	if !errors.Is(err, apierrors.ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestGetByIDReportsMissingUser(t *testing.T) {
	t.Parallel()

	repo := testutil.NewFakeUserRepository()
	service := NewUserService(repo)

	_, err := service.GetByID(uuid.New())
	if !errors.Is(err, apierrors.ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}
