package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"openfms/api/internal/model"
)

// AuthService handles authentication business logic
type AuthService struct {
	db *gorm.DB
}

// NewAuthService creates a new auth service
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

// Authenticate validates user credentials
func (s *AuthService) Authenticate(ctx context.Context, username, password string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if user.Status != 1 {
		return nil, errors.New("user is inactive")
	}

	return &user, nil
}

// CreateUser creates a new user
func (s *AuthService) CreateUser(ctx context.Context, user *model.User) error {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	return s.db.Create(user).Error
}
