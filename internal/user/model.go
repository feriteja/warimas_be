package user

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type User struct {
	ID       int
	Email    string
	Password string
	Role     Role
	SellerID *string
}

type Profile struct {
	ID          uuid.UUID
	UserID      uint
	FullName    *string
	Bio         *string
	AvatarURL   *string
	Phone       *string
	Email       *string
	DateOfBirth *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type UpdateProfileParams struct {
	UserID      uint
	FullName    *string
	Bio         *string
	AvatarURL   *string
	Phone       *string
	DateOfBirth *time.Time
}
