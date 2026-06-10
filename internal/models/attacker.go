package models

import (
	"time"

	"github.com/google/uuid"
)

// ThreatLevel represents the risk level of an attacker
type ThreatLevel string

const (
	ThreatLevelLow      ThreatLevel = "low"
	ThreatLevelMedium   ThreatLevel = "medium"
	ThreatLevelHigh     ThreatLevel = "high"
	ThreatLevelCritical ThreatLevel = "critical"
)

// Attacker represents a profiled threat actor
type Attacker struct {
	ID              uuid.UUID   `json:"id" db:"id"`
	Fingerprint     string      `json:"fingerprint" db:"fingerprint"` // Unique device fingerprint hash
	IPAddress       string      `json:"ip_address" db:"ip_address"`
	Country         string      `json:"country" db:"country"`
	City            string      `json:"city" db:"city"`
	Region          string      `json:"region" db:"region"`
	ISP             string      `json:"isp" db:"isp"`
	ASN             string      `json:"asn" db:"asn"`
	UserAgent       string      `json:"user_agent" db:"user_agent"`
	Browser         string      `json:"browser" db:"browser"`
	BrowserVersion  string      `json:"browser_version" db:"browser_version"`
	OS              string      `json:"os" db:"os"`
	OSVersion       string      `json:"os_version" db:"os_version"`
	TLSFingerprint  string      `json:"tls_fingerprint" db:"tls_fingerprint"` // JA3/JA4 hash
	IsVPN           bool        `json:"is_vpn" db:"is_vpn"`
	IsTor           bool        `json:"is_tor" db:"is_tor"`
	IsProxy         bool        `json:"is_proxy" db:"is_proxy"`
	ThreatLevel     ThreatLevel `json:"threat_level" db:"threat_level"`
	TriggerCount    int         `json:"trigger_count" db:"trigger_count"`
	TokensTriggered int         `json:"tokens_triggered" db:"tokens_triggered"`
	FirstSeenAt     time.Time   `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt      time.Time   `json:"last_seen_at" db:"last_seen_at"`
	Notes           string      `json:"notes" db:"notes"`
}

// TriggerEvent represents a single token trigger event
type TriggerEvent struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TokenID     uuid.UUID `json:"token_id" db:"token_id"`
	AttackerID  uuid.UUID `json:"attacker_id" db:"attacker_id"`
	IPAddress   string    `json:"ip_address" db:"ip_address"`
	UserAgent   string    `json:"user_agent" db:"user_agent"`
	Referrer    string    `json:"referrer" db:"referrer"`
	Country     string    `json:"country" db:"country"`
	City        string    `json:"city" db:"city"`
	Fingerprint string    `json:"fingerprint" db:"fingerprint"`
	Headers     string    `json:"headers" db:"headers"` // JSON-encoded request headers
	Metadata    string    `json:"metadata" db:"metadata"` // Additional context (JSON)
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
