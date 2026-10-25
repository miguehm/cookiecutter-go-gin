package repositories

import (
	"{{ cookiecutter.go_module }}/internal/core/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IUserRepository is the persistence contract of the users table.
type IUserRepository interface {
	IRepository[models.User, uuid.UUID]
}

type userRepository struct {
	repository[models.User, uuid.UUID]
}

// NewUserRepository returns a user repository bound to db.
func NewUserRepository(db *gorm.DB) IUserRepository {
	return &userRepository{
		repository[models.User, uuid.UUID]{
			db: db,
		},
	}
}
