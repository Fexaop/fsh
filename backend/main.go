package main

import (
	"fmt"
	"os"

	"github.com/Fexaop/fsh/backend/appconfig"
	"github.com/Fexaop/fsh/backend/query"
	"github.com/Fexaop/fsh/backend/routes"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	zerolog "github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zlog.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	// load env
	if err := godotenv.Load(); err != nil {
		zlog.Warn().Err(err).Msg("No .env file loaded")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	dbPath := os.Getenv("AUTH_DB_PATH")
	if err := query.InitDB(dbPath); err != nil {
		zlog.Fatal().Err(err).Msg("Failed to initialize auth database")
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.json"
	}
	cfg, err := appconfig.LoadConfig(configPath)
	if err != nil {
		zlog.Fatal().Err(err).Str("path", configPath).Msg("Failed to load runtime config")
	}
	router.Use(routes.AuthRequiredMiddleware(cfg.Auth.MiddlewareEnabled))


	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "200 OK",
		})
	})

	zlog.Info().Msgf("Server is running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		zlog.Fatal().Err(err).Msg("Failed to run server")
	}

	fmt.Printf("Server started on port %s\n", port)
}

