package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shubham/trading-platform/gateway-go/internal/auth"
)

const principalKey = "principal"

// JWTAuth validates the Bearer access token and stores the principal in the
// gin context. Reuses the same HS256 secret the Java API signs with.
func JWTAuth(v *auth.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		principal, err := v.Verify(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			// Never leak why: same 401 for expired, wrong-signature, or wrong type.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(principalKey, principal)
		c.Next()
	}
}

// PrincipalFrom returns the authenticated principal set by JWTAuth.
func PrincipalFrom(c *gin.Context) (auth.Principal, bool) {
	v, ok := c.Get(principalKey)
	if !ok {
		return auth.Principal{}, false
	}
	p, ok := v.(auth.Principal)
	return p, ok
}
