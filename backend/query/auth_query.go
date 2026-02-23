package query

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

type GoogleCredential struct {
	gorm.Model
	GoogleID     string    `gorm:"size:128;uniqueIndex;not null"`
	Email        string    `gorm:"size:256;uniqueIndex;not null"`
	Name         string    `gorm:"size:256"`
	Picture      string    `gorm:"size:512"`
	AccessToken  string    `gorm:"type:text;not null"`
	RefreshToken string    `gorm:"type:text"`
	IDToken      string    `gorm:"type:text"`
	TokenExpiry  time.Time `gorm:"not null"`
}

type OAuthState struct {
	State        string    `gorm:"primaryKey;size:128"`
	Completed    bool      `gorm:"not null;default:false"`
	SessionToken string    `gorm:"size:128"`
	ExpiresAt    time.Time `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type OAuthSession struct {
	ID                 uint      `gorm:"primaryKey"`
	SessionToken       string    `gorm:"size:128;uniqueIndex;not null"`
	GoogleCredentialID uint      `gorm:"index;not null"`
	Email              string    `gorm:"size:256;index;not null"`
	ExpiresAt          time.Time `gorm:"not null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func InitDB(path string) error {
	if path == "" {
		path = "storage.db"
	}

	database, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return err
	}

	if err := database.AutoMigrate(&GoogleCredential{}, &OAuthState{}, &OAuthSession{}); err != nil {
		return err
	}

	db = database
	return nil
}

func CreateOAuthState(ttl time.Duration) (string, error) {
	if db == nil {
		return "", errors.New("database not initialized")
	}

	state, err := GenerateSecureToken(32)
	if err != nil {
		return "", err
	}

	record := OAuthState{
		State:     state,
		ExpiresAt: time.Now().Add(ttl),
	}

	if err := db.Create(&record).Error; err != nil {
		return "", err
	}

	return state, nil
}

func GetOAuthState(state string) (*OAuthState, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	var record OAuthState
	if err := db.First(&record, "state = ?", state).Error; err != nil {
		return nil, err
	}

	return &record, nil
}

func CompleteOAuthState(state, sessionToken string) error {
	if db == nil {
		return errors.New("database not initialized")
	}

	return db.Model(&OAuthState{}).Where("state = ?", state).Updates(map[string]any{
		"completed":     true,
		"session_token": sessionToken,
		"updated_at":    time.Now(),
	}).Error
}

func UpsertGoogleCredential(googleID, email, name, picture, accessToken, refreshToken, idToken string, tokenExpiry time.Time) (*GoogleCredential, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	var credential GoogleCredential
	err := db.Where("google_id = ?", googleID).First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		credential = GoogleCredential{
			GoogleID:     googleID,
			Email:        email,
			Name:         name,
			Picture:      picture,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			IDToken:      idToken,
			TokenExpiry:  tokenExpiry,
		}
		if createErr := db.Create(&credential).Error; createErr != nil {
			return nil, createErr
		}
		return &credential, nil
	}
	if err != nil {
		return nil, err
	}

	credential.Email = email
	credential.Name = name
	credential.Picture = picture
	credential.AccessToken = accessToken
	if refreshToken != "" {
		credential.RefreshToken = refreshToken
	}
	credential.IDToken = idToken
	credential.TokenExpiry = tokenExpiry

	if err := db.Save(&credential).Error; err != nil {
		return nil, err
	}

	return &credential, nil
}

func CreateOAuthSession(credentialID uint, email string, ttl time.Duration) (*OAuthSession, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	sessionToken, err := GenerateSecureToken(32)
	if err != nil {
		return nil, err
	}

	session := OAuthSession{
		SessionToken:       sessionToken,
		GoogleCredentialID: credentialID,
		Email:              email,
		ExpiresAt:          time.Now().Add(ttl),
	}

	if err := db.Create(&session).Error; err != nil {
		return nil, err
	}

	return &session, nil
}

func GetOAuthSession(sessionToken string) (*OAuthSession, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	var session OAuthSession
	if err := db.First(&session, "session_token = ?", sessionToken).Error; err != nil {
		return nil, err
	}

	return &session, nil
}

func DeleteOAuthSession(sessionToken string) error {
	if db == nil {
		return errors.New("database not initialized")
	}

	return db.Where("session_token = ?", sessionToken).Delete(&OAuthSession{}).Error
}

func GetGoogleCredentialByID(id uint) (*GoogleCredential, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	var credential GoogleCredential
	if err := db.First(&credential, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &credential, nil
}

func GenerateSecureToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
