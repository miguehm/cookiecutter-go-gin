// Package models defines the GORM entities persisted by the application.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User is the users table. The password hash and the soft delete marker are
// never serialized to JSON.
type User struct {
	ID        uuid.UUID      `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Firstname string         `gorm:"size:255;not null" json:"firstname"`
	Lastname  string         `gorm:"size:255;not null" json:"lastname"`
	Email     string         `gorm:"size:100;not null;unique" json:"email"`
	Password  string         `gorm:"size:100;not null" json:"-"`
}
