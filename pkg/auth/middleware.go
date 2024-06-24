package auth

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"net/http"
	"os"
	"strings"
)

type KeyFetcher interface {
	FetchKeys(ctx context.Context, jwksUrl string) (jwk.Set, error)
}

type JWTValidator interface {
	ParseAndValidateToken(ctx context.Context, tokenString string, set jwk.Set) (jwt.Token, error)
}

type DefaultKeyFetcher struct{}

func (f *DefaultKeyFetcher) FetchKeys(ctx context.Context, jwksUrl string) (jwk.Set, error) {
	return jwk.Fetch(ctx, jwksUrl)
}

type DefaultJWTValidator struct{}

func (v *DefaultJWTValidator) ParseAndValidateToken(ctx context.Context, tokenString string, set jwk.Set) (jwt.Token, error) {
	return jwt.Parse([]byte(tokenString), jwt.WithKeySet(set))
}

// JWTMiddleware Middleware for handling JWT authentication.
func JWTMiddleware(jwksUrl string, fetcher KeyFetcher, validator JWTValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		set, err := fetcher.FetchKeys(ctx, jwksUrl)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get JWKS"})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
			return
		}

		authHeaderParts := strings.SplitN(authHeader, " ", 2)
		if len(authHeaderParts) != 2 || authHeaderParts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header format must be Bearer {token}"})
			return
		}

		token, err := validator.ParseAndValidateToken(ctx, authHeaderParts[1], set)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		scopeClaimName := os.Getenv("SCOPE_CLAIM_NAME")
		if scopeClaimName == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "SCOPE_CLAIM_NAME environment variable is empty",
			})
			return
		}

		scope, ok := token.PrivateClaims()[scopeClaimName].(string)
		if !ok || len(scope) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "scope claim is missing or empty"})
			return
		}

		c.Set("scope", scope)
		c.Next()
	}
}

// ScopeMiddleware Middleware for checking if the user has the necessary scope for the request.
func ScopeMiddleware() gin.HandlerFunc {
	permissions := map[string]string{
		"POST": "fern.write",
	}

	return func(c *gin.Context) {
		scopes, ok := c.Get("scope")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unable to retrieve scopes"})
			return
		}

		requiredPermission, ok := permissions[c.Request.Method]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid method"})
			return
		}

		if !strings.Contains(scopes.(string), requiredPermission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
			return
		}
		c.Next()
	}
}
