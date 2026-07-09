package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/coderfeye13/jobtracker/internal/ai"
	"github.com/coderfeye13/jobtracker/internal/gen"
	"github.com/coderfeye13/jobtracker/internal/handler"
	"github.com/coderfeye13/jobtracker/internal/store"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
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
	// CORS must be first so preflight OPTIONS requests are answered before any other middleware runs.
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderContentType},
	}))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	srv := handler.NewServer(st, aiClient)
	gen.RegisterHandlersWithBaseURL(e, srv, "/api/v1")

	log.Fatal(e.Start(":8080"))
}
