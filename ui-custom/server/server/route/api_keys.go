// API Keys management for Temporal UI
// Allows users to generate long-lived tokens for SDK authentication

package route

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/temporalio/ui-server/v2/server/config"
)

// APIKey represents a stored API key metadata
type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Scopes      []string  `json:"scopes"`
	CreatedAt   time.Time `json:"createdAt"`
	LastUsedAt  time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	CreatedBy   string    `json:"createdBy"`
}

// APIKeyCreateRequest is the request body for creating a new API key
type APIKeyCreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Scopes      []string `json:"scopes"`
	ExpiresIn   string   `json:"expiresIn,omitempty"` // e.g., "30d", "1y", "never"
}

// APIKeyCreateResponse includes the key metadata and the actual token (shown once)
type APIKeyCreateResponse struct {
	*APIKey
	Token string `json:"token"` // Only shown on creation
}

// In-memory store for API keys (in production, use a database)
var (
	apiKeysStore = make(map[string]*APIKey)
	apiKeysMu    sync.RWMutex
)

// JWTSigningKey is used to sign API key tokens
// In production, this should be loaded from config
var JWTSigningKey []byte

// SetJWTSigningKey sets the key used to sign API key tokens
func SetJWTSigningKey(key string) {
	JWTSigningKey = []byte(key)
}

// APIKeyClaims represents the JWT claims for an API key
type APIKeyClaims struct {
	jwt.RegisteredClaims
	Email  string   `json:"email"`
	Scopes []string `json:"scopes"`
	Type   string   `json:"type"` // "api_key"
	KeyID  string   `json:"keyId"`
}

// SetAPIKeyRoutes registers API key management routes
func SetAPIKeyRoutes(e *echo.Echo, cfgProvider *config.ConfigProviderWithRefresh) {
	api := e.Group("/api/v1/api-keys")

	// Middleware to check authentication
	api.Use(requireAuth(cfgProvider))

	api.GET("", listAPIKeys)
	api.POST("", createAPIKey)
	api.GET("/:id", getAPIKey)
	api.DELETE("/:id", deleteAPIKey)
}

// requireAuth is a middleware that validates the user is authenticated
func requireAuth(cfgProvider *config.ConfigProviderWithRefresh) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check for user cookie (set by OIDC flow)
			userData, err := getUserFromCookies(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			// Store user data in context for handlers
			c.Set("user", userData)
			return next(c)
		}
	}
}

// getUserFromCookies extracts user data from cookies set by OIDC
func getUserFromCookies(c echo.Context) (map[string]interface{}, error) {
	// Read user data from cookies (split across user0, user1, etc.)
	var userDataStr string
	for i := 0; i < 10; i++ {
		cookie, err := c.Cookie("user" + strconv.Itoa(i))
		if err != nil {
			break
		}
		userDataStr += cookie.Value
	}

	if userDataStr == "" {
		return nil, fmt.Errorf("no user data in cookies")
	}

	// Decode base64
	var userData map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(userDataStr))
	if err := decoder.Decode(&userData); err != nil {
		// Try base64 decode first
		decoded, decErr := decodeBase64(userDataStr)
		if decErr != nil {
			return nil, fmt.Errorf("failed to decode user data: %w", err)
		}
		if err := json.Unmarshal(decoded, &userData); err != nil {
			return nil, fmt.Errorf("failed to parse user data: %w", err)
		}
	}

	return userData, nil
}

// decodeBase64 decodes a base64 string
func decodeBase64(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// listAPIKeys returns all API keys for the current user
func listAPIKeys(c echo.Context) error {
	user := c.Get("user").(map[string]interface{})
	email, _ := user["email"].(string)

	apiKeysMu.RLock()
	defer apiKeysMu.RUnlock()

	var userKeys []*APIKey
	for _, key := range apiKeysStore {
		if key.CreatedBy == email {
			userKeys = append(userKeys, key)
		}
	}

	return c.JSON(http.StatusOK, userKeys)
}

// getAPIKey returns a specific API key
func getAPIKey(c echo.Context) error {
	id := c.Param("id")
	user := c.Get("user").(map[string]interface{})
	email, _ := user["email"].(string)

	apiKeysMu.RLock()
	defer apiKeysMu.RUnlock()

	key, exists := apiKeysStore[id]
	if !exists {
		return echo.NewHTTPError(http.StatusNotFound, "API key not found")
	}

	if key.CreatedBy != email {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}

	return c.JSON(http.StatusOK, key)
}

// createAPIKey creates a new API key and returns the token
func createAPIKey(c echo.Context) error {
	var req APIKeyCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	user := c.Get("user").(map[string]interface{})
	email, _ := user["email"].(string)

	// Default scopes if not provided
	if len(req.Scopes) == 0 {
		req.Scopes = []string{"default:read", "default:write"}
	}

	// Parse expiration
	var expiresAt time.Time
	switch req.ExpiresIn {
	case "", "never":
		expiresAt = time.Now().AddDate(10, 0, 0) // 10 years
	case "30d":
		expiresAt = time.Now().AddDate(0, 0, 30)
	case "90d":
		expiresAt = time.Now().AddDate(0, 0, 90)
	case "1y":
		expiresAt = time.Now().AddDate(1, 0, 0)
	default:
		// Try parsing as duration
		if dur, err := time.ParseDuration(req.ExpiresIn); err == nil {
			expiresAt = time.Now().Add(dur)
		} else {
			expiresAt = time.Now().AddDate(1, 0, 0) // Default 1 year
		}
	}

	// Generate key ID
	keyID := uuid.New().String()

	// Create key metadata
	key := &APIKey{
		ID:          keyID,
		Name:        req.Name,
		Description: req.Description,
		Scopes:      req.Scopes,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		CreatedBy:   email,
	}

	// Generate JWT token
	token, err := generateAPIKeyToken(email, keyID, req.Scopes, expiresAt)
	if err != nil {
		log.Printf("[API Keys] Failed to generate token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	// Store key metadata
	apiKeysMu.Lock()
	apiKeysStore[keyID] = key
	apiKeysMu.Unlock()

	log.Printf("[API Keys] Created API key '%s' for user %s (expires: %s)", req.Name, email, expiresAt.Format(time.RFC3339))

	return c.JSON(http.StatusCreated, &APIKeyCreateResponse{
		APIKey: key,
		Token:  token,
	})
}

// deleteAPIKey deletes an API key
func deleteAPIKey(c echo.Context) error {
	id := c.Param("id")
	user := c.Get("user").(map[string]interface{})
	email, _ := user["email"].(string)

	apiKeysMu.Lock()
	defer apiKeysMu.Unlock()

	key, exists := apiKeysStore[id]
	if !exists {
		return echo.NewHTTPError(http.StatusNotFound, "API key not found")
	}

	if key.CreatedBy != email {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}

	delete(apiKeysStore, id)
	log.Printf("[API Keys] Deleted API key '%s' for user %s", key.Name, email)

	return c.NoContent(http.StatusNoContent)
}

// generateAPIKeyToken creates a JWT token for the API key
func generateAPIKeyToken(email, keyID string, scopes []string, expiresAt time.Time) (string, error) {
	if len(JWTSigningKey) == 0 {
		// Generate a random key if not configured
		JWTSigningKey = make([]byte, 32)
		if _, err := rand.Read(JWTSigningKey); err != nil {
			return "", fmt.Errorf("failed to generate signing key: %w", err)
		}
		log.Println("[API Keys] Warning: Using auto-generated JWT signing key. Configure 'apiKeyJwtSecret' for persistence.")
	}

	claims := APIKeyClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "temporal-ui",
			ID:        keyID,
		},
		Email:  email,
		Scopes: scopes,
		Type:   "api_key",
		KeyID:  keyID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSigningKey)
}
