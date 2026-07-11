package route

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
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
	Namespace   string    `json:"namespace"`
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	OwnerID     string    `json:"ownerId"`
	LastUsedAt  time.Time `json:"lastUsedAt,omitempty"`
}

type APIKeyCreateRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Namespace   string    `json:"namespace"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
}

type APIKeyClaims struct {
	jwt.RegisteredClaims
	KeyID       string   `json:"key_id"`
	KeyName     string   `json:"key_name"`
	OwnerID     string   `json:"owner_id"`
	Namespace   string   `json:"namespace"`
	Permissions []string `json:"permissions"`
	Type        string   `json:"type"`
}

var (
	cacheMu        sync.RWMutex
	apiKeysCache   = make(map[string]*APIKey)
	cachePopulated bool
)

// HandleJWKS serves the public keys for all active API keys.
// When a key is deleted, it disappears from this endpoint,
// causing the Temporal Server to reject its JWTs within the next
// JWKS refresh cycle (default 1 minute).
func HandleJWKS(c echo.Context) error {
	dbConn := getDB()

	var jwkKeys []map[string]interface{}

	if dbConn != nil {
		rows, err := dbConn.Query("SELECT jwks_kid, public_key FROM api_keys WHERE jwks_kid IS NOT NULL AND public_key IS NOT NULL AND public_key != ''")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var kid, pubPEM string
				if err := rows.Scan(&kid, &pubPEM); err != nil {
					continue
				}
				block, _ := pem.Decode([]byte(pubPEM))
				if block == nil {
					continue
				}
				pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
				if err != nil {
					continue
				}
				rsaPub, ok := pubKey.(*rsa.PublicKey)
				if !ok {
					continue
				}
				jwkKeys = append(jwkKeys, map[string]interface{}{
					"kty": "RSA",
					"use": "sig",
					"alg": "RS256",
					"kid": kid,
					"n":   base64.RawURLEncoding.EncodeToString(rsaPub.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(rsaPub.E)).Bytes()),
				})
			}
		}
	}

	if jwkKeys == nil {
		jwkKeys = []map[string]interface{}{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"keys": jwkKeys})
}

func generateKeyID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "key_" + hex.EncodeToString(bytes), nil
}

func generateUniqueKID() string {
	bytes := make([]byte, 10)
	rand.Read(bytes)
	return "apk_" + hex.EncodeToString(bytes)
}

// generateAPIKeyJWT creates a unique RSA key pair for this API key,
// signs the JWT with the private key, and stores the public key in
// the database so it appears in the JWKS endpoint. Deleting the API
// key removes the public key from JWKS, revoking the token.
func generateAPIKeyJWT(apiKey *APIKey) (string, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	kid := generateUniqueKID()

	exp := apiKey.ExpiresAt
	if exp.IsZero() {
		exp = time.Now().Add(365 * 24 * time.Hour)
	}

	permissions := []string{apiKey.Namespace + ":admin"}

	claims := APIKeyClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   kid,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    "temporal-ui",
			Audience:  []string{"temporal"},
		},
		KeyID:       kid,
		KeyName:     apiKey.Name,
		OwnerID:     apiKey.OwnerID,
		Namespace:   apiKey.Namespace,
		Permissions: permissions,
		Type:        "api_key",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes})

	dbConn := getDB()
	if dbConn != nil {
		_, err = dbConn.Exec(
			"UPDATE api_keys SET jwks_kid = $1, public_key = $2 WHERE key_id = $3",
			kid, string(pubPEM), apiKey.ID,
		)
		if err != nil {
			log.Printf("[APIKeys] Failed to store JWKS key: %v", err)
		}
	}

	return token.SignedString(privKey)
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
		"SELECT key_id, name, description, key_hash, namespace, created_at, expires_at, owner_id FROM api_keys",
	)
	if err != nil {
		log.Printf("[APIKeys] Failed to load keys from DB: %v", err)
		return
	}
	defer rows.Close()

	loaded := make(map[string]*APIKey)
	for rows.Next() {
		var k APIKey
		var desc, keyHash, ns sql.NullString
		var exp sql.NullTime
		if err := rows.Scan(&k.KeyID, &k.Name, &desc, &keyHash, &ns, &k.CreatedAt, &exp, &k.OwnerID); err != nil {
			log.Printf("[APIKeys] Failed to scan key row: %v", err)
			continue
		}
		k.Description = desc.String
		k.Namespace = ns.String
		if exp.Valid {
			k.ExpiresAt = exp.Time
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
			"SELECT key_id, name, description, key_hash, namespace, created_at, expires_at, owner_id FROM api_keys WHERE owner_id = $1 ORDER BY created_at DESC",
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
			var description, keyHash, namespace, expiresAt sql.NullString
			if err := rows.Scan(&k.KeyID, &k.Name, &description, &keyHash, &namespace, &k.CreatedAt, &expiresAt, &k.OwnerID); err != nil {
				log.Printf("[APIKeys] Failed to scan key row: %v", err)
				continue
			}
			k.Description = description.String
			k.Namespace = namespace.String
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

	if req.Namespace == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Namespace is required")
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

	dbConn := getDB()
	if dbConn != nil {
		var desc sql.NullString
		if req.Description != "" {
			desc = sql.NullString{String: req.Description, Valid: true}
		}

		_, err = dbConn.Exec(
			"INSERT INTO api_keys (key_id, name, description, key_hash, namespace, expires_at, owner_id) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			keyID, req.Name, desc, "", req.Namespace, expiresAt, ownerID,
		)
		if err != nil {
			log.Printf("[APIKeys] DB insert failed: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to store API key")
		}
	}

	apiKey := &APIKey{
		ID:          keyID,
		Name:        req.Name,
		Description: req.Description,
		KeyID:       keyID,
		Namespace:   req.Namespace,
		CreatedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
		OwnerID:     ownerID,
	}

	token, err := generateAPIKeyJWT(apiKey)
	if err != nil {
		log.Printf("[APIKeys] Failed to generate token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate token")
	}
	apiKey.KeySecret = token

	if dbConn != nil {
		_, _ = dbConn.Exec("UPDATE api_keys SET key_hash = $1 WHERE key_id = $2", token, keyID)
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

		log.Printf("[APIKeys] Key %s deleted — JWKS will no longer contain its kid", keyID)
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
	ensureAPIMigrations()

	authMW := AuthMiddleware(cfgProvider)

	api := e.Group("/api-keys", authMW)

	api.GET("", ListAPIKeys)
	api.POST("", CreateAPIKey)
	api.DELETE("/:id", DeleteAPIKey)
}

// Ensure migrations for per-key JWKS columns
func ensureAPIMigrations() {
	dbConn := getDB()
	if dbConn == nil {
		return
	}
	// Add columns for per-key JWKS
	_, err := dbConn.Exec("ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS jwks_kid TEXT")
	if err != nil {
		log.Printf("[APIKeys] ALTER jwks_kid: %v", err)
	}
	_, err = dbConn.Exec("ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS public_key TEXT")
	if err != nil {
		log.Printf("[APIKeys] ALTER public_key: %v", err)
	}
	_, err = dbConn.Exec("ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_hash TEXT NOT NULL DEFAULT ''")
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Printf("[APIKeys] ALTER key_hash: %v", err)
		}
	}
	_, err = dbConn.Exec("ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS namespace TEXT NOT NULL DEFAULT ''")
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Printf("[APIKeys] ALTER namespace: %v", err)
		}
	}
	log.Println("[APIKeys] Migrations complete — per-key JWKS ready")
}
