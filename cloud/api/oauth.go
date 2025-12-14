// Package api provides OAuth handlers for GitHub and Google authentication
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/UPwith-me/Container-Maker/cloud/db"
)

// OAuth endpoints
const (
	GitHubAuthorizeURL = "https://github.com/login/oauth/authorize"
	GitHubTokenURL     = "https://github.com/login/oauth/access_token"
	GitHubUserURL      = "https://api.github.com/user"
	GitHubEmailsURL    = "https://api.github.com/user/emails"

	GoogleAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL     = "https://oauth2.googleapis.com/token"
	GoogleUserURL      = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// GitHubUser represents GitHub API user response
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubEmail represents GitHub email response
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// GoogleUser represents Google API user response
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// githubOAuth initiates GitHub OAuth flow
func (s *Server) githubOAuth(c echo.Context) error {
	state := uuid.New().String() // Should be stored and verified
	redirectURI := s.getOAuthRedirectURI(c, "github")

	params := url.Values{
		"client_id":    {s.config.GitHubClientID},
		"redirect_uri": {redirectURI},
		"scope":        {"user:email read:user"},
		"state":        {state},
	}

	authURL := GitHubAuthorizeURL + "?" + params.Encode()
	return c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// githubCallback handles GitHub OAuth callback
func (s *Server) githubCallback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing code parameter")
	}

	// Exchange code for token
	redirectURI := s.getOAuthRedirectURI(c, "github")
	tokenResp, err := s.exchangeGitHubCode(code, redirectURI)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to exchange code: "+err.Error())
	}

	// Get user info
	ghUser, err := s.getGitHubUser(tokenResp.AccessToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user info: "+err.Error())
	}

	// Get primary email if not public
	email := ghUser.Email
	if email == "" {
		emails, err := s.getGitHubEmails(tokenResp.AccessToken)
		if err == nil {
			for _, e := range emails {
				if e.Primary && e.Verified {
					email = e.Email
					break
				}
			}
		}
	}

	if email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "no verified email found on GitHub account")
	}

	// Find or create user
	user, err := s.findOrCreateOAuthUser("github", fmt.Sprintf("%d", ghUser.ID), email, ghUser.Name, ghUser.AvatarURL)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user: "+err.Error())
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokenPair(user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate tokens")
	}

	// For web, redirect with tokens; for CLI, return JSON
	if c.QueryParam("cli") == "true" {
		return c.HTML(http.StatusOK, fmt.Sprintf(`
			<html>
			<body>
				<h1>Authentication Successful</h1>
				<p>You can close this window and return to the CLI.</p>
				<script>
					// For CLI callback
					window.opener && window.opener.postMessage({
						access_token: "%s",
						refresh_token: "%s"
					}, "*");
				</script>
			</body>
			</html>
		`, accessToken, refreshToken))
	}

	// Redirect to frontend with tokens
	frontendURL := fmt.Sprintf("/auth/callback?access_token=%s&refresh_token=%s", accessToken, refreshToken)
	return c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}

// googleOAuth initiates Google OAuth flow
func (s *Server) googleOAuth(c echo.Context) error {
	state := uuid.New().String()
	redirectURI := s.getOAuthRedirectURI(c, "google")

	params := url.Values{
		"client_id":     {s.config.GoogleClientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
	}

	authURL := GoogleAuthorizeURL + "?" + params.Encode()
	return c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// googleCallback handles Google OAuth callback
func (s *Server) googleCallback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing code parameter")
	}

	redirectURI := s.getOAuthRedirectURI(c, "google")

	// Exchange code for token
	tokenResp, err := s.exchangeGoogleCode(code, redirectURI)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to exchange code: "+err.Error())
	}

	// Get user info
	googleUser, err := s.getGoogleUser(tokenResp.AccessToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user info: "+err.Error())
	}

	if !googleUser.VerifiedEmail {
		return echo.NewHTTPError(http.StatusBadRequest, "email not verified on Google account")
	}

	// Find or create user
	user, err := s.findOrCreateOAuthUser("google", googleUser.ID, googleUser.Email, googleUser.Name, googleUser.Picture)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user: "+err.Error())
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokenPair(user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate tokens")
	}

	frontendURL := fmt.Sprintf("/auth/callback?access_token=%s&refresh_token=%s", accessToken, refreshToken)
	return c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}

// Helper functions

func (s *Server) getOAuthRedirectURI(c echo.Context, provider string) string {
	scheme := "https"
	if c.Request().TLS == nil {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/api/v1/auth/%s/callback", scheme, c.Request().Host, provider)
}

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (s *Server) exchangeGitHubCode(code, redirectURI string) (*OAuthTokenResponse, error) {
	data := url.Values{
		"client_id":     {s.config.GitHubClientID},
		"client_secret": {s.config.GitHubClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	req, _ := http.NewRequest("POST", GitHubTokenURL, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (s *Server) getGitHubUser(accessToken string) (*GitHubUser, error) {
	req, _ := http.NewRequest("GET", GitHubUserURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Server) getGitHubEmails(accessToken string) ([]GitHubEmail, error) {
	req, _ := http.NewRequest("GET", GitHubEmailsURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	return emails, nil
}

func (s *Server) exchangeGoogleCode(code, redirectURI string) (*OAuthTokenResponse, error) {
	data := url.Values{
		"client_id":     {s.config.GoogleClientID},
		"client_secret": {s.config.GoogleClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	resp, err := http.PostForm(GoogleTokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (s *Server) getGoogleUser(accessToken string) (*GoogleUser, error) {
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", GoogleUserURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Server) findOrCreateOAuthUser(provider, providerID, email, name, avatarURL string) (*db.User, error) {
	// Try to find by provider ID
	var user *db.User
	var err error

	switch provider {
	case "github":
		user, err = s.db.GetUserByEmail(email)
		if err == nil && user.GitHubID == "" {
			// Link GitHub to existing account
			user.GitHubID = providerID
			user.AvatarURL = avatarURL
			_ = s.db.UpdateUser(user)
		}
	case "google":
		user, err = s.db.GetUserByEmail(email)
		if err == nil && user.GoogleID == "" {
			// Link Google to existing account
			user.GoogleID = providerID
			user.AvatarURL = avatarURL
			_ = s.db.UpdateUser(user)
		}
	}

	if user != nil {
		return user, nil
	}

	// Create new user
	user = &db.User{
		ID:            uuid.New().String(),
		Email:         strings.ToLower(email),
		Name:          name,
		AvatarURL:     avatarURL,
		EmailVerified: true, // OAuth emails are verified by the provider
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	switch provider {
	case "github":
		user.GitHubID = providerID
	case "google":
		user.GoogleID = providerID
	}

	if err := s.db.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}
