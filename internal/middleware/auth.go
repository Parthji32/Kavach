package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims holds the JWT token claims
type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Plan   string    `json:"plan"`
	jwt.RegisteredClaims
}

// jwtSecret is initialized once at package load time
var jwtSecret string

func init() {
	jwtSecret = os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Generate a random secret at startup
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			log.Fatal("CRITICAL: Failed to generate random JWT secret")
		}
		jwtSecret = hex.EncodeToString(bytes)
		log.Println("⚠️  WARNING: JWT_SECRET not set. Generated a random secret for this session.")
		log.Println("   JWTs will be invalid after restart. Set JWT_SECRET in .env for persistence.")
	}
}

func getJWTSecret() string {
	return jwtSecret
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(userID uuid.UUID, email, plan string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Plan:   plan,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "kavach",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

// AuthRequired is middleware that validates JWT tokens on protected routes
func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			authHeader = c.Cookies("kavach_token")
			if authHeader == "" {
				return c.Status(401).JSON(fiber.Map{
					"error":   "unauthorized",
					"message": "Missing authentication token",
				})
			}
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(getJWTSecret()), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
			})
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Invalid token claims",
			})
		}

		c.Locals("userID", claims.UserID)
		c.Locals("email", claims.Email)
		c.Locals("plan", claims.Plan)

		return c.Next()
	}
}

// GetUserID extracts the user ID from the Fiber context
func GetUserID(c *fiber.Ctx) uuid.UUID {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}

// GetUserEmail extracts the email from the Fiber context
func GetUserEmail(c *fiber.Ctx) string {
	email, ok := c.Locals("email").(string)
	if !ok {
		return ""
	}
	return email
}
