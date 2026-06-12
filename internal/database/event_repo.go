package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/parthjindal/kavach/internal/models"
)

// EventRepository handles database operations for trigger events
type EventRepository struct {
	db *sql.DB
}

// NewEventRepository creates a new event repository
func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// Create inserts a new trigger event
func (r *EventRepository) Create(event *models.TriggerEvent) error {
	query := `
		INSERT INTO trigger_events (id, token_id, attacker_id, ip_address, user_agent, referrer, country, city, fingerprint, headers, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.Exec(query,
		event.ID, event.TokenID, event.AttackerID,
		event.IPAddress, event.UserAgent, event.Referrer,
		event.Country, event.City, event.Fingerprint,
		event.Headers, event.Metadata, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create trigger event: %w", err)
	}
	return nil
}

// ListByTokenID retrieves all events for a specific token
func (r *EventRepository) ListByTokenID(tokenID uuid.UUID, limit int) ([]*models.TriggerEvent, error) {
	query := `
		SELECT id, token_id, attacker_id, ip_address, user_agent, referrer, country, city, fingerprint, headers, metadata, created_at
		FROM trigger_events WHERE token_id = $1
		ORDER BY created_at DESC LIMIT $2
	`
	return r.queryEvents(query, tokenID, limit)
}

// ListRecent retrieves the most recent trigger events across all tokens for a user
func (r *EventRepository) ListRecent(userID uuid.UUID, limit int) ([]*models.TriggerEvent, error) {
	query := `
		SELECT te.id, te.token_id, te.attacker_id, te.ip_address, te.user_agent, te.referrer, te.country, te.city, te.fingerprint, te.headers, te.metadata, te.created_at
		FROM trigger_events te
		JOIN tokens t ON te.token_id = t.id
		WHERE t.user_id = $1
		ORDER BY te.created_at DESC LIMIT $2
	`
	return r.queryEvents(query, userID, limit)
}

// CountToday returns the number of trigger events today for a user
func (r *EventRepository) CountToday(userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM trigger_events te
		JOIN tokens t ON te.token_id = t.id
		WHERE t.user_id = $1 AND te.created_at >= $2
	`
	today := time.Now().Truncate(24 * time.Hour)
	var count int
	err := r.db.QueryRow(query, userID, today).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count today's events: %w", err)
	}
	return count, nil
}

// CountUniqueAttackers returns the number of unique attackers for a user
func (r *EventRepository) CountUniqueAttackers(userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(DISTINCT te.fingerprint)
		FROM trigger_events te
		JOIN tokens t ON te.token_id = t.id
		WHERE t.user_id = $1 AND te.fingerprint != ''
	`
	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unique attackers: %w", err)
	}
	return count, nil
}

// GetTopCountries returns attack counts grouped by country
func (r *EventRepository) GetTopCountries(userID uuid.UUID, limit int) ([]CountryStats, error) {
	query := `
		SELECT te.country, COUNT(*) as count
		FROM trigger_events te
		JOIN tokens t ON te.token_id = t.id
		WHERE t.user_id = $1 AND te.country != ''
		GROUP BY te.country
		ORDER BY count DESC
		LIMIT $2
	`
	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top countries: %w", err)
	}
	defer rows.Close()

	var stats []CountryStats
	for rows.Next() {
		var s CountryStats
		if err := rows.Scan(&s.Country, &s.Count); err != nil {
			return nil, fmt.Errorf("failed to scan country stats: %w", err)
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating country stats: %w", err)
	}
	return stats, nil
}

// CountryStats holds country-level attack statistics
type CountryStats struct {
	Country string
	Count   int
}

// queryEvents is a helper to scan event rows
func (r *EventRepository) queryEvents(query string, args ...interface{}) ([]*models.TriggerEvent, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*models.TriggerEvent
	for rows.Next() {
		event := &models.TriggerEvent{}
		err := rows.Scan(
			&event.ID, &event.TokenID, &event.AttackerID,
			&event.IPAddress, &event.UserAgent, &event.Referrer,
			&event.Country, &event.City, &event.Fingerprint,
			&event.Headers, &event.Metadata, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}
	return events, nil
}
