package routes

import (
	"errors"
	"time"

	"github.com/Fexaop/fsh/backend/query"
	"gorm.io/gorm"
)

type sessionIdentity struct {
	session    *query.OAuthSession
	credential *query.GoogleCredential
}

func resolveSessionIdentity(sessionToken string) (*sessionIdentity, int, string) {
	if sessionToken == "" {
		return nil, 401, "missing session token"
	}

	session, err := query.GetOAuthSession(sessionToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 401, "invalid session"
		}
		return nil, 500, "failed to fetch session"
	}

	if time.Now().After(session.ExpiresAt) {
		_ = query.DeleteOAuthSession(sessionToken)
		return nil, 401, "session expired"
	}

	credential, err := query.GetGoogleCredentialByID(session.GoogleCredentialID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 401, "invalid session"
		}
		return nil, 500, "failed to fetch user"
	}

	return &sessionIdentity{
		session:    session,
		credential: credential,
	}, 0, ""
}
