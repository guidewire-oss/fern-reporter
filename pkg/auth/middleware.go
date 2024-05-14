package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	jwkSet      jwk.Set
	lastUpdated time.Time
	jwkMutex    sync.RWMutex
)

func customHTTPClient() *http.Client {
	//FIXME: Fix this client, so that it uses TLS!!!!
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}
}

func fetchJWKS(url string) (jwk.Set, error) {
	ctx := context.Background()
	client := customHTTPClient()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch JWKs, server returned: %d %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return jwk.Parse(data)
}

func UpdateJWKS(url string) error {
	set, err := fetchJWKS(url)
	if err != nil {
		return err
	}

	jwkMutex.Lock()
	defer jwkMutex.Unlock()

	jwkSet = set
	lastUpdated = time.Now()

	return nil
}

func getJWKS(url string) (jwk.Set, error) {
	jwkMutex.RLock()
	defer jwkMutex.RUnlock()

	if time.Since(lastUpdated) > 12*time.Hour {
		jwkMutex.RUnlock()
		err := UpdateJWKS(url)
		jwkMutex.RLock()
		if err != nil {
			return nil, err
		}
	}

	return jwkSet, nil
}
func JWTAuthMiddleware(jwksUrl string) gin.HandlerFunc {
	return func(c *gin.Context) {
		set, err := getJWKS(jwksUrl)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get JWKS"})
			return
		}
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header format must be Bearer {token}"})
			return
		}

		token, err := jwt.Parse([]byte(parts[1]), jwt.WithKeySet(set))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("validatedToken", token)
		c.Next()
	}
}
