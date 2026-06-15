package middleware

import (
	"net/http"
	"strings"

	"github.com/dilly3/govault/internal/auth"
	"github.com/dilly3/govault/internal/store"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	authenticator auth.Authenticator
	storer        store.Storer
}

func NewAuthMiddleware(authenticator auth.Authenticator, storer store.Storer) *AuthMiddleware {
	return &AuthMiddleware{authenticator: authenticator, storer: storer}
}

// Extracts JWT, validates it, and sets user details in context
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.authenticator.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Inject user context for subsequent handlers to use
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.UserEmail)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// Blocks access if the user's role is not in the allowed list
func (m *AuthMiddleware) RequireRoles(allowedPermissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleInHeader, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not found"})
			return
		}

		userRole := roleInHeader.(string)
		role, err := m.storer.GetRoleStore().GetRoleByName(userRole)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not found"})
			return
		}
		var allowed = false
		for _, permission := range allowedPermissions {
			if role.HasPermission(permission) {
				allowed = true
			}
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}
		c.Next()
	}
}
