package main

import (
	"context"
	"fmt"
	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/api/routers"
	"github.com/guidewire/fern-reporter/pkg/auth"
	"github.com/guidewire/fern-reporter/pkg/db"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"html/template"
	"log"

	"time"

	"embed"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

//go:embed pkg/views/test_runs.html
var testRunsTemplate embed.FS

func main() {
	initConfig()
	initDb()
	initServer()
}

func initConfig() {
	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func initDb() {
	db.Initialize()
}

func initServer() {
	serverConfig := config.GetServer()
	gin.SetMode(gin.DebugMode)
	router := gin.Default()

	configJWTMiddleware(router)

	router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "ACCESS_TOKEN"},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}))

	funcMap := template.FuncMap{
		"CalculateDuration": CalculateDuration,
	}
	templ, err := template.New("").Funcs(funcMap).ParseFS(testRunsTemplate, "pkg/views/test_runs.html")
	if err != nil {
		log.Fatalf("error parsing templates: %v", err)
	}
	router.SetHTMLTemplate(templ)

	// router.LoadHTMLGlob("pkg/views/*")
	routers.RegisterRouters(router)
	err = router.Run(serverConfig.Port)
	if err != nil {
		log.Fatalf("error starting routes: %v", err)
	}
}

func configJWTMiddleware(router *gin.Engine) {
	authConfig := config.GetAuth()

	if authConfig.Enabled == true {
		ctx := context.Background()

		jwksCache := jwk.NewCache(ctx)
		err := jwksCache.Register(authConfig.JSONWebKeysEndpoint, jwk.WithMinRefreshInterval(12*time.Hour))
		if err != nil {
			log.Fatalf("Failed to register JWKS URL: %v", err)
		}
		if _, err := jwksCache.Refresh(ctx, authConfig.JSONWebKeysEndpoint); err != nil {
			log.Fatalf("URL is not a valid JWKS: %v", err)
		}
		fmt.Println("JWKS cache initialized and refreshed")

		keyFetcher := &auth.DefaultKeyFetcher{}
		jwtValidator := &auth.DefaultJWTValidator{}

		router.Use(auth.JWTMiddleware(authConfig.JSONWebKeysEndpoint, keyFetcher, jwtValidator))
	}
}

func CalculateDuration(start, end time.Time) string {
	duration := end.Sub(start)
	return duration.String() // or format as needed
}
