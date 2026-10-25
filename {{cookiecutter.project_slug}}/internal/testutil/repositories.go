package testutil

import (
	"fmt"

	"{{ cookiecutter.go_module }}/internal/core/models"
	"{{ cookiecutter.go_module }}/internal/core/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FakeUserRepository is an in-memory IUserRepository used by the service and
// controller tests.
type FakeUserRepository struct {
	repositories.IUserRepository

	Users     map[uuid.UUID]*models.User
	CreateErr error
}

// NewFakeUserRepository returns an empty in-memory repository.
func NewFakeUserRepository() *FakeUserRepository {
	return &FakeUserRepository{Users: map[uuid.UUID]*models.User{}}
}

// Create stores user, returning CreateErr when set.
func (f *FakeUserRepository) Create(user *models.User) error {
	if f.CreateErr != nil {
		return f.CreateErr
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	f.Users[user.ID] = user
	return nil
}

// GetByPK returns the user with the given ID, or gorm.ErrRecordNotFound.
func (f *FakeUserRepository) GetByPK(id uuid.UUID) (*models.User, error) {
	user, ok := f.Users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

// GetOneBy supports the single "email = ?" lookup used by the tests and
// rejects any other query.
func (f *FakeUserRepository) GetOneBy(query any, args ...any) (*models.User, error) {
	if fmt.Sprint(query) != "email = ?" || len(args) != 1 {
		return nil, fmt.Errorf("unsupported query: %v %v", query, args)
	}
	email, _ := args[0].(string)
	for _, user := range f.Users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// Store inserts a user without going through Create, for login fixtures.
func (f *FakeUserRepository) Store(user *models.User) *models.User {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	f.Users[user.ID] = user
	return user
}
