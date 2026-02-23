package routes

import (
	"errors"
	"net/http"
	"time"

	"github.com/Fexaop/fsh/backend/query"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var publicAuthPaths = map[string]struct{}{
	"/auth/google/start":    {},
	"/auth/google/callback": {},
	"/callback":             {},
	"/auth/google/session":  {},
	"/auth/logout":          {},
	"/ws":                   {},
}

func AuthRequiredMiddleware(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}

		if _, ok := publicAuthPaths[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		sessionToken := c.GetHeader("X-Session-Token")
		if sessionToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session token"})
			c.Abort()
			return
		}

		session, err := query.GetOAuthSession(sessionToken)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
				c.Abort()
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch session"})
			c.Abort()
			return
		}

		if time.Now().After(session.ExpiresAt) {
			_ = query.DeleteOAuthSession(sessionToken)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			c.Abort()
			return
		}

		c.Next()
	}
}
