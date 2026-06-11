package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/base64"
	"math/big"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/parthjindal/kavach/internal/models"
)

// TokenService handles token generation and management
type TokenService struct {
	BaseURL string // e.g., "https://t.kavach.dev"
}

// NewTokenService creates a new token service
func NewTokenService(baseURL string) *TokenService {
	return &TokenService{
		BaseURL: baseURL,
	}
}

// GenerateToken creates a new canary token based on the type
func (s *TokenService) GenerateToken(userID uuid.UUID, req models.CreateTokenRequest) (*models.Token, error) {
	token := &models.Token{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		IsActive:    true,
		TriggerCount: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Generate unique trigger identifier
	triggerID := generateSecureID(16)
	token.TriggerID = triggerID

	// Set payload and trigger URL based on token type
	switch req.Type {
	case models.TokenTypeURL:
		token.TriggerURL = fmt.Sprintf("%s/t/%s", s.BaseURL, triggerID)
		token.Payload = token.TriggerURL

	case models.TokenTypeDocument:
		// Document tokens embed a tracking pixel/callback URL
		token.TriggerURL = fmt.Sprintf("%s/t/%s/doc", s.BaseURL, triggerID)
		token.Payload = fmt.Sprintf("Document token: %s (embed trigger URL in PDF/DOCX metadata)", token.TriggerURL)

	case models.TokenTypeAPIKey:
		// Generate a realistic-looking API key
		apiKey := generateFakeAPIKey()
		token.TriggerURL = fmt.Sprintf("%s/t/%s/key", s.BaseURL, triggerID)
		token.Payload = apiKey

	case models.TokenTypeDNS:
		// DNS tokens trigger on DNS resolution
		token.TriggerURL = fmt.Sprintf("%s/t/%s/dns", s.BaseURL, triggerID)
		token.Payload = fmt.Sprintf("%s.t.kavach.dev", triggerID)

	case models.TokenTypeEmail:
		// Email tokens trigger when email is sent to them
		token.TriggerURL = fmt.Sprintf("%s/t/%s/email", s.BaseURL, triggerID)
		token.Payload = fmt.Sprintf("%s@trap.kavach.dev", triggerID)

	case models.TokenTypeQRCode:
		token.TriggerURL = fmt.Sprintf("%s/t/%s/qr", s.BaseURL, triggerID)
		// Generate QR code as base64 PNG data URI
		qrDataURI, err := GenerateQRCode(token.TriggerURL)
		if err != nil {
			return nil, fmt.Errorf("failed to generate QR code: %w", err)
		}
		token.Payload = qrDataURI

	case models.TokenTypeClonedSite:
		token.TriggerURL = fmt.Sprintf("%s/t/%s/clone", s.BaseURL, triggerID)
		domain := req.Domain
		if domain == "" {
			domain = "example.com"
		}
		token.Payload = generateClonedSiteJS(domain, token.TriggerURL)

	case models.TokenTypeWebImage:
		token.TriggerURL = fmt.Sprintf("%s/t/%s/pixel", s.BaseURL, triggerID)
		token.Payload = token.TriggerURL

	case models.TokenTypeAWSKey:
		token.TriggerURL = fmt.Sprintf("%s/t/%s/aws", s.BaseURL, triggerID)
		token.Payload = generateFakeAWSKeys()

	default:
		return nil, fmt.Errorf("unsupported token type: %s", req.Type)
	}

	return token, nil
}

// generateSecureID creates a cryptographically random hex string
func generateSecureID(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to UUID-based ID if crypto/rand fails
		return strings.ReplaceAll(uuid.New().String(), "-", "")[:length*2]
	}
	return hex.EncodeToString(bytes)
}

// generateFakeAPIKey creates a realistic-looking API key
// Format: kv_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxx (32 hex chars)
func generateFakeAPIKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("kv_live_%s", hex.EncodeToString(bytes))
}

// generateFakeAWSKeys creates a realistic-looking AWS access key pair
// Access Key: AKIA + 16 uppercase alphanumeric characters
// Secret Key: 40 character base64-like string
func generateFakeAWSKeys() string {
	const alphaNum = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Generate Access Key ID: AKIA + 16 chars
	accessKey := "AKIA"
	for i := 0; i < 16; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphaNum))))
		accessKey += string(alphaNum[n.Int64()])
	}

	// Generate Secret Access Key: 40 bytes base64-encoded (trimmed to 40 chars)
	secretBytes := make([]byte, 30)
	rand.Read(secretBytes)
	secretKey := base64.StdEncoding.EncodeToString(secretBytes)[:40]

	return fmt.Sprintf("AWS_ACCESS_KEY_ID=%s\nAWS_SECRET_ACCESS_KEY=%s", accessKey, secretKey)
}

// generateClonedSiteJS generates a JavaScript snippet for cloned website detection
func generateClonedSiteJS(legitimateDomain, triggerURL string) string {
	return fmt.Sprintf(`<script>
(function() {
  var legitimate = %q;
  var currentHost = window.location.hostname;
  if (currentHost !== legitimate && currentHost !== "www." + legitimate) {
    var data = {
      cloned_url: window.location.href,
      referrer: document.referrer,
      timestamp: new Date().toISOString(),
      screen: screen.width + "x" + screen.height,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      language: navigator.language
    };
    var img = new Image();
    img.src = %q + "?d=" + encodeURIComponent(JSON.stringify(data));
    // Also try beacon API for reliability
    if (navigator.sendBeacon) {
      navigator.sendBeacon(%q, JSON.stringify(data));
    }
  }
})();
</script>`, legitimateDomain, triggerURL, triggerURL)
}

// ValidateToken checks if a token exists and is active
func (s *TokenService) ValidateToken(tokenID uuid.UUID) bool {
	// TODO: Add database lookup
	return true
}

// DeactivateToken marks a token as inactive
func (s *TokenService) DeactivateToken(tokenID uuid.UUID) error {
	// TODO: Update database
	return nil
}

// GetTokenStats returns aggregated statistics for a user's tokens
type TokenStats struct {
	TotalTokens    int `json:"total_tokens"`
	ActiveTokens   int `json:"active_tokens"`
	TotalTriggers  int `json:"total_triggers"`
	TriggersToday  int `json:"triggers_today"`
	UniqueAttackers int `json:"unique_attackers"`
}

func (s *TokenService) GetStats(userID uuid.UUID) (*TokenStats, error) {
	// TODO: Query database for stats
	return &TokenStats{}, nil
}
