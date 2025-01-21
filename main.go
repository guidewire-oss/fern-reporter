package main

import (
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/guidewire/fern-reporter/pkg/graph/generated"
	"github.com/guidewire/fern-reporter/pkg/graph/resolvers"
	"github.com/guidewire/fern-reporter/pkg/utils"
	"gorm.io/gorm"

	"context"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/api/routers"
	"github.com/guidewire/fern-reporter/pkg/auth"
	"github.com/guidewire/fern-reporter/pkg/db"
	"html/template"
	"log"

	"time"

	"embed"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

//go:embed pkg/views/test_runs.html
//go:embed pkg/views/insights.html
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

	if config.GetAuth().Enabled {
		checkAuthConfig()
		configJWTMiddleware(router)
	} else {
		log.Println("Auth is disabled, JWT Middleware is not configured.")
	}

	router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "ACCESS_TOKEN"},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}))

	funcMap := template.FuncMap{
		"CalculateDuration": utils.CalculateDuration,
		"FormatDate":        utils.FormatDate,
	}

	templ, err := template.New("").Funcs(funcMap).ParseFS(testRunsTemplate, "pkg/views/test_runs.html", "pkg/views/insights.html")
	if err != nil {
		log.Fatalf("error parsing templates: %v", err)
	}
	router.SetHTMLTemplate(templ)

	// router.LoadHTMLGlob("pkg/views/*")
	routers.RegisterRouters(router)

	router.POST("/query", GraphqlHandler(db.GetDb()))
	router.GET("/", PlaygroundHandler("/query"))
	err = router.Run(serverConfig.Port)
	if err != nil {
		log.Fatalf("error starting routes: %v", err)
	}

}

func PlaygroundHandler(path string) gin.HandlerFunc {
	h := playground.Handler("GraphQL playground", path)
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func GraphqlHandler(gormdb *gorm.DB) gin.HandlerFunc {
	h := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers.Resolver{DB: gormdb}}))
	h.AddTransport(transport.POST{})

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func checkAuthConfig() {
	if config.GetAuth().ScopeClaimName == "" {
		log.Fatal("Set SCOPE_CLAIM_NAME environment variable or add a default value in config.yaml")
	}
	if config.GetAuth().JSONWebKeysEndpoint == "" {
		log.Fatal("Set AUTH_JSON_WEB_KEYS_ENDPOINT environment variable or add a default value in config.yaml")
	}
}

func configJWTMiddleware(router *gin.Engine) {
	authConfig := config.GetAuth()
	ctx := context.Background()

	keyFetcher, err := auth.NewDefaultJWKSFetcher(ctx, authConfig.JSONWebKeysEndpoint)
	if err != nil {
		log.Fatalf("Failed to create JWKS fetcher: %v", err)
	}

	jwtValidator := &auth.DefaultJWTValidator{}

	router.Use(auth.JWTMiddleware(authConfig.JSONWebKeysEndpoint, keyFetcher, jwtValidator))
	log.Println("JWT Middleware configured successfully.")
}
