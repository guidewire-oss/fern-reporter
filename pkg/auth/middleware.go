package auth

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"net/http"
	"strings"
)

// Fetches the JWK set from the given URL using the provided context and cache.
func getKeys(ctx context.Context, jwksUrl string, jwksCache jwk.Cache) (jwk.Set, error) {
	return jwksCache.Get(ctx, jwksUrl)
}

// Parses and validates the JWT token from the Authorization header.
func parseAndValidateToken(c *gin.Context, set jwk.Set) (jwt.Token, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header missing")
	}

	authHeaderParts := strings.SplitN(authHeader, " ", 2)
	if len(authHeaderParts) != 2 || authHeaderParts[0] != "Bearer" {
		return nil, fmt.Errorf("authorization header format must be Bearer {token}")
	}

	token, err := jwt.Parse([]byte(authHeaderParts[1]), jwt.WithKeySet(set))
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

// JWTMiddleware Middleware for handling JWT authentication.
func JWTMiddleware(jwksUrl string, jwksCache jwk.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		set, err := getKeys(ctx, jwksUrl, jwksCache)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get JWKS"})
			return
		}

		token, err := parseAndValidateToken(c, set)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		fernScope, ok := token.PrivateClaims()["fern_scope"].(string)
		if !ok || len(fernScope) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "fern scope claim is missing or empty"})
			return
		}

		c.Set("fernScope", fernScope)
		c.Next()
	}
}

// ScopeMiddleware Middleware for checking if the user has the necessary scope for the request.
func ScopeMiddleware() gin.HandlerFunc {
	permissions := map[string]string{
		"GET":    "read",
		"POST":   "write",
		"PUT":    "write",
		"UPDATE": "write",
		"DELETE": "write",
	}

	return func(c *gin.Context) {
		scopes, ok := c.Get("fernScope")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unable to retrieve scopes"})
			return
		}

		pathAppID := c.Param("appID")
		requestMethod := c.Request.Method

		requiredPermission, ok := permissions[requestMethod]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid method"})
			return
		}

		requiredScope := pathAppID + "." + requiredPermission

		if !strings.Contains(scopes.(string), requiredScope) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
			return
		}
		c.Next()
	}
}
