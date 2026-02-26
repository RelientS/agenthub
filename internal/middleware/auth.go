package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Custom JWT claims for AgentHub tokens.
type AgentClaims struct {
	AgentID     uuid.UUID `json:"agent_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	jwt.RegisteredClaims
}

// AuthMiddleware returns a Gin middleware that validates JWT tokens from the
// Authorization header and sets agent_id and workspace_id in the context.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "missing authorization header",
				},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "invalid authorization header format, expected 'Bearer <token>'",
				},
			})
			return
		}

		tokenString := parts[1]

		claims := &AgentClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "invalid or expired token",
				},
			})
			return
		}

		// Set agent and workspace IDs in the context for downstream handlers.
		c.Set("agent_id", claims.AgentID)
		c.Set("workspace_id", claims.WorkspaceID)

		c.Next()
	}
}

// GenerateToken creates a signed JWT token for the given agent and workspace.
func GenerateToken(agentID, workspaceID uuid.UUID, secret string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := AgentClaims{
		AgentID:     agentID,
		WorkspaceID: workspaceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "agenthub",
			Subject:   agentID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GetAgentID extracts the agent_id from the Gin context. Returns uuid.Nil if not set.
func GetAgentID(c *gin.Context) uuid.UUID {
	val, exists := c.Get("agent_id")
	if !exists {
		return uuid.Nil
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// GetWorkspaceID extracts the workspace_id from the Gin context. Returns uuid.Nil if not set.
func GetWorkspaceID(c *gin.Context) uuid.UUID {
	val, exists := c.Get("workspace_id")
	if !exists {
		return uuid.Nil
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
