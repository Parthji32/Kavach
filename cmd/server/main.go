package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/parthjindal/kavach/internal/database"
	"github.com/parthjindal/kavach/internal/handlers"
	"github.com/parthjindal/kavach/internal/middleware"
	"github.com/parthjindal/kavach/internal/models"
	"github.com/parthjindal/kavach/internal/services"
	"github.com/google/uuid"
)

func main() {
	godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := database.Connect()
	if err != nil {
		log.Printf("Database not connected: %v", err)
		log.Println("   Running in demo mode with mock data")
		log.Println("   Set DATABASE_URL in .env to connect to PostgreSQL")
	}
	defer database.Close()

	app := fiber.New(fiber.Config{
		AppName:               "Kavach",
		ServerHeader:          "",
		DisableStartupMessage: false,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${ip} - ${latency}\n",
	}))

	// CORS: restrict to allowed origins (fix CRITICAL issue #15)
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:8080"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,HX-Request,HX-Target,HX-Trigger",
		AllowCredentials: true,
	}))

	// CSRF protection: validate Origin header on state-changing requests (fix HIGH issue #17)
	app.Use(csrfProtection(allowedOrigins))

	triggerHandler := handlers.NewTriggerHandler()
	pageHandler := handlers.NewPageHandler("./templates")

	// Private access gate: if KAVACH_ACCESS_KEY is set, gate all non-trigger routes
	app.Use(accessGate())

	// ===== HTML PAGE ROUTES =====
	app.Get("/", pageHandler.Dashboard)
	app.Get("/login", pageHandler.LoginPage)
	app.Get("/signup", pageHandler.SignupPage)
	app.Get("/tokens", pageHandler.TokensList)
	app.Get("/tokens/new", pageHandler.NewToken)
	app.Get("/alerts", pageHandler.AlertsList)
	app.Get("/attackers", pageHandler.AttackersList)
	app.Get("/attackers/:id", pageHandler.AttackerDetail)
	app.Get("/integrations", pageHandler.IntegrationsPage)
	app.Get("/settings", pageHandler.SettingsPage)

	// ===== TOKEN TRIGGER ROUTES (PUBLIC) =====
	trigger := app.Group("/t")
	trigger.Get("/:triggerID", triggerHandler.HandleTrigger)
	trigger.Get("/:triggerID/:type", triggerHandler.HandleTrigger)

	// ===== AUTH API (with rate limiting — fix MEDIUM issue #14) =====
	auth := app.Group("/api/v1/auth")
	auth.Use(limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error":   "rate_limited",
				"message": "Too many requests. Please try again in a minute.",
			})
		},
	}))
	if db != nil {
		authHandler := handlers.NewAuthHandler(db)
		auth.Post("/signup", authHandler.Signup)
		auth.Post("/login", authHandler.Login)
		auth.Post("/logout", authHandler.Logout)
		auth.Get("/me", middleware.AuthRequired(), authHandler.Me)
	} else {
		auth.Post("/signup", func(c *fiber.Ctx) error {
			return c.Status(503).JSON(fiber.Map{"error": "Database not configured. Set DATABASE_URL in .env"})
		})
		auth.Post("/login", func(c *fiber.Ctx) error {
			return c.Status(503).JSON(fiber.Map{"error": "Database not configured. Set DATABASE_URL in .env"})
		})
	}

	// ===== REST API =====
	api := app.Group("/api/v1")

	// Apply auth middleware to protected API routes when DB is connected (fix HIGH issue #13)
	if db != nil {
		api.Use(middleware.AuthRequired())
	}

	api.Get("/tokens", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"tokens": []interface{}{}, "message": "Connect database to see real tokens"})
	})
	api.Post("/tokens", handleTokenCreate)
	api.Get("/tokens/:id", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"token": nil})
	})
	api.Delete("/tokens/:id", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "token deleted"})
	})

	api.Get("/alerts/feed", handleAlertFeed)
	api.Get("/alerts", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"alerts": []interface{}{}})
	})

	api.Get("/attackers", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"attackers": []interface{}{}})
	})
	api.Get("/attackers/:id", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"attacker": nil})
	})

	api.Get("/stats", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"active_tokens":    24,
			"triggers_today":   7,
			"unique_attackers": 12,
			"threat_level":     "medium",
		})
	})

	api.Post("/integrations/slack", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Slack webhook saved (demo mode)"})
	})
	api.Post("/integrations/email", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Email settings saved (demo mode)"})
	})
	api.Post("/integrations/webhook", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Webhook saved (demo mode)"})
	})

	// ===== HEALTH CHECK =====
	app.Get("/health", func(c *fiber.Ctx) error {
		dbStatus := "disconnected"
		if db != nil {
			if err := db.Ping(); err == nil {
				dbStatus = "connected"
			}
		}
		return c.JSON(fiber.Map{
			"status":   "healthy",
			"service":  "kavach",
			"version":  "0.1.0",
			"database": dbStatus,
		})
	})

	// ===== STATIC FILES =====
	app.Static("/static", "./static")

	// Start
	addr := fmt.Sprintf(":%s", port)
	fmt.Println("")
	fmt.Println("  Kavach - Armor That Fights Back")
	fmt.Println("  ------------------------------------")
	fmt.Printf("  Dashboard:      http://localhost%s/\n", addr)
	fmt.Printf("  API:            http://localhost%s/api/v1/\n", addr)
	fmt.Printf("  Token triggers: http://localhost%s/t/{id}\n", addr)
	fmt.Printf("  Health:         http://localhost%s/health\n", addr)
	fmt.Println("  ------------------------------------")
	if db != nil {
		fmt.Println("  Database:       Connected")
	} else {
		fmt.Println("  Database:       Demo mode (set DATABASE_URL)")
	}
	fmt.Println("")

	if err := app.Listen(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// csrfProtection validates Origin header on state-changing requests
func csrfProtection(allowedOrigins string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only check state-changing methods
		method := c.Method()
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return c.Next()
		}

		// Skip CSRF for trigger routes (they're public honeypot endpoints)
		if len(c.Path()) > 2 && c.Path()[:3] == "/t/" {
			return c.Next()
		}

		origin := c.Get("Origin")
		// If no Origin header, check Referer as fallback
		if origin == "" {
			origin = c.Get("Referer")
			if origin == "" {
				// Allow requests with no Origin (e.g., same-origin, curl, etc.)
				return c.Next()
			}
		}

		// Check if origin is in the allowed list
		allowed := false
		for _, o := range splitOrigins(allowedOrigins) {
			normalizedOrigin := strings.TrimRight(origin, "/")
			normalizedAllowed := strings.TrimRight(o, "/")
			if normalizedOrigin == normalizedAllowed {
				allowed = true
				break
			}
		}

		if !allowed {
			return c.Status(403).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Cross-origin request blocked",
			})
		}

		return c.Next()
	}
}

// accessGate implements maintenance mode / private access control.
// If KAVACH_ACCESS_KEY is set, all routes except /t/* (trigger routes) require
// a matching ?key= query param or kavach_access cookie.
func accessGate() fiber.Handler {
	accessKey := os.Getenv("KAVACH_ACCESS_KEY")
	return func(c *fiber.Ctx) error {
		// If no access key configured, everything is public
		if accessKey == "" {
			return c.Next()
		}

		// Trigger routes are ALWAYS public (honeypot endpoints)
		path := c.Path()
		if strings.HasPrefix(path, "/t/") {
			return c.Next()
		}

		// Static assets should pass through for the coming-soon page to render
		if strings.HasPrefix(path, "/static/") {
			return c.Next()
		}

		// Health check stays public for monitoring
		if path == "/health" {
			return c.Next()
		}

		// Check query param
		if c.Query("key") == accessKey {
			// Set cookie so user doesn't need ?key= on every request
			c.Cookie(&fiber.Cookie{
				Name:     "kavach_access",
				Value:    accessKey,
				HTTPOnly: true,
				SameSite: "Lax",
				Expires:  time.Now().Add(24 * time.Hour),
			})
			return c.Next()
		}

		// Check cookie
		if c.Cookies("kavach_access") == accessKey {
			return c.Next()
		}

		// Block with "Coming Soon" page
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Status(503).SendString(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Coming Soon — Kavach</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{min-height:100vh;display:flex;align-items:center;justify-content:center;background:#0a0a0f;color:#e2e8f0;font-family:system-ui,sans-serif}
.container{text-align:center;padding:2rem}.logo{font-size:2.5rem;font-weight:700;background:linear-gradient(135deg,#06b6d4,#8b5cf6);-webkit-background-clip:text;-webkit-text-fill-color:transparent;margin-bottom:1rem}
.subtitle{color:#94a3b8;font-size:1.1rem;margin-bottom:2rem}
.badge{display:inline-block;padding:.4rem 1rem;border-radius:9999px;background:rgba(6,182,212,0.1);border:1px solid rgba(6,182,212,0.3);color:#06b6d4;font-size:.85rem}</style></head>
<body><div class="container"><div class="logo">Kavach</div><p class="subtitle">Armor That Fights Back</p><span class="badge">Coming Soon</span></div></body></html>`)
	}
}

// splitOrigins splits a comma-separated origins string
func splitOrigins(origins string) []string {
	var result []string
	for _, o := range splitByComma(origins) {
		trimmed := trimSpace(o)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitByComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && s[start] == ' ' {
		start++
	}
	end := len(s)
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}

// handleAlertFeed returns an HTML partial with mock alert data for HTMX polling (fix MEDIUM issue #4/#5)
func handleAlertFeed(c *fiber.Ctx) error {
	type alertData struct {
		Title     string
		TokenType string
		Severity  string
		IPAddress string
		Location  string
		Browser   string
		TimeAgo   string
	}

	alerts := []alertData{
		{Title: "API Key token triggered - production-db-key", TokenType: "api_key", Severity: "critical", IPAddress: "103.45.67.89", Location: "Mumbai, India", Browser: "Chrome/Linux", TimeAgo: "2 min ago"},
		{Title: "Document token opened - financials_2026.pdf", TokenType: "document", Severity: "critical", IPAddress: "45.33.21.110", Location: "Sao Paulo, Brazil", Browser: "Firefox/Win", TimeAgo: "18 min ago"},
		{Title: "URL token accessed - internal-wiki-backup", TokenType: "url", Severity: "warning", IPAddress: "92.168.1.44", Location: "Berlin, Germany", Browser: "Bot/Crawler", TimeAgo: "1 hr ago"},
	}

	const tmplStr = `{{range .}}<div class="flex items-start gap-3 p-3 rounded-lg bg-white/[0.02] border border-white/5 hover:border-kavach-accent/20 transition">
    <div class="mt-0.5 w-2 h-2 rounded-full shrink-0 {{if eq .Severity "critical"}}bg-red-400{{else if eq .Severity "warning"}}bg-amber-400{{else}}bg-blue-400{{end}}"></div>
    <div class="flex-1 min-w-0">
        <p class="text-sm text-gray-200 font-medium truncate">{{.Title}}</p>
        <div class="flex items-center gap-3 mt-1.5">
            <span class="text-[11px] text-gray-500">{{.IPAddress}}</span>
            <span class="text-[11px] text-gray-500">{{.Location}}</span>
            <span class="text-[11px] text-gray-500">{{.Browser}}</span>
        </div>
    </div>
    <span class="text-[11px] text-gray-500 shrink-0">{{.TimeAgo}}</span>
</div>
{{end}}`

	tmpl, err := template.New("alert-feed").Parse(tmplStr)
	if err != nil {
		log.Printf("Alert feed template error: %v", err)
		return c.SendString("")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, alerts); err != nil {
		log.Printf("Alert feed render error: %v", err)
		return c.SendString("")
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// handleTokenCreate handles POST /api/v1/tokens
// Returns HTML partial for HTMX requests, JSON for API requests
func handleTokenCreate(c *fiber.Ctx) error {
	// Parse token creation request
	type tokenReq struct {
		Name        string `json:"name" form:"name"`
		Type        string `json:"type" form:"type"`
		Description string `json:"description" form:"description"`
		Domain      string `json:"domain" form:"domain"`
	}

	var req tokenReq
	if err := c.BodyParser(&req); err != nil {
		if c.Get("HX-Request") == "true" {
			return c.SendString(`<div class="text-red-400 text-sm p-3">Invalid request. Please fill all fields.</div>`)
		}
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Name == "" || req.Type == "" {
		if c.Get("HX-Request") == "true" {
			return c.SendString(`<div class="text-red-400 text-sm p-3">Token name and type are required.</div>`)
		}
		return c.Status(400).JSON(fiber.Map{"error": "name and type are required"})
	}

	// Use the real token service to generate the token
	baseURL := os.Getenv("TRIGGER_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	tokenSvc := services.NewTokenService(baseURL)

	createReq := models.CreateTokenRequest{
		Name:        req.Name,
		Type:        models.TokenType(req.Type),
		Description: req.Description,
		Domain:      req.Domain,
	}

	generatedToken, err := tokenSvc.GenerateToken(uuid.New(), createReq)
	if err != nil {
		if c.Get("HX-Request") == "true" {
			return c.SendString(`<div class="text-red-400 text-sm p-3">Failed to generate token: ` + err.Error() + `</div>`)
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate token", "details": err.Error()})
	}

	// Check if request is from HTMX — return HTML partial
	if c.Get("HX-Request") == "true" {
		type tokenData struct {
			Name    string
			Type    string
			Payload string
		}
		data := struct {
			Token tokenData
		}{
			Token: tokenData{
				Name:    generatedToken.Name,
				Type:    string(generatedToken.Type),
				Payload: generatedToken.Payload,
			},
		}

		tmpl, err := template.ParseFiles("./templates/tokens/token_created.html")
		if err != nil {
			log.Printf("Token created template error: %v", err)
			return c.SendString(`<div class="bg-kavach-accent/10 border border-kavach-accent/30 rounded-xl p-5"><p class="text-kavach-accent font-semibold mb-2">Token Created!</p><p class="text-sm text-gray-300">Name: ` + generatedToken.Name + `</p><p class="text-sm text-gray-300 mt-1">Type: ` + string(generatedToken.Type) + `</p><pre class="mt-2 text-xs text-kavach-accent bg-kavach-dark border border-kavach-border rounded-lg p-3 overflow-x-auto">` + generatedToken.Payload + `</pre></div>`)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			log.Printf("Token created template render error: %v", err)
			return c.SendString(`<div class="bg-kavach-accent/10 border border-kavach-accent/30 rounded-xl p-5"><p class="text-kavach-accent font-semibold mb-2">Token Created!</p><p class="text-sm text-gray-300">Name: ` + generatedToken.Name + `</p><p class="text-sm text-gray-300 mt-1">Type: ` + string(generatedToken.Type) + `</p><pre class="mt-2 text-xs text-kavach-accent bg-kavach-dark border border-kavach-border rounded-lg p-3 overflow-x-auto">` + generatedToken.Payload + `</pre></div>`)
		}

		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Status(201).Send(buf.Bytes())
	}

	// Default: return JSON for API consumers
	return c.Status(201).JSON(fiber.Map{
		"message": "token created",
		"token": fiber.Map{
			"id":          generatedToken.ID,
			"name":        generatedToken.Name,
			"type":        generatedToken.Type,
			"payload":     generatedToken.Payload,
			"trigger_url": generatedToken.TriggerURL,
			"created_at":  generatedToken.CreatedAt,
		},
	})
}
