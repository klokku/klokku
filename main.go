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

func main() {
	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}
	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
