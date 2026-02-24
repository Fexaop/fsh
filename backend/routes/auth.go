package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Fexaop/fsh/backend/query"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

const (
	oauthStateTTL   = 10 * time.Minute
	oauthSessionTTL = 30 * 24 * time.Hour
)

type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type authSessionResponse struct {
	Status       string `json:"status"`
	SessionToken string `json:"sessionToken,omitempty"`
	Email        string `json:"email,omitempty"`
	Name         string `json:"name,omitempty"`
	Picture      string `json:"picture,omitempty"`
	ExpiresAt    int64  `json:"expiresAt,omitempty"`
}

type updateProfilePayload struct {
	Name    *string `json:"name"`
	Picture *string `json:"picture"`
}

func RegisterAuthRoutes(router *gin.Engine) {
	router.GET("/auth/google/start", startGoogleAuth)
	router.GET("/auth/google/callback", googleAuthCallback)
	router.GET("/callback", googleAuthCallback)
	router.GET("/auth/google/session", googleSessionByState)
	router.GET("/auth/me", getCurrentUser)
	router.PUT("/auth/me", updateCurrentUserProfile)
	router.POST("/auth/logout", logoutSession)
}

func startGoogleAuth(c *gin.Context) {
	cfg, err := getGoogleOAuthConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	state, err := query.CreateOAuthState(oauthStateTTL)
	if err != nil {
		log.Error().Err(err).Msg("failed to create oauth state")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize oauth flow"})
		return
	}

	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	c.JSON(http.StatusOK, gin.H{
		"authUrl": authURL,
		"state":   state,
	})
}

func googleAuthCallback(c *gin.Context) {
	cfg, err := getGoogleOAuthConfig()
	if err != nil {
		c.String(http.StatusInternalServerError, "OAuth is not configured")
		return
	}

	state := c.Query("state")
	code := c.Query("code")
	if state == "" || code == "" {
		c.String(http.StatusBadRequest, "Missing state or code")
		return
	}

	stateRecord, err := query.GetOAuthState(state)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.String(http.StatusBadRequest, "Invalid auth state")
			return
		}
		log.Error().Err(err).Msg("failed to read oauth state")
		c.String(http.StatusInternalServerError, "Failed to validate auth state")
		return
	}

	if stateRecord.Completed || time.Now().After(stateRecord.ExpiresAt) {
		c.String(http.StatusBadRequest, "Expired or already used auth state")
		return
	}

	token, err := cfg.Exchange(c.Request.Context(), code)
	if err != nil {
		log.Error().Err(err).Msg("token exchange failed")
		c.String(http.StatusBadRequest, "Token exchange failed")
		return
	}

	userinfo, err := fetchGoogleUserInfo(token.AccessToken)
	if err != nil {
		log.Error().Err(err).Msg("failed fetching user info")
		c.String(http.StatusBadRequest, "Failed to fetch Google profile")
		return
	}

	credential, err := query.UpsertGoogleCredential(
		userinfo.ID,
		userinfo.Email,
		userinfo.Name,
		userinfo.Picture,
		token.AccessToken,
		token.RefreshToken,
		extractIDToken(token),
		token.Expiry,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed storing credentials")
		c.String(http.StatusInternalServerError, "Failed to store credentials")
		return
	}

	session, err := query.CreateOAuthSession(credential.ID, credential.Email, oauthSessionTTL)
	if err != nil {
		log.Error().Err(err).Msg("failed creating session")
		c.String(http.StatusInternalServerError, "Failed to create session")
		return
	}

	if err := query.CompleteOAuthState(state, session.SessionToken); err != nil {
		log.Error().Err(err).Msg("failed completing oauth state")
		c.String(http.StatusInternalServerError, "Failed to finalize auth flow")
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, "<h2>Google login successful.</h2><p>You can now return to the fsh app.</p>")
}

func googleSessionByState(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state is required"})
		return
	}

	stateRecord, err := query.GetOAuthState(state)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown state"})
			return
		}
		log.Error().Err(err).Msg("failed to fetch oauth state")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read state"})
		return
	}

	if time.Now().After(stateRecord.ExpiresAt) {
		c.JSON(http.StatusBadRequest, authSessionResponse{Status: "expired"})
		return
	}

	if !stateRecord.Completed || stateRecord.SessionToken == "" {
		c.JSON(http.StatusOK, authSessionResponse{Status: "pending"})
		return
	}

	session, err := query.GetOAuthSession(stateRecord.SessionToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, authSessionResponse{Status: "not_found"})
			return
		}
		log.Error().Err(err).Msg("failed to fetch session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch session"})
		return
	}

	credential, err := query.GetGoogleCredentialByID(session.GoogleCredentialID)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch credential")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, authSessionResponse{
		Status:       "authenticated",
		SessionToken: session.SessionToken,
		Email:        credential.Email,
		Name:         credential.Name,
		Picture:      credential.Picture,
		ExpiresAt:    session.ExpiresAt.Unix(),
	})
}

func getCurrentUser(c *gin.Context) {
	sessionToken := c.GetHeader("X-Session-Token")
	if sessionToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session token"})
		return
	}

	session, err := query.GetOAuthSession(sessionToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		log.Error().Err(err).Msg("failed to fetch session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch session"})
		return
	}

	if time.Now().After(session.ExpiresAt) {
		_ = query.DeleteOAuthSession(sessionToken)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
		return
	}

	credential, err := query.GetGoogleCredentialByID(session.GoogleCredentialID)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch credential")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, authSessionResponse{
		Status:       "authenticated",
		SessionToken: session.SessionToken,
		Email:        credential.Email,
		Name:         credential.Name,
		Picture:      credential.Picture,
		ExpiresAt:    session.ExpiresAt.Unix(),
	})
}

func updateCurrentUserProfile(c *gin.Context) {
	identity, status, errMessage := resolveSessionIdentity(c.GetHeader("X-Session-Token"))
	if identity == nil {
		c.JSON(status, gin.H{"error": errMessage})
		return
	}

	var payload updateProfilePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if payload.Name == nil && payload.Picture == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name or picture is required"})
		return
	}

	nextName := identity.credential.Name
	if payload.Name != nil {
		nextName = strings.TrimSpace(*payload.Name)
		if nextName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
			return
		}
	}

	nextPicture := identity.credential.Picture
	if payload.Picture != nil {
		nextPicture = strings.TrimSpace(*payload.Picture)
	}

	credential, err := query.UpdateGoogleCredentialProfile(identity.credential.ID, nextName, nextPicture)
	if err != nil {
		log.Error().Err(err).Msg("failed to update user profile")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user profile"})
		return
	}

	c.JSON(http.StatusOK, authSessionResponse{
		Status:       "authenticated",
		SessionToken: identity.session.SessionToken,
		Email:        credential.Email,
		Name:         credential.Name,
		Picture:      credential.Picture,
		ExpiresAt:    identity.session.ExpiresAt.Unix(),
	})
}

func logoutSession(c *gin.Context) {
	sessionToken := c.GetHeader("X-Session-Token")
	if sessionToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing session token"})
		return
	}

	if err := query.DeleteOAuthSession(sessionToken); err != nil {
		log.Error().Err(err).Msg("failed to delete session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

func getGoogleOAuthConfig() (*oauth2.Config, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set")
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"openid",
			"email",
			"profile",
		},
		Endpoint: google.Endpoint,
	}, nil
}

func fetchGoogleUserInfo(accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	var userInfo googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	if userInfo.Email == "" || userInfo.ID == "" {
		return nil, errors.New("google profile did not include required fields")
	}

	return &userInfo, nil
}

func extractIDToken(token *oauth2.Token) string {
	idTokenValue := token.Extra("id_token")
	idToken, _ := idTokenValue.(string)
	return idToken
}
