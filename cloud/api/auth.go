// Package api provides authentication handlers for the Cloud Control Plane
package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/UPwith-me/Container-Maker/cloud/db"
)

// AuthRequest represents login/register request body
type AuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name,omitempty"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"`
	TokenType    string   `json:"token_type"`
	User         *db.User `json:"user"`
}

// Register handles user registration
func (s *Server) register(c echo.Context) error {
	var req AuthRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate
	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email and password are required")
	}
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters")
	}

	// Check if user exists
	existing, _ := s.db.GetUserByEmail(req.Email)
	if existing != nil {
		return echo.NewHTTPError(http.StatusConflict, "email already registered")
	}

	// Create user
	user := &db.User{
		ID:        uuid.New().String(),
		Email:     strings.ToLower(req.Email),
		Name:      req.Name,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := user.SetPassword(req.Password); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash password")
	}

	if err := s.db.CreateUser(user); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokenPair(user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate tokens")
	}

	return c.JSON(http.StatusCreated, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // 1 hour
		TokenType:    "Bearer",
		User:         user,
	})
}

// Login handles user authentication
func (s *Server) login(c echo.Context) error {
	var req AuthRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Find user
	user, err := s.db.GetUserByEmail(strings.ToLower(req.Email))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Verify password
	if !user.CheckPassword(req.Password) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Check if active
	if !user.IsActive {
		return echo.NewHTTPError(http.StatusForbidden, "account is disabled")
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokenPair(user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate tokens")
	}

	return c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		User:         user,
	})
}

// RefreshToken handles token refresh
func (s *Server) refreshToken(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate refresh token
	session, err := s.db.GetSessionByToken(req.RefreshToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
	}

	// Get user
	user, err := s.db.GetUserByID(session.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}

	// Delete old session (rotate)
	s.db.DeleteSession(req.RefreshToken)

	// Generate new tokens
	accessToken, refreshToken, err := s.generateTokenPair(user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate tokens")
	}

	return c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		User:         user,
	})
}

// Logout invalidates the current session
func (s *Server) logout(c echo.Context) error {
	// Extract refresh token from body or header
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	c.Bind(&req)

	if req.RefreshToken != "" {
		s.db.DeleteSession(req.RefreshToken)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

// generateTokenPair creates access and refresh tokens
func (s *Server) generateTokenPair(user *db.User) (accessToken, refreshToken string, err error) {
	// Access token (short-lived)
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "container-maker",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err = token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", "", err
	}

	// Refresh token (long-lived, stored in DB)
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return "", "", err
	}
	refreshToken = base64.RawURLEncoding.EncodeToString(refreshBytes)

	// Store session
	session := &db.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		Token:        refreshToken,
		UserAgent:    "",
		IPAddress:    "",
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour), // 30 days
		LastActiveAt: time.Now().UTC(),
	}

	if err := s.db.CreateSession(session); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// validateJWT parses and validates a JWT token, returning the claims
func (s *Server) validateJWT(tokenString string) (*Claims, error) {
	// Remove Bearer prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}
