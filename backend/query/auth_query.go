package query

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
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

type FamilyMember struct {
	ID                uint   `gorm:"primaryKey"`
	OwnerCredentialID uint   `gorm:"index:idx_owner_member_email,priority:1;not null"`
	Name              string `gorm:"size:256;not null"`
	Email             string `gorm:"size:256;index:idx_owner_member_email,priority:2,unique;not null"`
	Relation          string `gorm:"size:128;not null"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type FamilyInvitation struct {
	ID                  uint      `gorm:"primaryKey"`
	InviteID            string    `gorm:"size:32;uniqueIndex;not null"`
	InviterCredentialID uint      `gorm:"index;not null"`
	InviterEmail        string    `gorm:"size:256;index;not null"`
	InviterName         string    `gorm:"size:256"`
	Relation            string    `gorm:"size:128;not null"`
	Status              string    `gorm:"size:32;index;not null"`
	InviteeCredentialID *uint     `gorm:"index"`
	InviteeEmail        string    `gorm:"size:256;index"`
	ExpiresAt           time.Time `gorm:"index;not null"`
	RespondedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

const (
	FamilyInvitationStatusPending  = "pending"
	FamilyInvitationStatusAccepted = "accepted"
	FamilyInvitationStatusDeclined = "declined"
)

const inviteIDAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var (
	ErrInvalidInvitationAction = errors.New("invalid invitation action")
	ErrInvitationNotPending    = errors.New("invitation is not pending")
	ErrInvitationExpired       = errors.New("invitation has expired")
	ErrInvitationSelfResponse  = errors.New("cannot respond to own invitation")
)

func InitDB(path string) error {
	if path == "" {
		path = "storage.db"
	}

	database, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return err
	}

	if err := database.AutoMigrate(&GoogleCredential{}, &OAuthState{}, &OAuthSession{}, &FamilyMember{}, &FamilyInvitation{}); err != nil {
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

func UpdateGoogleCredentialProfile(id uint, name, picture string) (*GoogleCredential, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	var credential GoogleCredential
	if err := db.First(&credential, "id = ?", id).Error; err != nil {
		return nil, err
	}

	credential.Name = strings.TrimSpace(name)
	credential.Picture = strings.TrimSpace(picture)

	if err := db.Save(&credential).Error; err != nil {
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

func ListFamilyMembers(ownerCredentialID uint) ([]FamilyMember, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	var members []FamilyMember
	if err := db.Where("owner_credential_id = ?", ownerCredentialID).Order("name ASC").Find(&members).Error; err != nil {
		return nil, err
	}

	return members, nil
}

func CreateFamilyMember(ownerCredentialID uint, name, email, relation string) (*FamilyMember, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	member := FamilyMember{
		OwnerCredentialID: ownerCredentialID,
		Name:              strings.TrimSpace(name),
		Email:             strings.ToLower(strings.TrimSpace(email)),
		Relation:          strings.TrimSpace(relation),
	}

	if err := db.Create(&member).Error; err != nil {
		return nil, err
	}

	return &member, nil
}

func UpdateFamilyMember(ownerCredentialID, memberID uint, name, email, relation string) (*FamilyMember, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	var member FamilyMember
	if err := db.Where("id = ? AND owner_credential_id = ?", memberID, ownerCredentialID).First(&member).Error; err != nil {
		return nil, err
	}

	member.Name = strings.TrimSpace(name)
	member.Email = strings.ToLower(strings.TrimSpace(email))
	member.Relation = strings.TrimSpace(relation)

	if err := db.Save(&member).Error; err != nil {
		return nil, err
	}

	return &member, nil
}

func DeleteFamilyMember(ownerCredentialID, memberID uint) error {
	if db == nil {
		return errors.New("database not initialized")
	}

	result := db.Where("id = ? AND owner_credential_id = ?", memberID, ownerCredentialID).Delete(&FamilyMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func CreateFamilyInvitation(inviter *GoogleCredential, relation string, ttl time.Duration) (*FamilyInvitation, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}
	if inviter == nil {
		return nil, errors.New("inviter is required")
	}

	relation = strings.TrimSpace(relation)
	if relation == "" {
		relation = "Family"
	}

	for range 8 {
		inviteID, err := GenerateInviteID(10)
		if err != nil {
			return nil, err
		}

		invitation := FamilyInvitation{
			InviteID:            inviteID,
			InviterCredentialID: inviter.ID,
			InviterEmail:        strings.ToLower(strings.TrimSpace(inviter.Email)),
			InviterName:         strings.TrimSpace(inviter.Name),
			Relation:            relation,
			Status:              FamilyInvitationStatusPending,
			ExpiresAt:           time.Now().Add(ttl),
		}

		if err := db.Create(&invitation).Error; err != nil {
			if isUniqueConstraintError(err) {
				continue
			}
			return nil, err
		}

		return &invitation, nil
	}

	return nil, errors.New("failed to generate unique invitation id")
}

func ListOutgoingFamilyInvitations(inviterCredentialID uint) ([]FamilyInvitation, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	var invitations []FamilyInvitation
	if err := db.Where("inviter_credential_id = ?", inviterCredentialID).
		Order("created_at DESC").
		Find(&invitations).Error; err != nil {
		return nil, err
	}

	return invitations, nil
}

func GetFamilyInvitationByInviteID(inviteID string) (*FamilyInvitation, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}

	normalizedInviteID := strings.ToUpper(strings.TrimSpace(inviteID))
	var invitation FamilyInvitation
	if err := db.First(&invitation, "invite_id = ?", normalizedInviteID).Error; err != nil {
		return nil, err
	}

	return &invitation, nil
}

func RespondToFamilyInvitation(inviteID string, invitee *GoogleCredential, action string) (*FamilyInvitation, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}
	if invitee == nil {
		return nil, errors.New("invitee is required")
	}

	action = strings.ToLower(strings.TrimSpace(action))
	if action != "accept" && action != "decline" {
		return nil, ErrInvalidInvitationAction
	}

	normalizedInviteID := strings.ToUpper(strings.TrimSpace(inviteID))
	var updatedInvitation FamilyInvitation
	err := db.Transaction(func(tx *gorm.DB) error {
		var invitation FamilyInvitation
		if err := tx.First(&invitation, "invite_id = ?", normalizedInviteID).Error; err != nil {
			return err
		}

		if invitation.Status != FamilyInvitationStatusPending {
			return ErrInvitationNotPending
		}
		if time.Now().After(invitation.ExpiresAt) {
			return ErrInvitationExpired
		}
		if invitation.InviterCredentialID == invitee.ID {
			return ErrInvitationSelfResponse
		}

		now := time.Now()
		invitation.InviteeCredentialID = &invitee.ID
		invitation.InviteeEmail = strings.ToLower(strings.TrimSpace(invitee.Email))
		invitation.RespondedAt = &now

		if action == "accept" {
			invitation.Status = FamilyInvitationStatusAccepted

			if err := ensureFamilyMemberTx(
				tx,
				invitation.InviterCredentialID,
				invitee.Name,
				invitee.Email,
				invitation.Relation,
			); err != nil {
				return err
			}

			inviterDisplayName := strings.TrimSpace(invitation.InviterName)
			if inviterDisplayName == "" {
				inviterDisplayName = invitation.InviterEmail
			}
			if err := ensureFamilyMemberTx(
				tx,
				invitee.ID,
				inviterDisplayName,
				invitation.InviterEmail,
				"Family",
			); err != nil {
				return err
			}
		} else {
			invitation.Status = FamilyInvitationStatusDeclined
		}

		if err := tx.Save(&invitation).Error; err != nil {
			return err
		}

		updatedInvitation = invitation
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &updatedInvitation, nil
}

func GenerateInviteID(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invite id length must be positive")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	var b strings.Builder
	b.Grow(length)
	for _, value := range bytes {
		b.WriteByte(inviteIDAlphabet[int(value)%len(inviteIDAlphabet)])
	}

	return b.String(), nil
}

func ensureFamilyMemberTx(tx *gorm.DB, ownerCredentialID uint, name, email, relation string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	relation = strings.TrimSpace(relation)

	if email == "" {
		return errors.New("family member email is required")
	}
	if name == "" {
		name = email
	}
	if relation == "" {
		relation = "Family"
	}

	var existing FamilyMember
	err := tx.Where("owner_credential_id = ? AND email = ?", ownerCredentialID, email).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		member := FamilyMember{
			OwnerCredentialID: ownerCredentialID,
			Name:              name,
			Email:             email,
			Relation:          relation,
		}
		return tx.Create(&member).Error
	}
	if err != nil {
		return err
	}

	existing.Name = name
	existing.Relation = relation
	return tx.Save(&existing).Error
}

func isUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}
