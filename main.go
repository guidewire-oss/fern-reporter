package main

import (
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

	authConfig := config.GetAuth()
	if err := auth.UpdateJWKS(authConfig.KeysEndpoint); err != nil {
		log.Fatalf("error getting JWKs: %v", err)
	}
	router.Use(auth.JWTAuthMiddleware(authConfig.KeysEndpoint))
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

func CalculateDuration(start, end time.Time) string {
	duration := end.Sub(start)
	return duration.String() // or format as needed
}
