package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/parthjindal/kavach/internal/models"
)

// TokenRepository handles database operations for tokens
type TokenRepository struct {
	db *sql.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// Create inserts a new token into the database
func (r *TokenRepository) Create(token *models.Token, triggerID string) error {
	query := `
		INSERT INTO tokens (id, user_id, name, type, description, payload, trigger_url, trigger_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(query,
		token.ID, token.UserID, token.Name, token.Type,
		token.Description, token.Payload, token.TriggerURL,
		triggerID, token.IsActive, token.CreatedAt, token.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}
	return nil
}

// GetByID retrieves a token by its ID
func (r *TokenRepository) GetByID(id uuid.UUID) (*models.Token, error) {
	query := `
		SELECT id, user_id, name, type, description, payload, trigger_url, is_active, trigger_count, last_triggered, created_at, updated_at
		FROM tokens WHERE id = $1
	`
	token := &models.Token{}
	err := r.db.QueryRow(query, id).Scan(
		&token.ID, &token.UserID, &token.Name, &token.Type,
		&token.Description, &token.Payload, &token.TriggerURL,
		&token.IsActive, &token.TriggerCount, &token.LastTriggered,
		&token.CreatedAt, &token.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	return token, nil
}

// GetByTriggerID retrieves a token by its trigger ID (used when a token is triggered)
func (r *TokenRepository) GetByTriggerID(triggerID string) (*models.Token, error) {
	query := `
		SELECT id, user_id, name, type, description, payload, trigger_url, is_active, trigger_count, last_triggered, created_at, updated_at
		FROM tokens WHERE trigger_id = $1 AND is_active = true
	`
	token := &models.Token{}
	err := r.db.QueryRow(query, triggerID).Scan(
		&token.ID, &token.UserID, &token.Name, &token.Type,
		&token.Description, &token.Payload, &token.TriggerURL,
		&token.IsActive, &token.TriggerCount, &token.LastTriggered,
		&token.CreatedAt, &token.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token by trigger ID: %w", err)
	}
	return token, nil
}

// ListByUserID retrieves all tokens for a user
func (r *TokenRepository) ListByUserID(userID uuid.UUID, limit, offset int) ([]*models.Token, error) {
	query := `
		SELECT id, user_id, name, type, description, payload, trigger_url, is_active, trigger_count, last_triggered, created_at, updated_at
		FROM tokens WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.Token
	for rows.Next() {
		token := &models.Token{}
		err := rows.Scan(
			&token.ID, &token.UserID, &token.Name, &token.Type,
			&token.Description, &token.Payload, &token.TriggerURL,
			&token.IsActive, &token.TriggerCount, &token.LastTriggered,
			&token.CreatedAt, &token.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// IncrementTriggerCount updates the trigger count and last triggered timestamp
func (r *TokenRepository) IncrementTriggerCount(tokenID uuid.UUID) error {
	query := `
		UPDATE tokens 
		SET trigger_count = trigger_count + 1, last_triggered = $2, updated_at = $2
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.Exec(query, tokenID, now)
	if err != nil {
		return fmt.Errorf("failed to increment trigger count: %w", err)
	}
	return nil
}

// Deactivate marks a token as inactive
func (r *TokenRepository) Deactivate(tokenID uuid.UUID) error {
	query := `UPDATE tokens SET is_active = false, updated_at = $2 WHERE id = $1`
	_, err := r.db.Exec(query, tokenID, time.Now())
	return err
}

// Delete removes a token permanently
func (r *TokenRepository) Delete(tokenID uuid.UUID) error {
	query := `DELETE FROM tokens WHERE id = $1`
	_, err := r.db.Exec(query, tokenID)
	return err
}

// CountByUserID returns total token count for a user
func (r *TokenRepository) CountByUserID(userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM tokens WHERE user_id = $1 AND is_active = true`, userID).Scan(&count)
	return count, err
}

// Update modifies a token's name, description, and active status
func (r *TokenRepository) Update(tokenID uuid.UUID, name, description string, isActive bool) error {
	query := `
		UPDATE tokens 
		SET name = $2, description = $3, is_active = $4, updated_at = $5
		WHERE id = $1
	`
	_, err := r.db.Exec(query, tokenID, name, description, isActive, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update token: %w", err)
	}
	return nil
}
