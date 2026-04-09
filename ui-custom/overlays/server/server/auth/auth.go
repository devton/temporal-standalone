// Overlay for server/auth/auth.go - adds GetVerifier function
// This file is copied over the upstream version during build

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/temporalio/ui-server/v2/server/config"
)

const (
	AuthorizationExtrasHeader = "authorization-extras"
	cookieLen                 = 4000
	sessionStartCookie        = "session_start"
)

var tokenVerifier *oidc.IDTokenVerifier

func SetVerifier(v *oidc.IDTokenVerifier) {
	tokenVerifier = v
}

// GetVerifier returns the OIDC token verifier
func GetVerifier() *oidc.IDTokenVerifier {
	return tokenVerifier
}

func stripBearerPrefix(token string) string {
	return strings.TrimPrefix(token, "Bearer ")
}

func SetUser(c echo.Context, user *User) error {
	if user.OAuth2Token == nil {
		return errors.New("no OAuth2Token")
	}

	userR := UserResponse{
		AccessToken: user.OAuth2Token.AccessToken,
	}

	if user.IDToken != nil {
		userR.IDToken = user.IDToken.RawToken
		if user.IDToken.Claims != nil {
			userR.Name = user.IDToken.Claims.Name
			userR.Email = user.IDToken.Claims.Email
			userR.Picture = user.IDToken.Claims.Picture
		}
	}

	b, err := json.Marshal(userR)
	if err != nil {
		return errors.New("unable to serialize user data")
	}

	s := base64.StdEncoding.EncodeToString(b)
	parts := splitCookie(s)

	for i, p := range parts {
		cookie := &http.Cookie{
			Name:     "user" + strconv.Itoa(i),
			Value:    p,
			MaxAge:   int(time.Minute.Seconds()),
			Secure:   c.Request().TLS != nil,
			HttpOnly: false,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		}
		c.SetCookie(cookie)
	}

	if rt := user.OAuth2Token.RefreshToken; rt != "" {
		log.Println("[Auth] Setting refresh token cookie")

		var refreshMaxAge int
		if user.OAuth2Token.Expiry.IsZero() {
			refreshMaxAge = int((7 * 24 * time.Hour).Seconds())
			log.Printf("[Auth] Warning: No refresh token expiry from IdP, using 7-day default")
		} else {
			maxAge := time.Until(user.OAuth2Token.Expiry)
			if maxAge > 30*24*time.Hour {
				maxAge = 30 * 24 * time.Hour
				log.Printf("[Auth] Warning: IdP refresh token expiry > 30 days, capping at 30 days")
			}
			refreshMaxAge = int(maxAge.Seconds())
			log.Printf("[Auth] Setting refresh cookie MaxAge to %d seconds (%.1f days) from IdP",
				refreshMaxAge, maxAge.Hours()/24)
		}

		refreshCookie := &http.Cookie{
			Name:     "refresh",
			Value:    rt,
			MaxAge:   refreshMaxAge,
			Secure:   c.Request().TLS != nil,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		}
		c.SetCookie(refreshCookie)
	} else {
		log.Println("[Auth] No refresh token received from OAuth provider")
	}

	return nil
}

// SetSessionStart sets a cookie with the current timestamp to track when the session began.
func SetSessionStart(c echo.Context, maxSessionDuration time.Duration) {
	if maxSessionDuration <= 0 {
		return
	}

	cookie := &http.Cookie{
		Name:     sessionStartCookie,
		Value:    strconv.FormatInt(time.Now().Unix(), 10),
		MaxAge:   int(maxSessionDuration.Seconds()),
		Secure:   c.Request().TLS != nil,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(cookie)
}

// ValidateSessionDuration checks if the session has exceeded the configured max duration.
func ValidateSessionDuration(c echo.Context, maxSessionDuration time.Duration) error {
	if maxSessionDuration <= 0 {
		return nil
	}

	cookie, err := c.Request().Cookie(sessionStartCookie)
	if err != nil {
		return errors.New("session expired: missing session start")
	}

	startTime, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		return errors.New("session expired: invalid session start")
	}

	sessionAge := time.Since(time.Unix(startTime, 0))
	if sessionAge > maxSessionDuration {
		return fmt.Errorf("session expired: exceeded max duration of %v", maxSessionDuration)
	}

	return nil
}

func validateJWT(ctx context.Context, tokenString string) error {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	if tokenString == "" {
		log.Println("[JWT Validation] Token is empty after stripping Bearer prefix")
		return errors.New("token is empty")
	}

	if tokenVerifier == nil {
		log.Println("[JWT Validation] CRITICAL: No verifier configured but validation was requested")
		return errors.New("authentication verifier not initialized")
	}

	_, err := tokenVerifier.Verify(ctx, tokenString)
	if err != nil {
		log.Printf("[JWT Validation] Token verification failed: %v", err)
		return errors.New("token invalid or expired")
	}

	log.Println("[JWT Validation] Token verified successfully")
	return nil
}

// ValidateAuthHeaderExists validates that the autorization header exists if auth is enabled.
func ValidateAuthHeaderExists(c echo.Context, cfgProvider *config.ConfigProviderWithRefresh) error {
	cfg, err := cfgProvider.GetConfig()
	if err != nil {
		return err
	}

	isEnabled := cfg.Auth.Enabled
	if !isEnabled {
		return nil
	}

	token := c.Request().Header.Get(echo.HeaderAuthorization)
	if token == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	if err := ValidateSessionDuration(c, cfg.Auth.MaxSessionDuration); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	idToken := c.Request().Header.Get(AuthorizationExtrasHeader)
	if tokenVerifier != nil {
		ctx := c.Request().Context()
		if idToken != "" {
			log.Println("[Auth] Validating ID token from Authorization-Extras header")
			if err := validateJWT(ctx, idToken); err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("invalid ID token: %v", err))
			}
		} else {
			log.Println("[Auth] No Authorization-Extras header, validating Authorization header")
			if err := validateJWT(ctx, stripBearerPrefix(token)); err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("invalid token: %v", err))
			}
		}
	}

	if len(cfg.Auth.Providers) > 0 && cfg.Auth.Providers[0].UseIDTokenAsBearer {
		if idToken != "" {
			c.Request().Header.Set(echo.HeaderAuthorization, "Bearer "+idToken)
			c.Request().Header.Del(AuthorizationExtrasHeader)
		}
	}

	return nil
}

func splitCookie(val string) []string {
	splits := []string{}

	var l, r int
	for l, r = 0, cookieLen; r < len(val); l, r = r, r+cookieLen {
		for !utf8.RuneStart(val[r]) {
			r--
		}
		splits = append(splits, val[l:r])
	}
	splits = append(splits, val[l:])
	return splits
}
