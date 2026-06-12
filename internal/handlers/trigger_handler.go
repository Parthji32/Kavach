package handlers

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/parthjindal/kavach/internal/alerts"
	"github.com/parthjindal/kavach/internal/database"
	"github.com/parthjindal/kavach/internal/fingerprint"
	"github.com/parthjindal/kavach/internal/models"
	"github.com/parthjindal/kavach/internal/services"
)

// TriggerHandler handles incoming token trigger requests
type TriggerHandler struct {
	tokenRepo       *database.TokenRepository
	eventRepo       *database.EventRepository
	geoService      *services.GeoService
	attackerService *services.AttackerService
	alertService    *alerts.AlertService
}

// NewTriggerHandler creates a new trigger handler
func NewTriggerHandler() *TriggerHandler {
	geoService := services.NewGeoService()
	return &TriggerHandler{
		geoService:      geoService,
		attackerService: services.NewAttackerService(geoService),
		alertService:    alerts.NewAlertService(),
	}
}

// NewTriggerHandlerWithDB creates a trigger handler with database repositories
func NewTriggerHandlerWithDB(tokenRepo *database.TokenRepository, eventRepo *database.EventRepository) *TriggerHandler {
	geoService := services.NewGeoService()
	return &TriggerHandler{
		tokenRepo:       tokenRepo,
		eventRepo:       eventRepo,
		geoService:      geoService,
		attackerService: services.NewAttackerService(geoService),
		alertService:    alerts.NewAlertService(),
	}
}

// HandleTrigger processes a token trigger event
// This is the CRITICAL path — when an attacker touches a canary token,
// this endpoint captures everything about them.
//
// Route: GET /t/:triggerID
func (h *TriggerHandler) HandleTrigger(c *fiber.Ctx) error {
	triggerID := c.Params("triggerID")

	log.Printf("🚨 TOKEN TRIGGERED: %s at %s", triggerID, time.Now().Format(time.RFC3339))

	// 1. Capture the fingerprint from the request
	fp := captureFromFiber(c)

	// 2. Log fingerprint (redact sensitive headers)
	logFp := *fp
	safeHeaders := make(map[string]string)
	for k, v := range fp.Headers {
		if k != "Cookie" && k != "Authorization" && k != "X-Access-Key" {
			safeHeaders[k] = v
		}
	}
	logFp.Headers = safeHeaders
	fpJSON, _ := json.Marshal(logFp)
	log.Printf("📋 Fingerprint: %s", string(fpJSON))

	// 3. Look up which token this trigger belongs to
	var token *models.Token
	if h.tokenRepo != nil {
		var err error
		token, err = h.tokenRepo.GetByTriggerID(triggerID)
		if err != nil {
			log.Printf("⚠️  Error looking up token for trigger %s: %v", triggerID, err)
		}
		if token == nil {
			log.Printf("⚠️  No active token found for trigger ID: %s", triggerID)
		}
	} else {
		log.Printf("ℹ️  No database connected — running in demo mode. Token lookup skipped.")
	}

	// 4. Enrich with geo data
	geo, err := h.geoService.Lookup(fp.IPAddress)
	if err != nil {
		log.Printf("⚠️  Geo lookup failed for %s: %v", fp.IPAddress, err)
	} else if geo != nil {
		fp.Country = geo.Country
		fp.City = geo.City
		fp.Region = geo.Region
		fp.ISP = geo.ISP
		fp.ASN = geo.ASN
		fp.IsVPN = geo.IsVPN
		fp.IsTor = geo.IsTor
		fp.IsProxy = geo.IsProxy
	}

	// Parse browser/OS from user agent
	fp.Browser, fp.BrowserVer = fingerprint.ParseUserAgentBrowser(fp.UserAgent)
	fp.OS, fp.OSVersion = fingerprint.ParseUserAgentOS(fp.UserAgent)

	// Generate unique hash
	fp.UniqueHash = fp.GenerateHash()

	// 5. Find or create attacker profile based on fingerprint
	attacker, err := h.attackerService.FindOrCreate(fp)
	if err != nil {
		log.Printf("⚠️  Attacker correlation failed: %v", err)
	} else {
		log.Printf("👤 Attacker: %s (threat: %s, triggers: %d)", attacker.Fingerprint, attacker.ThreatLevel, attacker.TriggerCount)
	}

	// 6. Record the trigger event (if DB available)
	if h.eventRepo != nil && token != nil && attacker != nil {
		headersJSON, _ := json.Marshal(fp.Headers)
		event := &models.TriggerEvent{
			ID:          uuid.New(),
			TokenID:     token.ID,
			AttackerID:  attacker.ID,
			IPAddress:   fp.IPAddress,
			UserAgent:   fp.UserAgent,
			Referrer:    fp.Referrer,
			Country:     fp.Country,
			City:        fp.City,
			Fingerprint: fp.UniqueHash,
			Headers:     string(headersJSON),
			Metadata:    "{}",
			CreatedAt:   time.Now(),
		}
		if err := h.eventRepo.Create(event); err != nil {
			log.Printf("⚠️  Failed to save trigger event: %v", err)
		} else {
			log.Printf("💾 Trigger event saved: %s", event.ID)
		}
	} else {
		log.Printf("ℹ️  Trigger event not saved (demo mode or missing data)")
	}

	// 7. Increment token trigger count (if DB available)
	if h.tokenRepo != nil && token != nil {
		if err := h.tokenRepo.IncrementTriggerCount(token.ID); err != nil {
			log.Printf("⚠️  Failed to increment trigger count: %v", err)
		}
	}

	// 8. Dispatch alert (email, Slack, webhook)
	if token != nil && h.alertService != nil {
		h.alertService.Dispatch(alerts.AlertPayload{
			Token:       token,
			Fingerprint: fp,
			TriggeredAt: time.Now(),
		})
	} else {
		log.Printf("ℹ️  Alert dispatch skipped (no token resolved or alert service unavailable)")
	}

	// 9. Return a response that doesn't reveal this is a honeypot
	tokenType := c.Params("type") // doc, key, dns, email, or empty (url)

	switch tokenType {
	case "doc":
		// Return a 1x1 transparent pixel (tracking pixel in documents)
		return c.Status(200).
			Type("image/gif").
			Send(transparentGIF())

	case "key":
		// Return a realistic API error (looks like a real API)
		return c.Status(401).JSON(fiber.Map{
			"error":   "invalid_api_key",
			"message": "The API key provided is invalid or expired.",
		})

	case "dns":
		// DNS tokens are handled differently (via DNS server)
		return c.Status(200).SendString("OK")

	case "email":
		// Email token confirmation
		return c.Status(200).SendString("OK")

	case "qr":
		// QR code token — same as URL (capture fingerprint, return 404)
		return c.Status(404).JSON(fiber.Map{
			"error":   "not_found",
			"message": "The requested resource could not be found.",
		})

	case "clone":
		// Cloned website detection — accept beacon POST/GET from JS snippet
		log.Printf("🚨 CLONED SITE DETECTED: data=%s", c.Body())
		return c.Status(204).SendString("")

	case "pixel":
		// Web image / tracking pixel — return 1x1 transparent GIF
		return c.Status(200).
			Type("image/gif").
			Send(transparentGIF())

	case "aws":
		// AWS API key token — return realistic AWS error JSON
		return c.Status(403).JSON(fiber.Map{
			"__type":  "InvalidClientTokenId",
			"message": "The security token included in the request is invalid.",
		})

	default:
		// URL token — serve a realistic-looking page or redirect
		return c.Status(404).JSON(fiber.Map{
			"error":   "not_found",
			"message": "The requested resource could not be found.",
		})
	}
}

// captureFromFiber extracts fingerprint data from a Fiber context
func captureFromFiber(c *fiber.Ctx) *fingerprint.CapturedFingerprint {
	fp := &fingerprint.CapturedFingerprint{
		IPAddress:  c.IP(),
		UserAgent:  c.Get("User-Agent"),
		AcceptLang: c.Get("Accept-Language"),
		AcceptEnc:  c.Get("Accept-Encoding"),
		Referrer:   c.Get("Referer"),
		Headers:    make(map[string]string),
	}

	// Capture all request headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		fp.Headers[string(key)] = string(value)
	})

	// Extract real IP (considering Cloudflare, proxies)
	if cfIP := c.Get("CF-Connecting-IP"); cfIP != "" {
		fp.IPAddress = cfIP
	} else if realIP := c.Get("X-Real-IP"); realIP != "" {
		fp.IPAddress = realIP
	} else if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
		// Split by comma and take the first IP (the original client)
		parts := strings.Split(forwarded, ",")
		fp.IPAddress = strings.TrimSpace(parts[0])
	}

	return fp
}

// transparentGIF returns a 1x1 transparent GIF (tracking pixel)
func transparentGIF() []byte {
	return []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // Header: GIF89a
		0x01, 0x00, 0x01, 0x00,             // 1x1 pixel
		0x80, 0x00, 0x00,                   // GCT flag
		0xff, 0xff, 0xff,                   // Background: white
		0x00, 0x00, 0x00,                   // Black (unused)
		0x21, 0xf9, 0x04,                   // Graphic control ext
		0x01, 0x00, 0x00, 0x00, 0x00,       // Transparent
		0x2c, 0x00, 0x00, 0x00, 0x00,       // Image descriptor
		0x01, 0x00, 0x01, 0x00, 0x00,       // 1x1
		0x02, 0x02, 0x44, 0x01, 0x00,       // Image data
		0x3b,                               // Trailer
	}
}

// HandleDemoTrigger processes the landing page demo trigger.
// It captures a real fingerprint and returns JSON so the landing page
// can display actual server-side intelligence to the visitor.
//
// Route: GET /t/demo
func (h *TriggerHandler) HandleDemoTrigger(c *fiber.Ctx) error {
	log.Printf("🎯 DEMO TRIGGER from %s", c.IP())

	// 1. Capture the fingerprint from the request
	fp := captureFromFiber(c)

	// 2. Enrich with geo data
	geo, err := h.geoService.Lookup(fp.IPAddress)
	if err != nil {
		log.Printf("⚠️  Demo geo lookup failed for %s: %v", fp.IPAddress, err)
	} else if geo != nil {
		fp.Country = geo.Country
		fp.City = geo.City
		fp.Region = geo.Region
		fp.ISP = geo.ISP
		fp.ASN = geo.ASN
		fp.IsVPN = geo.IsVPN
		fp.IsTor = geo.IsTor
		fp.IsProxy = geo.IsProxy
	}

	// Parse browser/OS from user agent
	fp.Browser, fp.BrowserVer = fingerprint.ParseUserAgentBrowser(fp.UserAgent)
	fp.OS, fp.OSVersion = fingerprint.ParseUserAgentOS(fp.UserAgent)

	// Generate unique hash
	fp.UniqueHash = fp.GenerateHash()

	// 3. Partially redact IP for privacy (show first two octets)
	redactedIP := redactIP(fp.IPAddress)

	// 4. Return JSON response for the landing page
	return c.JSON(fiber.Map{
		"ip":          redactedIP,
		"city":        fp.City,
		"region":      fp.Region,
		"country":     fp.Country,
		"isp":         fp.ISP,
		"asn":         fp.ASN,
		"browser":     fp.Browser,
		"browser_ver": fp.BrowserVer,
		"os":          fp.OS,
		"os_ver":      fp.OSVersion,
		"is_vpn":      fp.IsVPN,
		"is_tor":      fp.IsTor,
		"is_proxy":    fp.IsProxy,
		"fingerprint": fp.UniqueHash,
		"tls_version": fp.TLSVersion,
		"accept_lang": fp.AcceptLang,
		"captured_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// redactIP partially masks an IP address for display (e.g., "103.45.●●.●●")
func redactIP(ip string) string {
	if ip == "" {
		return "●●●.●●.●●●.●●"
	}
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + ".●●.●●"
	}
	// IPv6 or unusual format — just mask the last half
	if len(ip) > 8 {
		return ip[:len(ip)/2] + "●●●●"
	}
	return "●●●.●●.●●●.●●"
}
