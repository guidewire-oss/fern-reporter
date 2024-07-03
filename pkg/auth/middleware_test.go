package auth_test

import (
	"fmt"
	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/auth"
	"github.com/guidewire/fern-reporter/pkg/auth/mocks"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("JWTMiddleware", func() {
	var (
		mockFetcher   *mocks.JWKSFetcher
		mockValidator *mocks.JWTValidator
		router        *gin.Engine
		recorder      *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		mockFetcher = new(mocks.JWKSFetcher)
		mockValidator = new(mocks.JWTValidator)
		router = gin.New()
		recorder = httptest.NewRecorder()
	})

	It("should abort with 500 if key fetcher fails", func() {
		mockFetcher.On("FetchKeys", mock.Anything, "test_url").Return(nil, fmt.Errorf("error"))
		router.Use(auth.JWTMiddleware("test_url", mockFetcher, mockValidator))

		req, _ := http.NewRequest("GET", "/", nil)
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
	})

	It("should abort with 401 if authorization header is missing", func() {
		mockFetcher.On("FetchKeys", mock.Anything, "test_url").Return(jwk.NewSet(), nil)
		router.Use(auth.JWTMiddleware("test_url", mockFetcher, mockValidator))

		req, _ := http.NewRequest("GET", "/", nil)
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
	})

	It("should abort with 401 if token is invalid", func() {
		mockFetcher.On("FetchKeys", mock.Anything, "test_url").Return(jwk.NewSet(), nil)
		mockValidator.On("ParseAndValidateToken", mock.Anything, "invalid_token", mock.Anything).Return(nil, fmt.Errorf("invalid token"))
		router.Use(auth.JWTMiddleware("test_url", mockFetcher, mockValidator))

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer invalid_token")
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
	})

	It("should set scope and call next handler if token is valid", func() {
		config.GetAuth().ScopeClaimName = "scope"
		jwkSet := jwk.NewSet()
		mockFetcher.On("FetchKeys", mock.Anything, "test_url").Return(jwkSet, nil)

		mockToken := jwt.New()
		scope := []interface{}{"fern.write", "fernproject.project-a"}
		err := mockToken.Set("scope", scope)
		Expect(err).To(BeNil())

		mockValidator.On("ParseAndValidateToken", mock.Anything, "valid_token", jwkSet).Return(mockToken, nil)

		router.Use(auth.JWTMiddleware("test_url", mockFetcher, mockValidator))
		router.POST("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Create the request body
		body := `{"Project": "project-a"}`
		req, _ := http.NewRequest("POST", "/", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer valid_token")
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(ContainSubstring("success"))
	})
})

var _ = Describe("ScopeMiddleware", func() {
	var (
		router   *gin.Engine
		recorder *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		router = gin.New()
		recorder = httptest.NewRecorder()
	})

	It("should abort with 401 if scope is not set in context", func() {
		router.Use(auth.ScopeMiddleware())
		router.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/", nil)
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
	})

	It("should abort with 403 if method is not in permissions map", func() {
		router.Use(func(c *gin.Context) {
			c.Set("scope", "fern.write")
		})
		router.Use(auth.ScopeMiddleware())
		router.DELETE("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("DELETE", "/", nil)
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusForbidden))
	})

	It("should abort with 403 if scope does not include required permission", func() {
		scopes := []interface{}{"fern.read"}
		router.Use(func(c *gin.Context) {
			c.Set("scope", scopes)
		})
		router.Use(auth.ScopeMiddleware())
		router.POST("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("POST", "/", nil)
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusForbidden))
	})
})
