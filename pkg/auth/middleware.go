package auth

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/config"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"log"
	"net/http"
	"strings"
	"time"
)

// JWKSFetcher interface for fetching keys from JWKS
type JWKSFetcher interface {
	Register(jwksUrl string, options ...jwk.RegisterOption) error
	Refresh(ctx context.Context, jwksUrl string) (jwk.Set, error)
	Get(ctx context.Context, jwksUrl string) (jwk.Set, error)
	FetchKeys(ctx context.Context, jwksUrl string) (jwk.Set, error)
}

// DefaultJWKSFetcher struct for fetching keys from JWKS
type DefaultJWKSFetcher struct {
	cache *jwk.Cache
}

// NewDefaultJWKSFetcher creates a new JWKSFetcher
func NewDefaultJWKSFetcher(ctx context.Context, jwksUrl string) (*DefaultJWKSFetcher, error) {
	cache := jwk.NewCache(ctx)
	err := cache.Register(jwksUrl, jwk.WithMinRefreshInterval(15*time.Second))
	if err != nil {
		log.Printf("Error registering JWKS URL: %v", err)
		return nil, err
	}
	_, err = cache.Refresh(ctx, jwksUrl)
	if err != nil {
		log.Printf("Error refreshing JWKS cache: %v", err)
		return nil, err
	}
	log.Printf("JWKS cache initialized and refreshed for URL: %s", jwksUrl)
	return &DefaultJWKSFetcher{cache: cache}, nil
}

func (f *DefaultJWKSFetcher) Register(jwksUrl string, options ...jwk.RegisterOption) error {
	log.Printf("Registering JWKS URL: %s", jwksUrl)
	return f.cache.Register(jwksUrl, options...)
}

func (f *DefaultJWKSFetcher) Refresh(ctx context.Context, jwksUrl string) (jwk.Set, error) {
	log.Printf("Refreshing JWKS cache for URL: %s", jwksUrl)
	return f.cache.Refresh(ctx, jwksUrl)
}

func (f *DefaultJWKSFetcher) Get(ctx context.Context, jwksUrl string) (jwk.Set, error) {
	log.Printf("Getting JWKS set for URL: %s", jwksUrl)
	return f.cache.Get(ctx, jwksUrl)
}

func (f *DefaultJWKSFetcher) FetchKeys(ctx context.Context, jwksUrl string) (jwk.Set, error) {
	return f.Get(ctx, jwksUrl)
}

// JWTValidator interface for validating JWT tokens
type JWTValidator interface {
	ParseAndValidateToken(ctx context.Context, tokenString string, set jwk.Set) (jwt.Token, error)
}

// DefaultJWTValidator struct for validating JWT tokens
type DefaultJWTValidator struct{}

func (v *DefaultJWTValidator) ParseAndValidateToken(ctx context.Context, tokenString string, set jwk.Set) (jwt.Token, error) {
	log.Printf("Parsing and validating token")
	return jwt.Parse([]byte(tokenString), jwt.WithKeySet(set), jwt.WithContext(ctx))
}

// JWTMiddleware Middleware for handling JWT authentication.
func JWTMiddleware(jwksUrl string, fetcher JWKSFetcher, validator JWTValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		set, err := fetcher.FetchKeys(ctx, jwksUrl)
		if err != nil {
			log.Printf("Failed to get JWKS: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get JWKS"})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("Authorization header missing")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
			return
		}

		authHeaderParts := strings.SplitN(authHeader, " ", 2)
		if len(authHeaderParts) != 2 || authHeaderParts[0] != "Bearer" {
			log.Printf("Authorization header format must be Bearer {token}")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header format must be Bearer {token}"})
			return
		}

		token, err := validator.ParseAndValidateToken(ctx, authHeaderParts[1], set)
		if err != nil {
			log.Printf("Invalid token: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		scope, ok := token.PrivateClaims()[config.GetAuth().ScopeClaimName].(string)
		if !ok || len(scope) == 0 {
			log.Printf("Scope claim is missing or empty")
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
			log.Printf("Unable to retrieve scopes")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unable to retrieve scopes"})
			return
		}

		requiredPermission, ok := permissions[c.Request.Method]
		if !ok {
			log.Printf("Invalid method: %s", c.Request.Method)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid method"})
			return
		}

		if !strings.Contains(scopes.(string), requiredPermission) {
			log.Printf("Insufficient scope: required permission %s not found in scopes %s", requiredPermission, scopes.(string))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
			return
		}
		c.Next()
	}
}
