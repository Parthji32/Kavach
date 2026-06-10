package models

import (
	"time"

	"github.com/google/uuid"
)

// Plan represents a user subscription plan
type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
	PlanTeam Plan = "team"
)

// User represents a registered user
type User struct {
	ID            uuid.UUID `json:"id" db:"id"`
	Email         string    `json:"email" db:"email"`
	PassHash      string    `json:"-" db:"pass_hash"`
	Name          string    `json:"name" db:"name"`
	Company       string    `json:"company" db:"company"`
	Plan          Plan      `json:"plan" db:"plan"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// SignupRequest is the request payload to register a new user
type SignupRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Credential string `json:"credential" validate:"required,min=8"`
	Name       string `json:"name" validate:"required,min=1,max=100"`
	Company    string `json:"company" validate:"max=200"`
}

// LoginRequest is the request payload to log in
type LoginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Credential string `json:"credential" validate:"required"`
}

// AuthResponse is returned after successful login/signup
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
