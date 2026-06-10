package models

import (
	"time"

	"github.com/google/uuid"
)

// TokenType represents the type of canary token
type TokenType string

const (
	TokenTypeURL      TokenType = "url"
	TokenTypeDocument TokenType = "document"
	TokenTypeAPIKey   TokenType = "api_key"
	TokenTypeDNS      TokenType = "dns"
	TokenTypeEmail    TokenType = "email"
)

// Token represents a canary token deployed by a user
type Token struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	Type        TokenType  `json:"type" db:"type"`
	Description string     `json:"description" db:"description"`
	TriggerID   string     `json:"trigger_id" db:"trigger_id"`     // Unique trigger identifier for URL routing
	Payload     string     `json:"payload" db:"payload"`       // The actual token value (URL, key, etc.)
	TriggerURL  string     `json:"trigger_url" db:"trigger_url"` // URL that captures fingerprint
	IsActive    bool       `json:"is_active" db:"is_active"`
	TriggerCount int       `json:"trigger_count" db:"trigger_count"`
	LastTriggered *time.Time `json:"last_triggered" db:"last_triggered"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateTokenRequest is the request payload to create a new token
type CreateTokenRequest struct {
	Name        string    `json:"name" validate:"required,min=1,max=100"`
	Type        TokenType `json:"type" validate:"required,oneof=url document api_key dns email"`
	Description string    `json:"description" validate:"max=500"`
}

// TokenResponse is the API response for a token
type TokenResponse struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Type         TokenType  `json:"type"`
	Description  string     `json:"description"`
	Payload      string     `json:"payload"`
	TriggerURL   string     `json:"trigger_url"`
	IsActive     bool       `json:"is_active"`
	TriggerCount int        `json:"trigger_count"`
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
