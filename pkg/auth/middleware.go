package auth

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/config"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/exp/slices"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const FP = "fernproject"

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
	if err := cache.Register(jwksUrl, jwk.WithMinRefreshInterval(12*time.Hour)); err != nil {
		log.Printf("Error registering JWKS URL: %v", err)
		return nil, err
	}
	if _, err := cache.Refresh(ctx, jwksUrl); err != nil {
		log.Printf("Error refreshing JWKS cache: %v", err)
		return nil, err
	}
	log.Printf("JWKS cache initialized and refreshed for URL: %s", jwksUrl)
	return &DefaultJWKSFetcher{cache: cache}, nil
}

func (f *DefaultJWKSFetcher) Register(jwksUrl string, options ...jwk.RegisterOption) error {
	return f.cache.Register(jwksUrl, options...)
}

func (f *DefaultJWKSFetcher) Refresh(ctx context.Context, jwksUrl string) (jwk.Set, error) {
	return f.cache.Refresh(ctx, jwksUrl)
}

func (f *DefaultJWKSFetcher) Get(ctx context.Context, jwksUrl string) (jwk.Set, error) {
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

		authConfig := config.GetAuth()
		scope, ok := token.PrivateClaims()[authConfig.ScopeClaimName].([]interface{})
		if !ok || len(scope) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "scope claim is missing or empty"})
			return
		}

		c.Set("scope", scope)
		c.Next()
	}
}

type RequestBody struct {
	Project string `json:"project" binding:"required"`
}

// ScopeMiddleware Middleware for checking if the user has the necessary scope for the request.
func ScopeMiddleware() gin.HandlerFunc {
	permissions := map[string]string{
		"POST": "fern.write",
	}

	return func(c *gin.Context) {
		scope, ok := c.Get("scope")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unable to retrieve scope"})
			return
		}

		requiredPermission, ok := permissions[c.Request.Method]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid method"})
			return
		}

		scopes := convertToStringSlice(scope.([]interface{}))

		if !slices.Contains(scopes, requiredPermission) || !containsSubstring(scopes, FP) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
			return
		}

		bodyBytes, err := readRequestBody(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read request body"})
			return
		}

		var requestBody RequestBody
		if err := c.BindJSON(&requestBody); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		fernProjectName, err := validateProjectName(scopes, requestBody.Project)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		c.Set("fernProjectName", fernProjectName)
		c.Next()
	}
}

// readRequestBody reads and returns the request body bytes.
func readRequestBody(c *gin.Context) ([]byte, error) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}

// validateProjectName checks if the project name from the request body matches the scope claim and returns the fernProjectName.
func validateProjectName(scopes []string, projectName string) (string, error) {
	for _, v := range scopes {
		if strings.HasPrefix(v, FP+".") {
			parts := strings.SplitN(v, ".", 2)
			if len(parts) != 2 || len(parts[1]) == 0 {
				return "", fmt.Errorf("fern project scope claim is not formatted properly")
			}

			fernProjectName := parts[1]
			if projectName != fernProjectName {
				return "", fmt.Errorf("project name does not match fern project scope claim")
			}
			return fernProjectName, nil
		}
	}
	return "", fmt.Errorf("fern project scope claim not found")
}

// convertToStringSlice converts a slice of interface{} to a slice of strings.
func convertToStringSlice(slice []interface{}) []string {
	strSlice := make([]string, len(slice))
	for i, v := range slice {
		strSlice[i] = fmt.Sprint(v)
	}
	return strSlice
}

// containsSubstring checks if any string in the slice contains the specified substring.
func containsSubstring(slice []string, substring string) bool {
	for _, v := range slice {
		if strings.Contains(v, substring) {
			return true
		}
	}
	return false
}
