package services

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/parthjindal/kavach/internal/fingerprint"
	"github.com/parthjindal/kavach/internal/models"
)

// AttackerService correlates events to attacker profiles
type AttackerService struct {
	geoService *GeoService
	// In demo mode, we store attackers in memory
	mu        sync.RWMutex
	attackers map[string]*models.Attacker
}

// NewAttackerService creates a new attacker correlation service
func NewAttackerService(geoService *GeoService) *AttackerService {
	return &AttackerService{
		geoService: geoService,
		attackers:  make(map[string]*models.Attacker),
	}
}

// FindOrCreate finds an existing attacker by fingerprint hash, or creates a new profile
func (s *AttackerService) FindOrCreate(fp *fingerprint.CapturedFingerprint) (*models.Attacker, error) {
	hash := fp.UniqueHash
	if hash == "" {
		hash = "fp_unknown_" + fp.IPAddress
	}

	// Fix NEW-2: Use a single write lock for the check-and-update path
	// to eliminate TOCTOU race between RUnlock and Lock.
	s.mu.Lock()
	if attacker, exists := s.attackers[hash]; exists {
		attacker.LastSeenAt = time.Now()
		attacker.TriggerCount++
		attacker.ThreatLevel = s.calculateThreatLevel(attacker)
		s.mu.Unlock()
		log.Printf("Existing attacker correlated: %s (triggers: %d)", hash, attacker.TriggerCount)
		return attacker, nil
	}
	s.mu.Unlock()

	// Enrich with geo data (outside lock - network call)
	geo, err := s.geoService.Lookup(fp.IPAddress)
	if err != nil {
		log.Printf("Geo lookup failed for %s: %v", fp.IPAddress, err)
		geo = &GeoInfo{IP: fp.IPAddress, Country: "Unknown", City: "Unknown"}
	}
	// Nil check for geo (fix B8 - defensive against future changes)
	if geo == nil {
		geo = &GeoInfo{IP: fp.IPAddress, Country: "Unknown", City: "Unknown"}
	}

	// Create new attacker profile
	attacker := &models.Attacker{
		ID:             uuid.New(),
		Fingerprint:    hash,
		IPAddress:      fp.IPAddress,
		Country:        geo.Country,
		City:           geo.City,
		Region:         geo.Region,
		ISP:            geo.ISP,
		ASN:            geo.ASN,
		UserAgent:      fp.UserAgent,
		Browser:        fp.Browser,
		BrowserVersion: fp.BrowserVer,
		OS:             fp.OS,
		OSVersion:      fp.OSVersion,
		TLSFingerprint: fp.TLSFingerprint,
		IsVPN:          geo.IsVPN,
		IsTor:          geo.IsTor,
		IsProxy:        geo.IsProxy,
		ThreatLevel:    models.ThreatLevelLow,
		TriggerCount:   1,
		TokensTriggered: 1,
		FirstSeenAt:    time.Now(),
		LastSeenAt:     time.Now(),
	}

	// Adjust threat based on indicators
	if geo.IsTor || geo.IsVPN || geo.IsProxy {
		attacker.ThreatLevel = models.ThreatLevelMedium
	}

	// Double-check under write lock (another goroutine may have created it during geo lookup)
	s.mu.Lock()
	if existing, exists := s.attackers[hash]; exists {
		// Another goroutine created it while we were doing geo lookup
		existing.LastSeenAt = time.Now()
		existing.TriggerCount++
		existing.ThreatLevel = s.calculateThreatLevel(existing)
		s.mu.Unlock()
		log.Printf("Existing attacker correlated (race resolved): %s (triggers: %d)", hash, existing.TriggerCount)
		return existing, nil
	}
	s.attackers[hash] = attacker
	s.mu.Unlock()
	log.Printf("New attacker profiled: %s from %s, %s", hash, geo.City, geo.Country)

	return attacker, nil
}

// calculateThreatLevel determines threat based on behavior patterns
func (s *AttackerService) calculateThreatLevel(a *models.Attacker) models.ThreatLevel {
	switch {
	case a.TriggerCount >= 10 || a.TokensTriggered >= 5:
		return models.ThreatLevelCritical
	case a.TriggerCount >= 5 || a.TokensTriggered >= 3:
		return models.ThreatLevelHigh
	case a.TriggerCount >= 2:
		return models.ThreatLevelMedium
	default:
		return models.ThreatLevelLow
	}
}

// GetAll returns all known attackers (for demo mode)
func (s *AttackerService) GetAll() []*models.Attacker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*models.Attacker, 0, len(s.attackers))
	for _, a := range s.attackers {
		result = append(result, a)
	}
	return result
}

// GetByID returns an attacker by ID
func (s *AttackerService) GetByID(id string) *models.Attacker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.attackers {
		if a.ID.String() == id {
			return a
		}
	}
	return nil
}

// GetMockAttackers returns demo attacker data
func GetMockAttackers() []models.Attacker {
	return []models.Attacker{
		{
			ID: uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			Fingerprint: "fp_3a7f2b9c1d4e5f60",
			IPAddress: "103.45.67.89",
			Country: "IN",
			City: "Mumbai",
			ISP: "BSNL",
			ASN: "AS9829",
			UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
			Browser: "Chrome",
			OS: "Linux",
			ThreatLevel: models.ThreatLevelCritical,
			TriggerCount: 12,
			TokensTriggered: 4,
			FirstSeenAt: time.Now().Add(-72 * time.Hour),
			LastSeenAt: time.Now().Add(-2 * time.Minute),
		},
		{
			ID: uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-f12345678901"),
			Fingerprint: "fp_8b2e4c6a0d1f3759",
			IPAddress: "45.33.21.110",
			Country: "BR",
			City: "Sao Paulo",
			ISP: "Amazon AWS",
			ASN: "AS16509",
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0)",
			Browser: "Firefox",
			OS: "Windows",
			ThreatLevel: models.ThreatLevelHigh,
			TriggerCount: 5,
			TokensTriggered: 2,
			FirstSeenAt: time.Now().Add(-48 * time.Hour),
			LastSeenAt: time.Now().Add(-18 * time.Minute),
		},
		{
			ID: uuid.MustParse("c3d4e5f6-a7b8-9012-cdef-123456789012"),
			Fingerprint: "fp_1c5f9a3d7b2e4068",
			IPAddress: "92.168.1.44",
			Country: "DE",
			City: "Berlin",
			ISP: "Deutsche Telekom",
			ASN: "AS3320",
			UserAgent: "Mozilla/5.0 (compatible; Googlebot/2.1)",
			Browser: "Bot",
			OS: "Unknown",
			ThreatLevel: models.ThreatLevelMedium,
			TriggerCount: 3,
			TokensTriggered: 1,
			FirstSeenAt: time.Now().Add(-24 * time.Hour),
			LastSeenAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID: uuid.MustParse("d4e5f6a7-b8c9-0123-defa-234567890123"),
			Fingerprint: "fp_9e1a3b5c7d2f4068",
			IPAddress: "185.220.101.1",
			Country: "XX",
			City: "Unknown",
			ISP: "Tor Network",
			ASN: "AS0",
			UserAgent: "curl/7.88.1",
			Browser: "curl",
			OS: "Unknown",
			ThreatLevel: models.ThreatLevelHigh,
			TriggerCount: 7,
			TokensTriggered: 3,
			FirstSeenAt: time.Now().Add(-96 * time.Hour),
			LastSeenAt: time.Now().Add(-3 * time.Hour),
			Notes: "Tor exit node - likely automated scanning",
		},
	}
}
