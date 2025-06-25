package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleSubAdmin Role = "subadmin"
	RoleUser     Role = "user"
)

func (r Role) isValid() bool {
	return r == RoleAdmin || r == RoleSubAdmin || r == RoleUser
}

type User struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	Name       string     `db:"name" json:"name"`
	Email      string     `db:"email" json:"email"`
	Password   string     `db:"password" json:"-"`
	Roles      []UserRole `db:"-" json:"roles"`
	Addresses  []Address  `db:"-" json:"addresses"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	ArchivedAt *time.Time `db:"archived_at" json:"archived_at,omitempty"`
	CreatedBy  uuid.UUID  `db:"created_by" json:"created_by"`
}

type UserRole struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	UserID     uuid.UUID  `db:"user_id" json:"user_id"`
	Role       Role       `db:"role" json:"role"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	ArchivedAt *time.Time `db:"archived_at" json:"archived_at,omitempty"`
}

type Address struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Address   string    `db:"address" json:"address"`
	Latitude  float64   `db:"latitude" json:"latitude"`
	Longitude float64   `db:"longitude" json:"longitude"`
}
