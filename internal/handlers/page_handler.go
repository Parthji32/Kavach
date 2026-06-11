package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/parthjindal/kavach/internal/database"
	"github.com/parthjindal/kavach/internal/models"
	"github.com/parthjindal/kavach/internal/services"
)

// PageHandler serves HTML pages using Go templates + HTMX
type PageHandler struct {
	templateDir string
	db          *sql.DB
	funcMap     template.FuncMap
}

// NewPageHandler creates a new page handler
func NewPageHandler(templateDir string, db ...*sql.DB) *PageHandler {
	funcMap := template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			// Capitalize first letter safely (only if it's lowercase ASCII a-z)
			if s[0] >= 'a' && s[0] <= 'z' {
				return string(s[0]-32) + s[1:]
			}
			return s
		},
		"upper": func(s string) string {
			out := make([]byte, len(s))
			for i, c := range []byte(s) {
				if c >= 'a' && c <= 'z' {
					out[i] = c - 32
				} else {
					out[i] = c
				}
			}
			return string(out)
		},
	}

	var dbConn *sql.DB
	if len(db) > 0 {
		dbConn = db[0]
	}

	return &PageHandler{
		templateDir: templateDir,
		db:          dbConn,
		funcMap:     funcMap,
	}
}

// ===================== DATA STRUCTS =====================

type DashboardData struct {
	Title        string
	ActiveNav    string
	AlertCount   int
	Stats        DashboardStats
	RecentAlerts []AlertItem
	TopCountries []CountryItem
	RecentIPs    []IPItem
	RecentTokens []TokenItem
}

type DashboardStats struct {
	ActiveTokens       int
	TokensThisWeek     int
	TriggersToday      int
	TriggersDelta      int
	UniqueAttackers    int
	AttackersThisMonth int
	ThreatLevel        string
	CriticalAlerts     int
}

type AlertItem struct {
	ID        string
	Title     string
	TokenType string
	Severity  string
	IPAddress string
	Location  string
	Browser   string
	TimeAgo   string
}

type CountryItem struct {
	Flag       string
	Name       string
	Percentage int
}

type IPItem struct {
	IP      string
	TimeAgo string
}

type TokenItem struct {
	ID            string
	Name          string
	Type          string
	IsTriggered   bool
	TriggerCount  int
	LastTriggered string
	Payload       string
	Description   string
	CreatedAt     string
	IsActive      bool
}

// EventItem represents a trigger event in the token detail timeline
type EventItem struct {
	IPAddress string
	Country   string
	City      string
	Browser   string
	UserAgent string
	TimeAgo   string
	CreatedAt string
}

type AttackerItem struct {
	ID              string
	Fingerprint     string
	IPAddress       string
	Country         string
	City            string
	ISP             string
	ASN             string
	Browser         string
	OS              string
	ThreatLevel     string
	TriggerCount    int
	TokensTriggered int
	FirstSeen       string
	LastSeen        string
	Notes           string
	UserAgent       string
	TLSFingerprint  string
	Events          []AlertItem
}

type IntegrationData struct {
	Title      string
	ActiveNav  string
	AlertCount int
	SlackURL   string
	EmailTo    string
	EmailFrom  string
	IsSlackOn  bool
	IsEmailOn  bool
}

// TokenDetailData is the data struct for the token detail page
type TokenDetailData struct {
	Title         string
	ActiveNav     string
	AlertCount    int
	Token         TokenItem
	TriggerURL    string
	Events        []EventItem
	LastTriggered string
}

// TokenEditData is the data struct for the token edit page
type TokenEditData struct {
	Title      string
	ActiveNav  string
	AlertCount int
	Token      TokenItem
}

// ===================== PAGE HANDLERS =====================

// Dashboard renders the main dashboard page
func (h *PageHandler) Dashboard(c *fiber.Ctx) error {
	data := DashboardData{
		Title:      "Dashboard",
		ActiveNav:  "dashboard",
		AlertCount: 3,
		Stats: DashboardStats{
			ActiveTokens:       24,
			TokensThisWeek:     3,
			TriggersToday:      7,
			TriggersDelta:      2,
			UniqueAttackers:    12,
			AttackersThisMonth: 4,
			ThreatLevel:        "medium",
			CriticalAlerts:     3,
		},
		RecentAlerts: []AlertItem{
			{ID: "1", Title: "API Key token triggered - production-db-key", TokenType: "api_key", Severity: "critical", IPAddress: "103.45.67.89", Location: "Mumbai, India", Browser: "Chrome/Linux", TimeAgo: "2 min ago"},
			{ID: "2", Title: "Document token opened - financials_2026.pdf", TokenType: "document", Severity: "critical", IPAddress: "45.33.21.110", Location: "Sao Paulo, Brazil", Browser: "Firefox/Win", TimeAgo: "18 min ago"},
			{ID: "3", Title: "URL token accessed - internal-wiki-backup", TokenType: "url", Severity: "warning", IPAddress: "92.168.1.44", Location: "Berlin, Germany", Browser: "Bot/Crawler", TimeAgo: "1 hr ago"},
		},
		TopCountries: []CountryItem{
			{Flag: "\xf0\x9f\x87\xae\xf0\x9f\x87\xb3", Name: "India", Percentage: 34},
			{Flag: "\xf0\x9f\x87\xa7\xf0\x9f\x87\xb7", Name: "Brazil", Percentage: 22},
			{Flag: "\xf0\x9f\x87\xa9\xf0\x9f\x87\xae", Name: "Germany", Percentage: 18},
			{Flag: "\xf0\x9f\x8c\x90", Name: "Tor/VPN", Percentage: 26},
		},
		RecentIPs: []IPItem{
			{IP: "103.45.67.89", TimeAgo: "2m"},
			{IP: "45.33.21.110", TimeAgo: "18m"},
			{IP: "92.168.1.44", TimeAgo: "1h"},
			{IP: "185.220.101.1", TimeAgo: "3h"},
		},
		RecentTokens: []TokenItem{
			{ID: "1", Name: "production-db-key", Type: "api_key", IsTriggered: true, TriggerCount: 12, LastTriggered: "2 min ago"},
			{ID: "2", Name: "financials_2026.pdf", Type: "document", IsTriggered: true, TriggerCount: 3, LastTriggered: "18 min ago"},
			{ID: "3", Name: "internal-wiki-backup", Type: "url", IsTriggered: false, TriggerCount: 5, LastTriggered: "1 hr ago"},
			{ID: "4", Name: "staging-api.internal", Type: "dns", IsTriggered: false, TriggerCount: 1, LastTriggered: "3 hrs ago"},
			{ID: "5", Name: "server-room-access-qr", Type: "qr_code", IsTriggered: true, TriggerCount: 2, LastTriggered: "45 min ago"},
			{ID: "6", Name: "marketing-site-clone-detect", Type: "cloned_site", IsTriggered: false, TriggerCount: 0, LastTriggered: "Never"},
			{ID: "7", Name: "newsletter-tracking-pixel", Type: "web_image", IsTriggered: true, TriggerCount: 8, LastTriggered: "5 min ago"},
			{ID: "8", Name: "dev-aws-creds-honeypot", Type: "aws_key", IsTriggered: false, TriggerCount: 1, LastTriggered: "2 hrs ago"},
		},
	}

	return h.renderPage(c, "dashboard/index.html", data)
}

// TokensList renders the tokens list page
func (h *PageHandler) TokensList(c *fiber.Ctx) error {
	var tokens []TokenItem

	// Try to load real tokens from database
	if h.db != nil {
		tokenRepo := database.NewTokenRepository(h.db)
		// Get default user ID
		var userID uuid.UUID
		row := h.db.QueryRow("SELECT id FROM users WHERE email = 'admin@kavach.dev'")
		if err := row.Scan(&userID); err == nil {
			dbTokens, err := tokenRepo.ListByUserID(userID, 100, 0)
			if err == nil && len(dbTokens) > 0 {
				tokens = make([]TokenItem, len(dbTokens))
				for i, t := range dbTokens {
					lastTriggered := "Never"
					if t.LastTriggered != nil {
						lastTriggered = formatTimeAgo(*t.LastTriggered)
					}
					tokens[i] = TokenItem{
						ID:            t.ID.String(),
						Name:          t.Name,
						Type:          string(t.Type),
						IsTriggered:   t.TriggerCount > 0,
						TriggerCount:  t.TriggerCount,
						LastTriggered: lastTriggered,
						Payload:       t.Payload,
						Description:   t.Description,
						CreatedAt:     t.CreatedAt.Format("Jan 2, 2006"),
						IsActive:      t.IsActive,
					}
				}
			} else if err != nil {
				log.Printf("Failed to list tokens from DB: %v", err)
			}
		}
	}

	// Fallback to mock data if DB is nil or returned no tokens
	if tokens == nil {
		tokens = []TokenItem{
			{ID: "1", Name: "production-db-key", Type: "api_key", IsTriggered: true, TriggerCount: 12, LastTriggered: "2 min ago", Payload: "kv_live_a3f9c2b1d4e5f6a7b8c9d0e1f2a3b4c5", Description: "Planted in .env on staging server", CreatedAt: "Jun 1, 2026", IsActive: true},
			{ID: "2", Name: "financials_2026.pdf", Type: "document", IsTriggered: true, TriggerCount: 3, LastTriggered: "18 min ago", Payload: "https://t.kavach.dev/t/abc123/doc", Description: "Fake financial report in shared drive", CreatedAt: "May 28, 2026", IsActive: true},
			{ID: "3", Name: "internal-wiki-backup", Type: "url", IsTriggered: false, TriggerCount: 5, LastTriggered: "1 hr ago", Payload: "https://t.kavach.dev/t/def456", Description: "Link planted in internal wiki", CreatedAt: "May 25, 2026", IsActive: true},
			{ID: "4", Name: "staging-api.internal", Type: "dns", IsTriggered: false, TriggerCount: 1, LastTriggered: "3 hrs ago", Payload: "staging-api.t.kavach.dev", Description: "Fake DNS entry in hosts file", CreatedAt: "May 20, 2026", IsActive: true},
			{ID: "5", Name: "admin-creds-backup", Type: "api_key", IsTriggered: false, TriggerCount: 0, LastTriggered: "Never", Payload: "kv_live_x9y8z7w6v5u4t3s2r1q0p9o8n7m6l5k4", Description: "Planted in password manager export", CreatedAt: "Jun 5, 2026", IsActive: true},
			{ID: "6", Name: "hr-contact@trap.kavach.dev", Type: "email", IsTriggered: false, TriggerCount: 0, LastTriggered: "Never", Payload: "hr-contact@trap.kavach.dev", Description: "Fake HR email in company directory", CreatedAt: "Jun 8, 2026", IsActive: true},
			{ID: "7", Name: "office-wifi-qr", Type: "qr_code", IsTriggered: true, TriggerCount: 2, LastTriggered: "4 hrs ago", Payload: "data:image/png;base64,iVBORw0KGgo...", Description: "QR code posted in break room", CreatedAt: "Jun 9, 2026", IsActive: true},
			{ID: "8", Name: "company-login-page", Type: "cloned_site", IsTriggered: true, TriggerCount: 1, LastTriggered: "6 hrs ago", Payload: "<script>/* clone detection */</script>", Description: "Clone detection on login.company.com", CreatedAt: "Jun 7, 2026", IsActive: true},
			{ID: "9", Name: "newsletter-tracker", Type: "web_image", IsTriggered: false, TriggerCount: 8, LastTriggered: "1 hr ago", Payload: "https://t.kavach.dev/t/px9876/pixel", Description: "Tracking pixel in leaked email template", CreatedAt: "Jun 3, 2026", IsActive: true},
			{ID: "10", Name: "s3-backup-creds", Type: "aws_key", IsTriggered: true, TriggerCount: 3, LastTriggered: "30 min ago", Payload: "AWS_ACCESS_KEY_ID=[REDACTED_AWS_KEY]\n[REDACTED_AWS_SECRET]", Description: "Fake AWS keys in .env on honeypot server", CreatedAt: "Jun 6, 2026", IsActive: true},
		}
	}

	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
		Tokens     []TokenItem
	}{
		Title:      "Tokens",
		ActiveNav:  "tokens",
		AlertCount: 3,
		Tokens:     tokens,
	}

	return h.renderPage(c, "tokens/index.html", data)
}

// NewToken renders the create token form
func (h *PageHandler) NewToken(c *fiber.Ctx) error {
	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
	}{
		Title:      "Create Token",
		ActiveNav:  "tokens",
		AlertCount: 0,
	}

	return h.renderPage(c, "tokens/new.html", data)
}

// TokenDetail renders a single token detail page
func (h *PageHandler) TokenDetail(c *fiber.Ctx) error {
	id := c.Params("id")

	// Try to load token from database
	var token TokenItem
	var triggerURL string
	var lastTriggered string
	var events []EventItem
	found := false

	if h.db != nil {
		tokenID, parseErr := uuid.Parse(id)
		if parseErr == nil {
			tokenRepo := database.NewTokenRepository(h.db)
			dbToken, err := tokenRepo.GetByID(tokenID)
			if err == nil && dbToken != nil {
				lt := "Never"
				if dbToken.LastTriggered != nil {
					lt = formatTimeAgo(*dbToken.LastTriggered)
				}
				token = TokenItem{
					ID:            dbToken.ID.String(),
					Name:          dbToken.Name,
					Type:          string(dbToken.Type),
					IsTriggered:   dbToken.TriggerCount > 0,
					TriggerCount:  dbToken.TriggerCount,
					LastTriggered: lt,
					Payload:       dbToken.Payload,
					Description:   dbToken.Description,
					CreatedAt:     dbToken.CreatedAt.Format("Jan 2, 2006"),
					IsActive:      dbToken.IsActive,
				}
				triggerURL = dbToken.TriggerURL
				lastTriggered = lt
				found = true
			}
		}
	}

	// Fallback to mock data for demo mode
	if !found {
		baseURL := os.Getenv("TRIGGER_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}
		mockTokens := getMockTokenByID(id)
		if mockTokens != nil {
			token = *mockTokens
			triggerURL = baseURL + "/t/" + id
			lastTriggered = token.LastTriggered
		} else {
			// Default fallback so template doesn't crash
			token = TokenItem{
				ID:          id,
				Name:        "Unknown Token",
				Type:        "url",
				IsActive:    false,
				Description: "",
				Payload:     "",
				CreatedAt:   "Unknown",
			}
			lastTriggered = "Never"
		}
	}

	// Mock events for demo (in production these come from the events table)
	if events == nil {
		events = []EventItem{}
	}

	data := TokenDetailData{
		Title:         "Token Detail",
		ActiveNav:     "tokens",
		AlertCount:    0,
		Token:         token,
		TriggerURL:    triggerURL,
		Events:        events,
		LastTriggered: lastTriggered,
	}

	return h.renderPage(c, "tokens/detail.html", data)
}

// TokenEdit renders the token edit form
func (h *PageHandler) TokenEdit(c *fiber.Ctx) error {
	id := c.Params("id")

	// Try to load token from database
	var token TokenItem
	found := false

	if h.db != nil {
		tokenID, parseErr := uuid.Parse(id)
		if parseErr == nil {
			tokenRepo := database.NewTokenRepository(h.db)
			dbToken, err := tokenRepo.GetByID(tokenID)
			if err == nil && dbToken != nil {
				lt := "Never"
				if dbToken.LastTriggered != nil {
					lt = formatTimeAgo(*dbToken.LastTriggered)
				}
				token = TokenItem{
					ID:            dbToken.ID.String(),
					Name:          dbToken.Name,
					Type:          string(dbToken.Type),
					IsTriggered:   dbToken.TriggerCount > 0,
					TriggerCount:  dbToken.TriggerCount,
					LastTriggered: lt,
					Payload:       dbToken.Payload,
					Description:   dbToken.Description,
					CreatedAt:     dbToken.CreatedAt.Format("Jan 2, 2006"),
					IsActive:      dbToken.IsActive,
				}
				found = true
			}
		}
	}

	// Fallback to mock data for demo mode
	if !found {
		mockToken := getMockTokenByID(id)
		if mockToken != nil {
			token = *mockToken
		} else {
			// Default fallback so template doesn't crash
			token = TokenItem{
				ID:          id,
				Name:        "Unknown Token",
				Type:        "url",
				IsActive:    true,
				Description: "",
				Payload:     "",
				CreatedAt:   "Unknown",
			}
		}
	}

	data := TokenEditData{
		Title:      "Edit Token",
		ActiveNav:  "tokens",
		AlertCount: 0,
		Token:      token,
	}

	return h.renderPage(c, "tokens/edit.html", data)
}

// getMockTokenByID returns a mock token by ID for demo mode
func getMockTokenByID(id string) *TokenItem {
	mockTokens := map[string]TokenItem{
		"1":  {ID: "1", Name: "production-db-key", Type: "api_key", IsTriggered: true, TriggerCount: 12, LastTriggered: "2 min ago", Payload: "kv_live_a3f9c2b1d4e5f6a7b8c9d0e1f2a3b4c5", Description: "Planted in .env on staging server", CreatedAt: "Jun 1, 2026", IsActive: true},
		"2":  {ID: "2", Name: "financials_2026.pdf", Type: "document", IsTriggered: true, TriggerCount: 3, LastTriggered: "18 min ago", Payload: "https://t.kavach.dev/t/abc123/doc", Description: "Fake financial report in shared drive", CreatedAt: "May 28, 2026", IsActive: true},
		"3":  {ID: "3", Name: "internal-wiki-backup", Type: "url", IsTriggered: false, TriggerCount: 5, LastTriggered: "1 hr ago", Payload: "https://t.kavach.dev/t/def456", Description: "Link planted in internal wiki", CreatedAt: "May 25, 2026", IsActive: true},
		"4":  {ID: "4", Name: "staging-api.internal", Type: "dns", IsTriggered: false, TriggerCount: 1, LastTriggered: "3 hrs ago", Payload: "staging-api.t.kavach.dev", Description: "Fake DNS entry in hosts file", CreatedAt: "May 20, 2026", IsActive: true},
		"5":  {ID: "5", Name: "admin-creds-backup", Type: "api_key", IsTriggered: false, TriggerCount: 0, LastTriggered: "Never", Payload: "kv_live_x9y8z7w6v5u4t3s2r1q0p9o8n7m6l5k4", Description: "Planted in password manager export", CreatedAt: "Jun 5, 2026", IsActive: true},
		"6":  {ID: "6", Name: "hr-contact@trap.kavach.dev", Type: "email", IsTriggered: false, TriggerCount: 0, LastTriggered: "Never", Payload: "hr-contact@trap.kavach.dev", Description: "Fake HR email in company directory", CreatedAt: "Jun 8, 2026", IsActive: true},
		"7":  {ID: "7", Name: "office-wifi-qr", Type: "qr_code", IsTriggered: true, TriggerCount: 2, LastTriggered: "4 hrs ago", Payload: "data:image/png;base64,iVBORw0KGgo...", Description: "QR code posted in break room", CreatedAt: "Jun 9, 2026", IsActive: true},
		"8":  {ID: "8", Name: "company-login-page", Type: "cloned_site", IsTriggered: true, TriggerCount: 1, LastTriggered: "6 hrs ago", Payload: "<script>/* clone detection */</script>", Description: "Clone detection on login.company.com", CreatedAt: "Jun 7, 2026", IsActive: true},
		"9":  {ID: "9", Name: "newsletter-tracker", Type: "web_image", IsTriggered: false, TriggerCount: 8, LastTriggered: "1 hr ago", Payload: "https://t.kavach.dev/t/px9876/pixel", Description: "Tracking pixel in leaked email template", CreatedAt: "Jun 3, 2026", IsActive: true},
		"10": {ID: "10", Name: "s3-backup-creds", Type: "aws_key", IsTriggered: true, TriggerCount: 3, LastTriggered: "30 min ago", Payload: "AWS_ACCESS_KEY_ID=[REDACTED_AWS_KEY]\n[REDACTED_AWS_SECRET]", Description: "Fake AWS keys in .env on honeypot server", CreatedAt: "Jun 6, 2026", IsActive: true},
	}
	if t, ok := mockTokens[id]; ok {
		return &t
	}
	return nil
}

// AlertsList renders the alerts page
func (h *PageHandler) AlertsList(c *fiber.Ctx) error {
	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
		Alerts     []AlertItem
	}{
		Title:      "Alerts",
		ActiveNav:  "alerts",
		AlertCount: 3,
		Alerts: []AlertItem{
			{ID: "1", Title: "API Key token triggered - production-db-key", TokenType: "api_key", Severity: "critical", IPAddress: "103.45.67.89", Location: "Mumbai, India", Browser: "Chrome/Linux", TimeAgo: "2 min ago"},
			{ID: "2", Title: "Document token opened - financials_2026.pdf", TokenType: "document", Severity: "critical", IPAddress: "45.33.21.110", Location: "Sao Paulo, Brazil", Browser: "Firefox/Win", TimeAgo: "18 min ago"},
			{ID: "3", Title: "URL token accessed - internal-wiki-backup", TokenType: "url", Severity: "warning", IPAddress: "92.168.1.44", Location: "Berlin, Germany", Browser: "Bot/Crawler", TimeAgo: "1 hr ago"},
			{ID: "4", Title: "DNS token resolved - staging-api.internal", TokenType: "dns", Severity: "warning", IPAddress: "185.220.101.1", Location: "Tor Exit Node", Browser: "curl/7.88", TimeAgo: "3 hrs ago"},
			{ID: "5", Title: "API Key scanned - production-db-key", TokenType: "api_key", Severity: "info", IPAddress: "103.45.67.89", Location: "Mumbai, India", Browser: "Chrome/Linux", TimeAgo: "5 hrs ago"},
			{ID: "6", Title: "URL token accessed - internal-wiki-backup", TokenType: "url", Severity: "info", IPAddress: "92.168.1.44", Location: "Berlin, Germany", Browser: "Bot/Crawler", TimeAgo: "8 hrs ago"},
			{ID: "7", Title: "Document token opened - financials_2026.pdf", TokenType: "document", Severity: "critical", IPAddress: "45.33.21.110", Location: "Sao Paulo, Brazil", Browser: "Firefox/Win", TimeAgo: "12 hrs ago"},
			{ID: "8", Title: "API Key used in request - production-db-key", TokenType: "api_key", Severity: "critical", IPAddress: "103.45.67.89", Location: "Mumbai, India", Browser: "Python/requests", TimeAgo: "1 day ago"},
		},
	}

	return h.renderPage(c, "alerts/index.html", data)
}

// AttackersList renders the attackers page
func (h *PageHandler) AttackersList(c *fiber.Ctx) error {
	mockAttackers := services.GetMockAttackers()
	items := make([]AttackerItem, len(mockAttackers))
	for i, a := range mockAttackers {
		items[i] = AttackerItem{
			ID:              a.ID.String(),
			Fingerprint:     a.Fingerprint,
			IPAddress:       a.IPAddress,
			Country:         a.Country,
			City:            a.City,
			ISP:             a.ISP,
			ASN:             a.ASN,
			Browser:         a.Browser,
			OS:              a.OS,
			ThreatLevel:     string(a.ThreatLevel),
			TriggerCount:    a.TriggerCount,
			TokensTriggered: a.TokensTriggered,
			FirstSeen:       formatTimeAgo(a.FirstSeenAt),
			LastSeen:        formatTimeAgo(a.LastSeenAt),
			Notes:           a.Notes,
		}
	}

	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
		Attackers  []AttackerItem
	}{
		Title:      "Attackers",
		ActiveNav:  "attackers",
		AlertCount: 3,
		Attackers:  items,
	}

	return h.renderPage(c, "attackers/index.html", data)
}

// AttackerDetail renders a single attacker profile
func (h *PageHandler) AttackerDetail(c *fiber.Ctx) error {
	id := c.Params("id")
	mockAttackers := services.GetMockAttackers()

	var found *models.Attacker
	for i := range mockAttackers {
		if mockAttackers[i].ID.String() == id {
			found = &mockAttackers[i]
			break
		}
	}

	if found == nil {
		// Default to first attacker for demo
		if len(mockAttackers) > 0 {
			found = &mockAttackers[0]
		} else {
			return c.Status(404).SendString("Attacker not found")
		}
	}

	item := AttackerItem{
		ID:              found.ID.String(),
		Fingerprint:     found.Fingerprint,
		IPAddress:       found.IPAddress,
		Country:         found.Country,
		City:            found.City,
		ISP:             found.ISP,
		ASN:             found.ASN,
		Browser:         found.Browser,
		OS:              found.OS,
		ThreatLevel:     string(found.ThreatLevel),
		TriggerCount:    found.TriggerCount,
		TokensTriggered: found.TokensTriggered,
		FirstSeen:       found.FirstSeenAt.Format("Jan 2, 2006 3:04 PM"),
		LastSeen:        formatTimeAgo(found.LastSeenAt),
		Notes:           found.Notes,
		UserAgent:       found.UserAgent,
		TLSFingerprint:  found.TLSFingerprint,
		Events: []AlertItem{
			{Title: "Triggered production-db-key", TokenType: "api_key", Severity: "critical", IPAddress: found.IPAddress, TimeAgo: "2 min ago"},
			{Title: "Triggered internal-wiki-backup", TokenType: "url", Severity: "warning", IPAddress: found.IPAddress, TimeAgo: "1 hr ago"},
			{Title: "Triggered financials_2026.pdf", TokenType: "document", Severity: "critical", IPAddress: found.IPAddress, TimeAgo: "3 hrs ago"},
		},
	}

	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
		Attacker   AttackerItem
	}{
		Title:      "Attacker - " + found.IPAddress,
		ActiveNav:  "attackers",
		AlertCount: 3,
		Attacker:   item,
	}

	return h.renderPage(c, "attackers/detail.html", data)
}

// LoginPage renders the login form
func (h *PageHandler) LoginPage(c *fiber.Ctx) error {
	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
	}{
		Title:      "Login",
		ActiveNav:  "",
		AlertCount: 0,
	}

	return h.renderPage(c, "auth/login.html", data)
}

// SignupPage renders the signup form
func (h *PageHandler) SignupPage(c *fiber.Ctx) error {
	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
	}{
		Title:      "Sign Up",
		ActiveNav:  "",
		AlertCount: 0,
	}

	return h.renderPage(c, "auth/signup.html", data)
}

// IntegrationsPage renders the integrations settings
func (h *PageHandler) IntegrationsPage(c *fiber.Ctx) error {
	data := IntegrationData{
		Title:      "Integrations",
		ActiveNav:  "integrations",
		AlertCount: 0,
		SlackURL:   "",
		EmailTo:    "",
		EmailFrom:  "",
		IsSlackOn:  false,
		IsEmailOn:  false,
	}

	return h.renderPage(c, "integrations/index.html", data)
}

// SettingsPage renders the settings page
func (h *PageHandler) SettingsPage(c *fiber.Ctx) error {
	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
	}{
		Title:      "Settings",
		ActiveNav:  "settings",
		AlertCount: 0,
	}

	return h.renderPage(c, "settings/index.html", data)
}

// ===================== TEMPLATE RENDERING =====================

// renderPage parses and executes the layout + page template
func (h *PageHandler) renderPage(c *fiber.Ctx, page string, data interface{}) error {
	layoutPath := filepath.Join(h.templateDir, "layouts", "base.html")
	pagePath := filepath.Join(h.templateDir, page)

	tmpl, err := template.New("").Funcs(h.funcMap).ParseFiles(layoutPath, pagePath)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		return c.Status(500).SendString("Template Error: " + err.Error())
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "base.html", data)
	if err != nil {
		log.Printf("Template execute error: %v", err)
		return c.Status(500).SendString("Render Error: " + err.Error())
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// formatTimeAgo converts a time to a human-readable "X ago" string
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hr ago"
		}
		return fmt.Sprintf("%d hrs ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
