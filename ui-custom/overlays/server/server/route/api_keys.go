package route

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

// APIKey represents an API key for SDK access
type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	KeyID       string    `json:"keyId"`
	KeySecret   string    `json:"keySecret,omitempty"` // Only returned on creation
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	OwnerID     string    `json:"ownerId"`
	LastUsedAt  time.Time `json:"lastUsedAt,omitempty"`
}

// APIKeyCreateRequest is the request body for creating an API key
type APIKeyCreateRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
}

// JWT claims for API keys
type APIKeyClaims struct {
	jwt.RegisteredClaims
	KeyID   string `json:"key_id"`
	KeyName string `json:"key_name"`
	OwnerID string `json:"owner_id"`
	Type    string `json:"type"`
}

// In-memory store for API keys (should be replaced with database in production)
var apiKeysStore = make(map[string]*APIKey)

// JWT secret for signing API key tokens (from env JWT_SECRET)
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "temporal-api-key-secret-change-in-production"
	}
	return []byte(secret)
}

// generateKeyID generates a random key ID
func generateKeyID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "key_" + hex.EncodeToString(bytes), nil
}

// generateKeySecret generates a random key secret
func generateKeySecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateJWTToken generates a JWT token for the API key
func generateJWTToken(apiKey *APIKey) (string, error) {
	claims := APIKeyClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   apiKey.KeyID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(apiKey.ExpiresAt),
			Issuer:    "temporal-standalone",
		},
		KeyID:   apiKey.KeyID,
		KeyName: apiKey.Name,
		OwnerID: apiKey.OwnerID,
		Type:    "api_key",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// getOwnerIDFromContext extracts the owner ID from the auth context
func getOwnerIDFromContext(c echo.Context) string {
	// Try to get from JWT claims in context
	if user := c.Get("user"); user != nil {
		if claims, ok := user.(*jwt.Token); ok {
			if mapClaims, ok := claims.Claims.(jwt.MapClaims); ok {
				if sub, ok := mapClaims["sub"].(string); ok {
					return sub
				}
			}
		}
	}
	// Fallback to a default owner
	return "default-user"
}

// ListAPIKeys returns all API keys for the current user
func ListAPIKeys(c echo.Context) error {
	ownerID := getOwnerIDFromContext(c)

	keys := make([]*APIKey, 0)
	for _, key := range apiKeysStore {
		if key.OwnerID == ownerID {
			// Create a copy without the secret
			keyCopy := *key
			keyCopy.KeySecret = ""
			keys = append(keys, &keyCopy)
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"keys": keys,
	})
}

// CreateAPIKey creates a new API key
func CreateAPIKey(c echo.Context) error {
	var req APIKeyCreateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Name is required")
	}

	keyID, err := generateKeyID()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate key ID")
	}

	keySecret, err := generateKeySecret()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate key secret")
	}

	ownerID := getOwnerIDFromContext(c)

	apiKey := &APIKey{
		ID:          keyID,
		Name:        req.Name,
		Description: req.Description,
		KeyID:       keyID,
		KeySecret:   keySecret,
		CreatedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
		OwnerID:     ownerID,
	}

	// Generate JWT token
	token, err := generateJWTToken(apiKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate token")
	}

	// Store the token as the key secret
	apiKey.KeySecret = token
	apiKeysStore[keyID] = apiKey

	return c.JSON(http.StatusCreated, apiKey)
}

// DeleteAPIKey deletes an API key
func DeleteAPIKey(c echo.Context) error {
	keyID := c.Param("id")
	if keyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Key ID is required")
	}

	ownerID := getOwnerIDFromContext(c)

	key, exists := apiKeysStore[keyID]
	if !exists {
		return echo.NewHTTPError(http.StatusNotFound, "API key not found")
	}

	if key.OwnerID != ownerID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	delete(apiKeysStore, keyID)

	return c.NoContent(http.StatusNoContent)
}

// RegisterAPIKeyRoutes registers the API key routes
func RegisterAPIKeyRoutes(e *echo.Group) {
	api := e.Group("/api-keys")

	api.GET("", ListAPIKeys)
	api.POST("", CreateAPIKey)
	api.DELETE("/:id", DeleteAPIKey)
}
