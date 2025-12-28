package main

import (
	"os"

	"github.com/klokku/klokku/internal/app"
	log "github.com/sirupsen/logrus"
)

func init() {
	level := os.Getenv("LOG_LEVEL")
	if level != "" {
		logrusLevel, err := log.ParseLevel(level)
		if err != nil {
			log.Fatal(err)
		}
		log.SetLevel(logrusLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

// @title Klokku API
// @version 0.9
// @description Time planning and tracking API
// @host localhost:8181
// @BasePath /
// @securityDefinitions.apikey XUserId
// @in header
// @name X-User-Id
// @description User ID header required for authentication
func main() {
	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}
	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
