package main

import (
	"fern-reporter/config"
	"fern-reporter/pkg/api/routers"
	"fern-reporter/pkg/db"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	initConfig()
	initDb()
	initServer()
}

func initConfig() {
	if err := config.LoadConfig("config/config.yaml"); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func initDb() {
	db.Init()
}

func initServer() {
	serverConfig := config.GetServer()
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "ACCESS_TOKEN"},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}))
	router.LoadHTMLGlob("views/*")
	routers.RegisterRouters(router)
	router.Run(serverConfig.Port)
}
