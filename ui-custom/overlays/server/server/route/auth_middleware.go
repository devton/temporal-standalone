package route

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/temporalio/ui-server/v2/server/config"
)

const UserContextKey = "user"

type UserInfo struct {
	Subject  string `json:"sub"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	IsAPIKey bool   `json:"-"`
	KeyName  string `json:"keyName,omitempty"`
	KeyID    string `json:"keyId,omitempty"`
}

func AuthMiddleware(cfgProvider *config.ConfigProviderWithRefresh) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Try to get user from OIDC session cookies (server-set, safe to parse without verify)
			userInfo, err := getUserFromCookies(c)
			if err == nil && userInfo != nil {
				c.Set(UserContextKey, userInfo)
				return next(c)
			}

			// 2. Try Authorization-Extras header (OIDC ID token from browser)
			idToken := c.Request().Header.Get("Authorization-Extras")
			if idToken != "" {
				ui, err := parseUnverifiedJWT(idToken)
				if err == nil && ui != nil {
					c.Set(UserContextKey, ui)
					return next(c)
				}
				log.Printf("[AuthMiddleware] ID token parse failed: %v", err)
			}

			// 3. Try Authorization header (Bearer token)
			authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
			if authHeader != "" {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				if tokenString != "" {
					// 3a. Try API key JWT (HS256, issuer == "temporal-standalone")
					if ui, err := verifyAPIKeyToken(tokenString); err == nil && ui != nil {
						c.Set(UserContextKey, ui)
						return next(c)
					}

					// 3b. Fallback: unverified parse for OIDC Bearer tokens
					ui, err := parseUnverifiedJWT(tokenString)
					if err == nil && ui != nil {
						c.Set(UserContextKey, ui)
						return next(c)
					}
					log.Printf("[AuthMiddleware] Bearer token parse failed: %v", err)
				}
			}

			return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
		}
	}
}

func verifyAPIKeyToken(tokenString string) (*UserInfo, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "temporal-api-key-secret-change-in-production"
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, echo.NewHTTPError(http.StatusUnauthorized, "unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid API key token")
	}

	issuer, _ := claims["iss"].(string)
	if issuer != "temporal-standalone" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "wrong issuer")
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "api_key" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "not an API key token")
	}

	subject, _ := claims["sub"].(string)
	if subject == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "no subject in token")
	}

	return &UserInfo{
		Subject:  subject,
		IsAPIKey: true,
		KeyName:  getStringClaim(claims, "key_name"),
		KeyID:    getStringClaim(claims, "key_id"),
	}, nil
}

func getStringClaim(claims jwt.MapClaims, key string) string {
	v, _ := claims[key].(string)
	return v
}

func getUserFromCookies(c echo.Context) (*UserInfo, error) {
	var userData strings.Builder
	for i := 0; i < 10; i++ {
		cookie, err := c.Request().Cookie("user" + strconv.Itoa(i))
		if err != nil {
			break
		}
		userData.WriteString(cookie.Value)
	}

	if userData.Len() == 0 {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "no session")
	}

	decoded, err := base64.StdEncoding.DecodeString(userData.String())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid session")
	}

	var userResp struct {
		AccessToken string `json:"AccessToken"`
		IDToken     string `json:"IDToken"`
		Name        string `json:"Name"`
		Email       string `json:"Email"`
		Picture     string `json:"Picture"`
	}
	if err := json.Unmarshal(decoded, &userResp); err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid session data")
	}

	// Extract from ID token if present
	if userResp.IDToken != "" {
		ui, _ := parseUnverifiedJWT(userResp.IDToken)
		if ui != nil {
			return ui, nil
		}
	}

	// Fallback to user response fields
	subject := userResp.Email
	if subject == "" {
		subject = "unknown"
	}
	return &UserInfo{
		Subject: subject,
		Email:   userResp.Email,
		Name:    userResp.Name,
	}, nil
}

func parseUnverifiedJWT(tokenString string) (*UserInfo, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) < 2 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "not a JWT")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Try standard encoding
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, err
		}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	var subject string
	if email, ok := claims["email"].(string); ok && email != "" {
		subject = email
	} else if preferredUsername, ok := claims["preferred_username"].(string); ok && preferredUsername != "" {
		subject = preferredUsername
	} else if sub, ok := claims["sub"].(string); ok && sub != "" {
		subject = sub
	} else if name, ok := claims["name"].(string); ok && name != "" {
		subject = name
	}

	if subject == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "no subject in token")
	}

	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)

	return &UserInfo{
		Subject: subject,
		Email:   email,
		Name:    name,
	}, nil
}
