package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/guidewire/fern-reporter/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Auth Middleware", func() {
	var (
		jwksCache *jwk.Cache
		jwksUrl   string
		recorder  *httptest.ResponseRecorder
		router    *gin.Engine
	)

	// Create a mock JWKS response
	mockJWKSResponse := `{
		"keys": [
			{
				"kty": "RSA",
				"kid": "fake-key-id",
				"use": "sig",
				"n": "fake-modulus",
				"e": "AQAB"
			}
		]
	}`

	ginkgo.BeforeEach(func() {
		// Create a mock HTTP server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockJWKSResponse))
		}))
		jwksUrl = mockServer.URL

		// Initialize JWKS cache with the mock server URL
		ctx := context.Background()
		jwksCache = jwk.NewCache(ctx)
		err := jwksCache.Register(jwksUrl, jwk.WithMinRefreshInterval(12*time.Hour))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Set up Gin router
		gin.SetMode(gin.TestMode)
		router = gin.New()
	})

	ginkgo.Context("JWTMiddleware", func() {
		ginkgo.BeforeEach(func() {
			router.Use(auth.JWTMiddleware(jwksUrl, *jwksCache))
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
		})

		ginkgo.It("should return 401 if Authorization header is missing", func() {
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusUnauthorized))
		})

		ginkgo.It("should return 401 if token is invalid", func() {
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer invalid.token")
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusUnauthorized))
		})

		/*ginkgo.FIt("should return 200 if token is valid", func() {
			// Generate a valid token
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"exp":        time.Now().Add(time.Hour).Unix(),
				"iat":        time.Now().Unix(),
				"iss":        "test",
				"fern_scope": "appID.read",
			})

			pk, err := rsa.GenerateKey(rand.Reader, 2048)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			token.Header["kid"] = "fake-key-id"
			tokenString, err := token.SignedString(pk)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// Create a request with the valid token
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
			gomega.Expect(recorder.Body.String()).To(gomega.ContainSubstring("success"))
		})*/
	})

	ginkgo.Context("ScopeMiddleware", func() {
		ginkgo.BeforeEach(func() {
			router.Use(func(c *gin.Context) {
				c.Set("fernScope", "appID.read")
			})
			router.Use(auth.ScopeMiddleware())
			router.GET("/app/:appID", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
		})

		ginkgo.It("should return 403 if scope is insufficient", func() {
			req, _ := http.NewRequest(http.MethodGet, "/app/anotherAppID", nil)
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusForbidden))
		})

		ginkgo.It("should allow access if scope is sufficient", func() {
			req, _ := http.NewRequest(http.MethodGet, "/app/appID", nil)
			recorder = httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
		})

	})
})
