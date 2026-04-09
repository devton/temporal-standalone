// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package route

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/temporalio/ui-server/v2/server/auth"
	"github.com/temporalio/ui-server/v2/server/config"
)

// UserContextKey is the key used to store user info in Echo context
const UserContextKey = "user"

// UserInfo holds extracted user information from OIDC session
type UserInfo struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

// AuthMiddleware creates a middleware that extracts user info from OIDC session
// and populates the Echo context for downstream handlers.
func AuthMiddleware(cfgProvider *config.ConfigProviderWithRefresh) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			log.Printf("[AuthMiddleware] Processing request: %s %s", c.Request().Method, c.Request().URL.Path)

			// Try to get user info from cookies (browser session)
			userInfo, err := getUserFromCookies(c)
			if err == nil && userInfo != nil {
				log.Printf("[AuthMiddleware] Got user from cookies: subject=%s", userInfo.Subject)
				c.Set(UserContextKey, userInfo)
				return next(c)
			} else if err != nil {
				log.Printf("[AuthMiddleware] No user from cookies: %v", err)
			}

			// If auth is enabled, try to get from Authorization header
			cfg, err := cfgProvider.GetConfig()
			if err != nil {
				log.Printf("[AuthMiddleware] Error getting config: %v", err)
				return next(c)
			}

			if !cfg.Auth.Enabled {
				log.Printf("[AuthMiddleware] Auth disabled, proceeding without user")
				return next(c)
			}

			// Try Authorization header (API calls)
			authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
			if authHeader != "" {
				log.Printf("[AuthMiddleware] Found Authorization header")
				userInfo, err := getUserFromAuthToken(c, authHeader, cfg)
				if err == nil && userInfo != nil {
					log.Printf("[AuthMiddleware] Got user from token: subject=%s", userInfo.Subject)
					c.Set(UserContextKey, userInfo)
				} else if err != nil {
					log.Printf("[AuthMiddleware] Error getting user from token: %v", err)
				}
			} else {
				log.Printf("[AuthMiddleware] No Authorization header found")
			}

			return next(c)
		}
	}
}

// getUserFromCookies extracts user info from session cookies
func getUserFromCookies(c echo.Context) (*UserInfo, error) {
	// Read all user cookies (user0, user1, etc.)
	var userData strings.Builder
	for i := 0; i < 10; i++ {
		cookie, err := c.Request().Cookie("user" + strconv.Itoa(i))
		if err != nil {
			break
		}
		userData.WriteString(cookie.Value)
	}

	if userData.Len() == 0 {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "no user session")
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(userData.String())
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid session data")
	}

	// Parse UserResponse
	var userResp auth.UserResponse
	if err := json.Unmarshal(decoded, &userResp); err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid user data")
	}

	// If we have an ID token, extract claims from it
	if userResp.IDToken != "" {
		userInfo, err := parseIDTokenClaims(userResp.IDToken)
		if err != nil {
			log.Printf("[AuthMiddleware] Error parsing ID token: %v", err)
		}
		if userInfo != nil {
			return userInfo, nil
		}
	}

	// Fallback: create UserInfo from UserResponse
	// Use email as subject if available
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

// getUserFromAuthToken extracts user info from Authorization header
func getUserFromAuthToken(c echo.Context, authHeader string, cfg *config.Config) (*UserInfo, error) {
	// Strip "Bearer " prefix
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "empty token")
	}

	// If using OIDC, verify token and extract claims
	verifier := auth.GetVerifier()
	if verifier != nil {
		ctx := c.Request().Context()
		idToken, err := verifier.Verify(ctx, tokenString)
		if err != nil {
			log.Printf("[AuthMiddleware] Token verification failed: %v", err)
			return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		// Extract claims
		var claims struct {
			Subject           string `json:"sub"`
			Email             string `json:"email"`
			Name              string `json:"name"`
			PreferredUsername string `json:"preferred_username"`
		}
		if err := idToken.Claims(&claims); err != nil {
			log.Printf("[AuthMiddleware] Error extracting claims: %v", err)
			return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
		}

		subject := claims.Subject
		if subject == "" {
			subject = claims.Email
		}
		if subject == "" {
			subject = claims.PreferredUsername
		}

		return &UserInfo{
			Subject: subject,
			Email:   claims.Email,
			Name:    claims.Name,
		}, nil
	}

	// If no OIDC verifier, try to parse as unverified JWT (for development)
	// This should not be used in production without proper verification
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid token format")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid claims")
	}

	// Extract subject
	var subject string
	if sub, ok := claims["sub"].(string); ok {
		subject = sub
	} else if email, ok := claims["email"].(string); ok {
		subject = email
	} else if preferredUsername, ok := claims["preferred_username"].(string); ok {
		subject = preferredUsername
	}

	if subject == "" {
		subject = "default-user"
	}

	// Extract other claims
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)

	return &UserInfo{
		Subject: subject,
		Email:   email,
		Name:    name,
	}, nil
}

// parseIDTokenClaims parses an ID token and extracts claims without verification
// (verification was already done during login)
func parseIDTokenClaims(idToken string) (*UserInfo, error) {
	// Parse without verification (already verified during OIDC flow)
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid claims")
	}

	// Extract subject
	var subject string
	if sub, ok := claims["sub"].(string); ok {
		subject = sub
	} else if email, ok := claims["email"].(string); ok {
		subject = email
	} else if preferredUsername, ok := claims["preferred_username"].(string); ok {
		subject = preferredUsername
	}

	if subject == "" {
		subject = "unknown"
	}

	// Extract other claims
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)

	return &UserInfo{
		Subject: subject,
		Email:   email,
		Name:    name,
	}, nil
}