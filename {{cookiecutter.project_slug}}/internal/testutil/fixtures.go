package testutil

import (
	"{{ cookiecutter.go_module }}/internal/core/models"
)

func NewUser() *models.User {
	return &models.User{
		Firstname: "Ada",
		Lastname:  "Lovelace",
		Email:     "ada@example.com",
		Password:  "hashed-password",
	}
}
