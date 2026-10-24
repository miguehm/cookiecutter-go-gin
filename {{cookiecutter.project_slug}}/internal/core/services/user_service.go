package services

import (
	"errors"

	"{{ cookiecutter.go_module }}/internal/core/apierrors"
	"{{ cookiecutter.go_module }}/internal/core/models"
	"{{ cookiecutter.go_module }}/internal/core/repositories"
	"{{ cookiecutter.go_module }}/internal/core/vo"
	"{{ cookiecutter.go_module }}/pkg/cryptography"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	userRepository repositories.IUserRepository
}

func NewUserService(userRepo repositories.IUserRepository) *UserService {
	return &UserService{
		userRepo,
	}
}

// CreateUser stores a new user with a bcrypt hashed password.
func (s *UserService) CreateUser(req *vo.CreateUserRequest) (*models.User, error) {
	hashed, err := cryptography.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Firstname: req.Firstname,
		Lastname:  req.Lastname,
		Email:     req.Email,
		Password:  string(hashed),
	}
	if err := s.userRepository.Create(user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apierrors.ErrEmailTaken
		}
		return nil, err
	}
	return user, nil
}

// Login verifies the credentials of a user. Unknown users and wrong
// passwords are indistinguishable to avoid leaking which accounts exist.
func (s *UserService) Login(email, password string) (*models.User, error) {
	user, err := s.userRepository.GetOneBy("email = ?", email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrInvalidCredentials
		}
		return nil, err
	}
	if err := cryptography.VerifyPassword(user.Password, password); err != nil {
		return nil, apierrors.ErrInvalidCredentials
	}
	return user, nil
}

// GetByID returns a single user by primary key.
func (s *UserService) GetByID(id uuid.UUID) (*models.User, error) {
	user, err := s.userRepository.GetByPK(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
