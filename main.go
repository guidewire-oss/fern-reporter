package main

import (
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/google/uuid"
	"github.com/guidewire/fern-reporter/pkg/graph/generated"
	"github.com/guidewire/fern-reporter/pkg/graph/resolvers"
	"github.com/guidewire/fern-reporter/pkg/utils"
	"github.com/mileusna/useragent"
	"gorm.io/gorm"

	"context"
	"html/template"
	"log"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/guidewire/fern-reporter/config"
	"github.com/guidewire/fern-reporter/pkg/api/routers"
	"github.com/guidewire/fern-reporter/pkg/auth"
	"github.com/guidewire/fern-reporter/pkg/db"

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
	initLogger()
	initDb()
	initServer()
}

func initConfig() {
	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Println("[LOG]: Configuration loaded successfully")
}

func initDb() {
	db.Initialize()
	utils.GetLogger().Info("[LOG]: Database initialized successfully")
}

func initLogger() {
	utils.InitLoggerOnce()
	utils.GetLogger().Info("[LOG]: Logging Service Configured Successfully")
}

func initServer() {
	serverConfig := config.GetServer()
	gin.SetMode(gin.ReleaseMode) // Set Gin mode to ReleaseMode to suppress debug logs
	router := gin.New()
	router.Use(gin.Recovery())

	// Add cookie middleware BEFORE routes
	router.Use(SetMiddlewareCookie())

	if config.GetAuth().Enabled {
		checkAuthConfig()
		configJWTMiddleware(router)
	} else {
		utils.GetLogger().Info("[LOG]: Authentication is disabled, JWT Middleware not configured")
	}

	router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "DELETE", "PUT"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "ACCESS_TOKEN", "User-Agent"},
		AllowCredentials: true,
		AllowOriginFunc:  isAllowedOrigin,
		MaxAge:           12 * time.Hour,
	}))

	funcMap := template.FuncMap{
		"CalculateDuration": utils.CalculateDuration,
		"FormatDate":        utils.FormatDate,
	}

	templ, err := template.New("").Funcs(funcMap).ParseFS(testRunsTemplate, "pkg/views/test_runs.html", "pkg/views/insights.html")
	if err != nil {
		utils.GetLogger().Fatal("[ERROR]: Unable to parse HTML templates: ", err)
	}
	router.SetHTMLTemplate(templ)

	// router.LoadHTMLGlob("pkg/views/*")
	routers.RegisterRouters(router)

	router.POST("/query", GraphqlHandler(db.GetDb()))
	router.GET("/", PlaygroundHandler("/query"))
	err = router.Run(serverConfig.Port)
	if err != nil {
		utils.GetLogger().Fatal("[ERROR]: Unable to start the server: ", err)
	}
	utils.GetLogger().Info(fmt.Sprintf("[LOG]: Server started successfully on port %s", serverConfig.Port))
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
		utils.GetLogger().Fatal("[ERROR]: Set SCOPE_CLAIM_NAME environment variable or add a default value in config.yaml", nil)
	}
	if config.GetAuth().JSONWebKeysEndpoint == "" {
		utils.GetLogger().Fatal("[ERROR]: Set AUTH_JSON_WEB_KEYS_ENDPOINT environment variable or add a default value in config.yaml", nil)
	}
}

func configJWTMiddleware(router *gin.Engine) {
	authConfig := config.GetAuth()
	ctx := context.Background()

	keyFetcher, err := auth.NewDefaultJWKSFetcher(ctx, authConfig.JSONWebKeysEndpoint)
	if err != nil {
		utils.GetLogger().Fatal("[ERROR]: Failed to create JWKS fetcher", err)
	}

	jwtValidator := &auth.DefaultJWTValidator{}

	router.Use(auth.JWTMiddleware(authConfig.JSONWebKeysEndpoint, keyFetcher, jwtValidator))
	utils.GetLogger().Info("[LOG]: JWT Middleware configured successfully")
}

func isAllowedOrigin(origin string) bool {
	origin = strings.ToLower(origin)

	if strings.Contains(origin, "localhost") || strings.HasPrefix(origin, "https://fern") {
		return true
	}
	return false
}

func SetMiddlewareCookie() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := c.Cookie(utils.CookieName)
		if err != nil {
			// Cookie not found, generate and set
			newUUID := uuid.New().String()
			ua := useragent.Parse(c.GetHeader("User-Agent"))
			// Check if it's a browser (not a bot, and has browser name)
			if ua.Name != "" && !ua.Bot {
				c.SetCookie(
					utils.CookieName,
					newUUID,
					int(100*365*24*time.Hour.Seconds()), // 100 years
					"/",
					"",
					false, // secure (set to true if using HTTPS)
					true,  // httpOnly
				)
			}
		}
		c.Next()
	}
}
