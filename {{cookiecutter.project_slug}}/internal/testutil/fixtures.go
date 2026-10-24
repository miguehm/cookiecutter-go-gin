package testutil

import (
	"{{ cookiecutter.go_module }}/internal/core/models"
	"{{ cookiecutter.go_module }}/pkg/cryptography"

	"github.com/google/uuid"
)

func NewUser() *models.User {
	return &models.User{
		ID:        uuid.New(),
		Firstname: "Ada",
		Lastname:  "Lovelace",
		Email:     "ada@example.com",
		Password:  "hashed-password",
	}
}

// MustHashPassword hashes a password with bcrypt for tests.
func MustHashPassword(password string) string {
	hashed, err := cryptography.HashPassword(password)
	if err != nil {
		panic(err)
	}
	return string(hashed)
}
