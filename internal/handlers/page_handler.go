package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/parthjindal/kavach/internal/database"
)

// PageHandler serves HTML pages using Go templates + HTMX
type PageHandler struct {
	templateDir string
	db          *sql.DB
	funcMap     template.FuncMap
}

// NewPageHandler creates a new page handler
func NewPageHandler(templateDir string, db *sql.DB) *PageHandler {
	funcMap := template.FuncMap{
		"title": func(s string) string {
			if s == "" {
				return "Low"
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
	}
	return &PageHandler{
		templateDir: templateDir,
		db:          db,
		funcMap:     funcMap,
	}
}

// ===== DATA STRUCTURES FOR TEMPLATES =====

// DashboardStats holds KPI data for the dashboard
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

// AlertItem represents a single alert event
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

// CountryItem represents attack origin data
type CountryItem struct {
	Flag       string
	Name       string
	Percentage int
}

// IPItem represents a recent IP
type IPItem struct {
	IP      string
	TimeAgo string
}

// TokenDisplayItem holds token data for template rendering
type TokenDisplayItem struct {
	ID            string
	Name          string
	Type          string
	IsActive      bool
	IsTriggered   bool
	TriggerCount  int
	LastTriggered string
	CreatedAt     string
	Payload       string
	Description   string
}

// TokenDetailData holds full token data for the detail page
type TokenDetailData struct {
	ID            string
	Name          string
	Type          string
	Description   string
	IsActive      bool
	IsTriggered   bool
	TriggerCount  int
	LastTriggered string
	CreatedAt     string
	Payload       string
	TriggerURL    string
	Events        []AlertItem
}

// AttackerDisplayItem holds attacker data for list rendering
type AttackerDisplayItem struct {
	ID              string
	Fingerprint     string
	IPAddress       string
	Country         string
	City            string
	Browser         string
	OS              string
	ThreatLevel     string
	TriggerCount    int
	TokensTriggered int
	FirstSeen       string
	LastSeen        string
}

// AttackerDetailData holds full attacker data for the detail page
type AttackerDetailData struct {
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

// DashboardData holds all data for the dashboard page
type DashboardData struct {
	Title        string
	ActiveNav    string
	AlertCount   int
	Stats        DashboardStats
	RecentAlerts []AlertItem
	TopCountries []CountryItem
	RecentIPs    []IPItem
}

// ===== PAGE HANDLERS =====

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
			{Flag: "🇮🇳", Name: "India", Percentage: 34},
			{Flag: "🇧🇷", Name: "Brazil", Percentage: 22},
			{Flag: "🇩🇪", Name: "Germany", Percentage: 18},
			{Flag: "🇺🇸", Name: "USA", Percentage: 14},
		},
		RecentIPs: []IPItem{
			{IP: "103.45.67.89", TimeAgo: "2m ago"},
			{IP: "45.33.21.110", TimeAgo: "18m ago"},
			{IP: "92.168.1.44", TimeAgo: "1h ago"},
		},
	}

	return h.renderPage(c, "dashboard/index.html", data)
}

// TokensList renders the tokens list page
func (h *PageHandler) TokensList(c *fiber.Ctx) error {
	var tokens []TokenDisplayItem

	if h.db != nil {
		tokenRepo := database.NewTokenRepository(h.db)
		var userID uuid.UUID
		row := h.db.QueryRow("SELECT id FROM users WHERE email = 'admin@kavach.dev'")
		if err := row.Scan(&userID); err == nil {
			dbTokens, err := tokenRepo.ListByUserID(userID, 100, 0)
			if err == nil && dbTokens != nil {
				baseURL := os.Getenv("TRIGGER_BASE_URL")
				if baseURL == "" {
					baseURL = "http://localhost:8080"
				}
				for _, t := range dbTokens {
					tokens = append(tokens, TokenDisplayItem{
						ID:            t.ID.String(),
						Name:          t.Name,
						Type:          string(t.Type),
						IsActive:      t.IsActive,
						IsTriggered:   t.TriggerCount > 0,
						TriggerCount:  t.TriggerCount,
						LastTriggered: formatTimePtr(t.LastTriggered),
						CreatedAt:     t.CreatedAt.Format("Jan 2, 2006"),
						Payload:       baseURL + "/t/" + t.TriggerID,
						Description:   t.Description,
					})
				}
			}
		}
	}

	// If no tokens from DB, use mock data for demo mode
	if tokens == nil {
		tokens = []TokenDisplayItem{
			{ID: "demo-1", Name: "production-db-credentials", Type: "api_key", IsActive: true, IsTriggered: true, TriggerCount: 7, LastTriggered: "2 min ago", CreatedAt: "Jun 1, 2026", Payload: "kv_live_sk_prod_a8f3n2k1x9p4", Description: "Fake DB credentials in .env"},
			{ID: "demo-2", Name: "internal-wiki-backup", Type: "url", IsActive: true, IsTriggered: false, TriggerCount: 0, LastTriggered: "Never", CreatedAt: "Jun 3, 2026", Payload: "https://kavach-hh30.onrender.com/t/demo-url", Description: "Hidden URL in wiki"},
			{ID: "demo-3", Name: "financials_2026.pdf", Type: "document", IsActive: true, IsTriggered: true, TriggerCount: 3, LastTriggered: "18 min ago", CreatedAt: "Jun 5, 2026", Payload: "https://kavach-hh30.onrender.com/t/demo-doc/doc", Description: "Fake financial report"},
		}
	}

	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
		Tokens     []TokenDisplayItem
	}{
		Title:      "Tokens",
		ActiveNav:  "tokens",
		AlertCount: 3,
		Tokens:     tokens,
	}

	return h.renderPage(c, "tokens/index.html", data)
}

// NewToken renders the create token page
func (h *PageHandler) NewToken(c *fiber.Ctx) error {
	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
	}{
		Title:      "Create Token",
		ActiveNav:  "tokens",
		AlertCount: 3,
	}
	return h.renderPage(c, "tokens/new.html", data)
}

// TokenDetail renders the token detail page
func (h *PageHandler) TokenDetail(c *fiber.Ctx) error {
	tokenID := c.Params("id")

	var token TokenDetailData

	if h.db != nil {
		parsedID, err := uuid.Parse(tokenID)
		if err == nil {
			tokenRepo := database.NewTokenRepository(h.db)
			t, err := tokenRepo.GetByID(parsedID)
			if err == nil && t != nil {
				baseURL := os.Getenv("TRIGGER_BASE_URL")
				if baseURL == "" {
					baseURL = "http://localhost:8080"
				}
				token = TokenDetailData{
					ID:            t.ID.String(),
					Name:          t.Name,
					Type:          string(t.Type),
					Description:   t.Description,
					IsActive:      t.IsActive,
					IsTriggered:   t.TriggerCount > 0,
					TriggerCount:  t.TriggerCount,
					LastTriggered: formatTimePtr(t.LastTriggered),
					CreatedAt:     t.CreatedAt.Format("Jan 2, 2006 3:04 PM"),
					Payload:       baseURL + "/t/" + t.TriggerID,
					TriggerURL:    baseURL + "/t/" + t.TriggerID,
					Events:        []AlertItem{},
				}
			}
		}
	}

	// Mock data if not found
	if token.ID == "" {
		token = TokenDetailData{
			ID:            tokenID,
			Name:          "production-db-credentials",
			Type:          "api_key",
			Description:   "Fake database credentials deployed in production .env file",
			IsActive:      true,
			IsTriggered:   true,
			TriggerCount:  7,
			LastTriggered: "2 min ago",
			CreatedAt:     "Jun 1, 2026 9:30 AM",
			Payload:       "kv_live_sk_prod_a8f3n2k1x9p4",
			TriggerURL:    "https://kavach-hh30.onrender.com/t/demo-token",
			Events: []AlertItem{
				{ID: "e1", Title: "Token accessed from Mumbai", Severity: "critical", IPAddress: "103.45.67.89", Location: "Mumbai, India", Browser: "Chrome 125", TimeAgo: "2 min ago"},
				{ID: "e2", Title: "Token accessed from São Paulo", Severity: "warning", IPAddress: "45.33.21.110", Location: "São Paulo, Brazil", Browser: "Firefox 126", TimeAgo: "4 hr ago"},
			},
		}
	}

	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
		Token      TokenDetailData
	}{
		Title:      token.Name,
		ActiveNav:  "tokens",
		AlertCount: 3,
		Token:      token,
	}

	return h.renderPage(c, "tokens/detail.html", data)
}

// TokenEdit renders the token edit page
func (h *PageHandler) TokenEdit(c *fiber.Ctx) error {
	tokenID := c.Params("id")

	var token TokenDetailData

	if h.db != nil {
		parsedID, err := uuid.Parse(tokenID)
		if err == nil {
			tokenRepo := database.NewTokenRepository(h.db)
			t, err := tokenRepo.GetByID(parsedID)
			if err == nil && t != nil {
				token = TokenDetailData{
					ID:          t.ID.String(),
					Name:        t.Name,
					Type:        string(t.Type),
					Description: t.Description,
					IsActive:    t.IsActive,
				}
			}
		}
	}

	// Mock data if not found
	if token.ID == "" {
		token = TokenDetailData{
			ID:          tokenID,
			Name:        "production-db-credentials",
			Type:        "api_key",
			Description: "Fake database credentials deployed in production .env file",
			IsActive:    true,
		}
	}

	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
		Token      TokenDetailData
	}{
		Title:      "Edit " + token.Name,
		ActiveNav:  "tokens",
		AlertCount: 3,
		Token:      token,
	}

	return h.renderPage(c, "tokens/edit.html", data)
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
			{ID: "4", Title: "DNS token resolved - internal-api.corp.local", TokenType: "dns", Severity: "info", IPAddress: "185.220.101.42", Location: "Unknown (Tor)", Browser: "curl", TimeAgo: "3 hr ago"},
			{ID: "5", Title: "QR Code scanned - server-room-access", TokenType: "qr_code", Severity: "critical", IPAddress: "103.45.67.92", Location: "Mumbai, India", Browser: "Safari/iOS", TimeAgo: "5 hr ago"},
			{ID: "6", Title: "Cloned site detected - login.company.com", TokenType: "cloned_site", Severity: "critical", IPAddress: "45.33.21.115", Location: "Sao Paulo, Brazil", Browser: "Chrome/Win", TimeAgo: "8 hr ago"},
			{ID: "7", Title: "Email token triggered - ceo@internal.corp", TokenType: "email", Severity: "warning", IPAddress: "92.168.1.55", Location: "Berlin, Germany", Browser: "Thunderbird", TimeAgo: "12 hr ago"},
			{ID: "8", Title: "AWS key used in us-east-1", TokenType: "aws_key", Severity: "info", IPAddress: "54.23.45.67", Location: "Virginia, USA", Browser: "AWS CLI", TimeAgo: "1 day ago"},
		},
	}

	return h.renderPage(c, "alerts/index.html", data)
}

// AttackersList renders the attackers list page
func (h *PageHandler) AttackersList(c *fiber.Ctx) error {
	var attackers []AttackerDisplayItem

	// Mock data for demo
	attackers = []AttackerDisplayItem{
		{ID: "a1", Fingerprint: "fp_3a7f2b9c1e8d", IPAddress: "103.45.67.89", Country: "IN", City: "Mumbai", Browser: "Chrome 125", OS: "Linux", ThreatLevel: "high", TriggerCount: 14, TokensTriggered: 4, FirstSeen: "Jun 1, 2026", LastSeen: "2 min ago"},
		{ID: "a2", Fingerprint: "fp_8b2c4d6e1f3a", IPAddress: "45.33.21.110", Country: "BR", City: "Sao Paulo", Browser: "Firefox 126", OS: "Windows", ThreatLevel: "medium", TriggerCount: 6, TokensTriggered: 2, FirstSeen: "Jun 3, 2026", LastSeen: "18 min ago"},
		{ID: "a3", Fingerprint: "fp_tor_exit_9x2", IPAddress: "185.220.101.42", Country: "XX", City: "Unknown", Browser: "curl", OS: "Linux", ThreatLevel: "critical", TriggerCount: 31, TokensTriggered: 8, FirstSeen: "May 28, 2026", LastSeen: "3 hr ago"},
	}

	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
		Attackers  []AttackerDisplayItem
	}{
		Title:      "Attackers",
		ActiveNav:  "attackers",
		AlertCount: 3,
		Attackers:  attackers,
	}

	return h.renderPage(c, "attackers/index.html", data)
}

// AttackerDetail renders the attacker detail page
func (h *PageHandler) AttackerDetail(c *fiber.Ctx) error {
	attackerID := c.Params("id")

	attacker := AttackerDetailData{
		ID:              attackerID,
		Fingerprint:     "fp_3a7f2b9c1e8d",
		IPAddress:       "103.45.67.89",
		Country:         "India",
		City:            "Mumbai",
		ISP:             "Jio Fiber",
		ASN:             "AS55836",
		Browser:         "Chrome 125",
		OS:              "Linux (Ubuntu)",
		ThreatLevel:     "high",
		TriggerCount:    14,
		TokensTriggered: 4,
		FirstSeen:       "Jun 1, 2026 08:14 AM",
		LastSeen:        "2 min ago",
		Notes:           "",
		UserAgent:       "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
		TLSFingerprint:  "ja3_771,4865-49199",
		Events: []AlertItem{
			{ID: "e1", Title: "API Key triggered - production-db-key", Severity: "critical", IPAddress: "103.45.67.89", Location: "Mumbai, India", Browser: "Chrome/Linux", TimeAgo: "2 min ago"},
			{ID: "e2", Title: "Document opened - financials_2026.pdf", Severity: "critical", IPAddress: "103.45.67.89", Location: "Mumbai, India", Browser: "Chrome/Linux", TimeAgo: "45 min ago"},
			{ID: "e3", Title: "URL accessed - internal-wiki", Severity: "warning", IPAddress: "103.45.67.89", Location: "Mumbai, India", Browser: "Chrome/Linux", TimeAgo: "2 hr ago"},
		},
	}

	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
		Attacker   AttackerDetailData
	}{
		Title:      "Attacker " + attacker.Fingerprint,
		ActiveNav:  "attackers",
		AlertCount: 3,
		Attacker:   attacker,
	}

	return h.renderPage(c, "attackers/detail.html", data)
}

// LoginPage renders the login page
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

// SignupPage renders the signup page
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

// IntegrationsPage renders the integrations settings page
func (h *PageHandler) IntegrationsPage(c *fiber.Ctx) error {
	data := struct {
		Title      string
		ActiveNav  string
		AlertCount int
	}{
		Title:      "Integrations",
		ActiveNav:  "integrations",
		AlertCount: 0,
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

// renderPage renders a template with the base layout
func (h *PageHandler) renderPage(c *fiber.Ctx, templateName string, data interface{}) error {
	basePath := filepath.Join(h.templateDir, "layouts", "base.html")
	contentPath := filepath.Join(h.templateDir, templateName)

	tmpl, err := template.New("").Funcs(h.funcMap).ParseFiles(basePath, contentPath)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		return c.Status(500).Type("html").SendString(`<html><body style="background:#0A0A14;color:#e2e8f0;font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0"><div style="text-align:center"><h1 style="font-size:24px;margin-bottom:8px">Something went wrong</h1><p style="color:#64748B">Please try again or contact support.</p><a href="/app" style="color:#7C3AED;margin-top:16px;display:inline-block">← Back to Dashboard</a></div></body></html>`)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "base.html", data)
	if err != nil {
		log.Printf("Template execute error: %v", err)
		return c.Status(500).Type("html").SendString(`<html><body style="background:#0A0A14;color:#e2e8f0;font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0"><div style="text-align:center"><h1 style="font-size:24px;margin-bottom:8px">Something went wrong</h1><p style="color:#64748B">Please try again or contact support.</p><a href="/app" style="color:#7C3AED;margin-top:16px;display:inline-block">← Back to Dashboard</a></div></body></html>`)
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// formatTimePtr handles *time.Time (nil = "Never")
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "Never"
	}
	return formatTimeAgo(*t)
}

// formatTimeAgo formats a time as a relative time string
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	dur := time.Since(t)
	switch {
	case dur < time.Minute:
		return "Just now"
	case dur < time.Hour:
		m := int(dur.Minutes())
		if m == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", m)
	case dur < 24*time.Hour:
		h := int(dur.Hours())
		if h == 1 {
			return "1 hr ago"
		}
		return fmt.Sprintf("%d hrs ago", h)
	default:
		d := int(dur.Hours() / 24)
		if d == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", d)
	}
}
