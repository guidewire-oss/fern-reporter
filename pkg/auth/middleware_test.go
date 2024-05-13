package auth_test

import (
	"github.com/gin-gonic/gin"
	"github.com/guidewire/fern-reporter/pkg/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("JWK Management", func() {
	var (
		server *ghttp.Server
		url    string
	)

	BeforeEach(func() {
		// Create a fake server to simulate JWK provider
		server = ghttp.NewServer()
		url = server.URL()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("UpdateJWKs", func() {
		Context("when fetching JWKs is successful", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
							"keys": []interface{}{
								map[string]interface{}{
									"kty": "RSA",
									"alg": "RS256",
									"kid": "mAMF03ZNwGBz54bjNJLGtlTC9oP8zJSLrpkfBIH1R-E",
									"use": "sig",
									"e":   "AQAB",
									"n":   "2uCExuw6kt86vt28clwQ8d0C1UHMUFUbBlthwiOpTTQYkFSbBUQKBJ16P9pnBrVwVr6-s1-84SKGnJnK6EX6IuiTKJQeEurV67ivoahtZXFBk02fBWd8LrkmDdCE59EsVB8zmHycYMCjm133n1THXjcpjQXKHWmTr3D7mP0jgGZWSdxTgGuWbglX5_OhqEZy7LNQQQYwBnGTsBxCm9Fr6g9b_dWz7l_pXpuVuaesMhL7zahwwCBE6d-tpcN_jhujTT6UhRB63uQsehchAot1BWNdBRsOtQtt4OW9EGqUD8ebVtAt8wchRi6wjCva9MLXQQNWehQftSTRqHZ8HNIOsw",
								},
							},
						}),
					),
				)
			})

			It("should update jwkSet and lastUpdated", func() {
				err := auth.UpdateJWKs(url)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when fetching JWKs fails", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/"),
						ghttp.RespondWith(http.StatusInternalServerError, nil),
					),
				)
			})

			It("should return an error and not update jwkSet", func() {
				err := auth.UpdateJWKs(url)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("JWTAuthMiddleware", func() {
		var (
			router *gin.Engine
			c      *gin.Context
			rec    *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			gin.SetMode(gin.TestMode)
			router = gin.New()
			router.Use(auth.JWTAuthMiddleware(url))

			rec = httptest.NewRecorder()
			c, _ = gin.CreateTestContext(rec)
		})

		Context("when Authorization header is missing", func() {
			It("should return an Unauthorized error", func() {
				req, _ := http.NewRequest("GET", "/", nil)
				router.ServeHTTP(rec, req)
				Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when Authorization header is in the wrong format", func() {
			It("should reject the request with an Unauthorized error", func() {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Add("Authorization", "InvalidTokenFormat")
				router.ServeHTTP(rec, req)
				Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when token is invalid", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{"keys": []interface{}{"key1"}}),
					),
				)
				c.Request, _ = http.NewRequest("GET", "/", nil)
				c.Request.Header.Add("Authorization", "Bearer InvalidToken")
			})

			It("should return an Unauthorized error for invalid token", func() {
				router.ServeHTTP(rec, c.Request)
				Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})

})
