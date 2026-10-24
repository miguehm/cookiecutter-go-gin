package vo

import (
	"{{ cookiecutter.go_module }}/internal/core/models"

	"github.com/google/uuid"
)

// CreateUserRequest is the payload of POST /signup.
type CreateUserRequest struct {
	Firstname string `json:"firstname" binding:"required,max=255"`
	Lastname  string `json:"lastname" binding:"required,max=255"`
	Email     string `json:"email" binding:"required,email,max=100"`
	Password  string `json:"password" binding:"required,min=8,max=72"`
}

// LoginRequest is the payload of POST /login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserVO is the public representation of a user: the password never leaves
// the API.
type UserVO struct {
	ID        uuid.UUID `json:"id"`
	Firstname string    `json:"firstname"`
	Lastname  string    `json:"lastname"`
	Email     string    `json:"email"`
}

func NewUserVO(user *models.User) UserVO {
	return UserVO{
		ID:        user.ID,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Email:     user.Email,
	}
}

// TokenVO is the payload returned by POST /login.
type TokenVO struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}
