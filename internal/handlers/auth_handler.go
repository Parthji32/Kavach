package handlers

import (
	"database/sql"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/parthjindal/kavach/internal/middleware"
	"github.com/parthjindal/kavach/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication routes
type AuthHandler struct {
	db *sql.DB
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

func isEmpty(s string) bool {
	return len(s) == 0
}

// isProduction checks if the ENV variable is set to "production"
func isProduction() bool {
	return os.Getenv("ENV") == "production"
}

// maxCredentialLen caps credential length to prevent DoS via bcrypt on huge inputs.
// bcrypt itself truncates at 72 bytes, but we cap at 128 to be explicit.
const maxCredentialLen = 128

// Signup handles user registration
// POST /api/v1/auth/signup
func (h *AuthHandler) Signup(c *fiber.Ctx) error {
	var req models.SignupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if isEmpty(req.Email) || isEmpty(req.Credential) || isEmpty(req.Name) {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Email, credential, and name are required",
		})
	}

	// Validate email format (fix S09)
	if _, err := mail.ParseAddress(req.Email); err != nil || !strings.Contains(req.Email, ".") {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Invalid email address format",
		})
	}

	// Validate email length to prevent abuse
	if len(req.Email) > 254 {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Email address is too long",
		})
	}

	if len(req.Credential) < 8 {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Credential must be at least 8 characters",
		})
	}

	// Cap credential length to prevent DoS (bcrypt is intentionally slow on long inputs)
	if len(req.Credential) > maxCredentialLen {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Credential must be 128 characters or fewer",
		})
	}

	// Cap name length
	if len(req.Name) > 128 {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Name must be 128 characters or fewer",
		})
	}

	// Check if email already exists
	var existingID string
	err := h.db.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&existingID)
	if err == nil {
		return c.Status(409).JSON(fiber.Map{
			"error":   "email_exists",
			"message": "An account with this email already exists",
		})
	}

	// Hash the credential
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Credential), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash credential: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to create account",
		})
	}

	// Create user
	user := models.User{
		ID:        uuid.New(),
		Email:     req.Email,
		PassHash:  string(hashed),
		Name:      req.Name,
		Company:   req.Company,
		Plan:      models.PlanFree,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = h.db.Exec(
		`INSERT INTO users (id, email, pass_hash, name, company, plan, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (email) DO NOTHING`,
		user.ID, user.Email, user.PassHash, user.Name, user.Company,
		user.Plan, user.IsActive, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to create account",
		})
	}

	// Check if we actually inserted (ON CONFLICT = email already taken)
	var checkID uuid.UUID
	_ = h.db.QueryRow("SELECT id FROM users WHERE email = $1", user.Email).Scan(&checkID)
	if checkID != user.ID {
		return c.Status(409).JSON(fiber.Map{
			"error":   "conflict",
			"message": "An account with this email already exists",
		})
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Email, string(user.Plan))
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Account created but failed to generate token",
		})
	}

	// Set cookie for browser-based auth
	c.Cookie(&fiber.Cookie{
		Name:     "kavach_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   isProduction(),
		SameSite: "Lax",
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
	})

	log.Printf("New user registered: %s (%s)", user.Email, user.ID)

	return c.Status(201).JSON(models.AuthResponse{
		Token: token,
		User:  user,
	})
}

// Login handles user authentication
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if isEmpty(req.Email) || isEmpty(req.Credential) {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Email and credential are required",
		})
	}

	// Cap credential length on login too — prevents bcrypt DoS
	if len(req.Credential) > maxCredentialLen {
		return c.Status(401).JSON(fiber.Map{
			"error":   "invalid_credentials",
			"message": "Invalid email or credential",
		})
	}

	// Find user by email
	var user models.User
	err := h.db.QueryRow(
		`SELECT id, email, pass_hash, name, company, plan, is_active, created_at, updated_at
		 FROM users WHERE email = $1`,
		req.Email,
	).Scan(
		&user.ID, &user.Email, &user.PassHash, &user.Name,
		&user.Company, &user.Plan, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return c.Status(401).JSON(fiber.Map{
			"error":   "invalid_credentials",
			"message": "Invalid email or credential",
		})
	}
	if err != nil {
		log.Printf("Database error during login: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Login failed",
		})
	}

	if !user.IsActive {
		return c.Status(403).JSON(fiber.Map{
			"error":   "account_disabled",
			"message": "This account has been disabled",
		})
	}

	// Verify credential
	if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(req.Credential)); err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error":   "invalid_credentials",
			"message": "Invalid email or credential",
		})
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Email, string(user.Plan))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Login succeeded but failed to generate token",
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "kavach_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   isProduction(),
		SameSite: "Lax",
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
	})

	log.Printf("User logged in: %s", user.Email)

	return c.JSON(models.AuthResponse{
		Token: token,
		User:  user,
	})
}

// Logout clears the auth cookie
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "kavach_token",
		Value:    "",
		HTTPOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
	})

	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

// Me returns the current user info
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var user models.User
	err := h.db.QueryRow(
		`SELECT id, email, name, company, plan, is_active, created_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Company, &user.Plan, &user.IsActive, &user.CreatedAt)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user_not_found"})
	}

	return c.JSON(user)
}
