package main

import (
	"context"
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/coderfeye13/jobtracker/internal/ai"
	"github.com/coderfeye13/jobtracker/internal/gen"
	"github.com/coderfeye13/jobtracker/internal/handler"
	"github.com/coderfeye13/jobtracker/internal/store"
)

func main() {
	st, err := store.New("jobtracker.db")
	if err != nil {
		log.Fatal(err)
	}

	var aiClient *ai.Client
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		aiClient, err = ai.NewClient(context.Background(), key)
		if err != nil {
			log.Fatalf("AI client init failed: %v", err)
		}
	} else {
		log.Println("GEMINI_API_KEY not set — /ai/parse-job will be unavailable")
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	srv := handler.NewServer(st, aiClient)
	gen.RegisterHandlersWithBaseURL(e, srv, "/api/v1")

	log.Fatal(e.Start(":8080"))
}
