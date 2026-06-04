package route

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/temporalio/ui-server/v2/server/config"
)

type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	KeyID       string    `json:"keyId"`
	KeySecret   string    `json:"keySecret,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	OwnerID     string    `json:"ownerId"`
	LastUsedAt  time.Time `json:"lastUsedAt,omitempty"`
}

type APIKeyCreateRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
}

type APIKeyClaims struct {
	jwt.RegisteredClaims
	KeyID   string `json:"key_id"`
	KeyName string `json:"key_name"`
	OwnerID string `json:"owner_id"`
	Type    string `json:"type"`
}

var (
	cacheMu        sync.RWMutex
	apiKeysCache   = make(map[string]*APIKey)
	cachePopulated bool
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "temporal-api-key-secret-change-in-production"
	}
	return []byte(secret)
}

func generateKeyID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "key_" + hex.EncodeToString(bytes), nil
}

func generateKeySecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

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

func getOwnerIDFromContext(c echo.Context) (string, error) {
	if user := c.Get(UserContextKey); user != nil {
		if userInfo, ok := user.(*UserInfo); ok {
			if userInfo.Subject != "" {
				return userInfo.Subject, nil
			}
		}
	}

	if user := c.Get("user"); user != nil {
		if claims, ok := user.(*jwt.Token); ok {
			if mapClaims, ok := claims.Claims.(jwt.MapClaims); ok {
				if sub, ok := mapClaims["sub"].(string); ok {
					return sub, nil
				}
				if email, ok := mapClaims["email"].(string); ok {
					return email, nil
				}
			}
		}
	}

	return "", echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
}

func ensureCache(dbConn *sql.DB) {
	if cachePopulated {
		return
	}
	if dbConn == nil {
		return
	}

	rows, err := dbConn.Query(
		"SELECT key_id, name, description, key_secret, created_at, expires_at, owner_id FROM api_keys",
	)
	if err != nil {
		log.Printf("[APIKeys] Failed to load keys from DB: %v", err)
		return
	}
	defer rows.Close()

	loaded := make(map[string]*APIKey)
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.KeyID, &k.Name, &k.Description, &k.KeySecret, &k.CreatedAt, &k.ExpiresAt, &k.OwnerID); err != nil {
			log.Printf("[APIKeys] Failed to scan key row: %v", err)
			continue
		}
		k.ID = k.KeyID
		loaded[k.KeyID] = &k
	}

	cacheMu.Lock()
	for key, val := range loaded {
		apiKeysCache[key] = val
	}
	cachePopulated = true
	cacheMu.Unlock()
}

func cacheGet(keyID string) (*APIKey, bool) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	k, ok := apiKeysCache[keyID]
	return k, ok
}

func cacheSet(keyID string, k *APIKey) {
	cacheMu.Lock()
	apiKeysCache[keyID] = k
	cacheMu.Unlock()
}

func cacheDelete(keyID string) {
	cacheMu.Lock()
	delete(apiKeysCache, keyID)
	cacheMu.Unlock()
}

func ListAPIKeys(c echo.Context) error {
	ownerID, err := getOwnerIDFromContext(c)
	if err != nil {
		return err
	}

	dbConn := getDB()
	if dbConn != nil {
		ensureCache(dbConn)

		rows, err := dbConn.Query(
			"SELECT key_id, name, description, key_secret, created_at, expires_at, owner_id FROM api_keys WHERE owner_id = $1 ORDER BY created_at DESC",
			ownerID,
		)
		if err != nil {
			log.Printf("[APIKeys] DB query failed, falling back to cache: %v", err)
			return listFromCache(ownerID, c)
		}
		defer rows.Close()

		keys := make([]*APIKey, 0)
		for rows.Next() {
			var k APIKey
			if err := rows.Scan(&k.KeyID, &k.Name, &k.Description, &k.KeySecret, &k.CreatedAt, &k.ExpiresAt, &k.OwnerID); err != nil {
				continue
			}
			k.ID = k.KeyID
			k.KeySecret = ""
			keys = append(keys, &k)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{"keys": keys})
	}

	return listFromCache(ownerID, c)
}

func listFromCache(ownerID string, c echo.Context) error {
	keys := make([]*APIKey, 0)
	cacheMu.RLock()
	for _, key := range apiKeysCache {
		if key.OwnerID == ownerID {
			keyCopy := *key
			keyCopy.KeySecret = ""
			keys = append(keys, &keyCopy)
		}
	}
	cacheMu.RUnlock()

	return c.JSON(http.StatusOK, map[string]interface{}{"keys": keys})
}

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

	ownerID, err := getOwnerIDFromContext(c)
	if err != nil {
		return err
	}

	var expiresAt sql.NullTime
	if !req.ExpiresAt.IsZero() {
		expiresAt = sql.NullTime{Time: req.ExpiresAt, Valid: true}
	}

	apiKey := &APIKey{
		ID:          keyID,
		Name:        req.Name,
		Description: req.Description,
		KeyID:       keyID,
		CreatedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
		OwnerID:     ownerID,
	}

	token, err := generateJWTToken(apiKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate token")
	}
	apiKey.KeySecret = token

	dbConn := getDB()
	if dbConn != nil {
		var desc sql.NullString
		if req.Description != "" {
			desc = sql.NullString{String: req.Description, Valid: true}
		}

		_, err = dbConn.Exec(
			"INSERT INTO api_keys (key_id, name, description, key_secret, expires_at, owner_id) VALUES ($1, $2, $3, $4, $5, $6)",
			keyID, req.Name, desc, token, expiresAt, ownerID,
		)
		if err != nil {
			log.Printf("[APIKeys] DB insert failed: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to store API key")
		}
	}

	cacheSet(keyID, apiKey)

	return c.JSON(http.StatusCreated, apiKey)
}

func DeleteAPIKey(c echo.Context) error {
	keyID := c.Param("id")
	if keyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Key ID is required")
	}

	ownerID, err := getOwnerIDFromContext(c)
	if err != nil {
		return err
	}

	dbConn := getDB()
	if dbConn != nil {
		var dbOwner string
		err := dbConn.QueryRow("SELECT owner_id FROM api_keys WHERE key_id = $1", keyID).Scan(&dbOwner)
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "API key not found")
		}
		if err != nil {
			log.Printf("[APIKeys] DB query failed: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to lookup API key")
		}
		if dbOwner != ownerID {
			return echo.NewHTTPError(http.StatusForbidden, "Access denied")
		}

		_, err = dbConn.Exec("DELETE FROM api_keys WHERE key_id = $1", keyID)
		if err != nil {
			log.Printf("[APIKeys] DB delete failed: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete API key")
		}
	} else {
		key, exists := cacheGet(keyID)
		if !exists {
			return echo.NewHTTPError(http.StatusNotFound, "API key not found")
		}
		if key.OwnerID != ownerID {
			return echo.NewHTTPError(http.StatusForbidden, "Access denied")
		}
	}

	cacheDelete(keyID)

	return c.NoContent(http.StatusNoContent)
}

func RegisterAPIKeyRoutes(e *echo.Group, cfgProvider *config.ConfigProviderWithRefresh) {
	initDB()

	authMW := AuthMiddleware(cfgProvider)

	api := e.Group("/api-keys", authMW)

	api.GET("", ListAPIKeys)
	api.POST("", CreateAPIKey)
	api.DELETE("/:"+"id", DeleteAPIKey)
}
