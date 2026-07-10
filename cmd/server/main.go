package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	gmailv1 "google.golang.org/api/gmail/v1"

	"github.com/coderfeye13/jobtracker/internal/ai"
	"github.com/coderfeye13/jobtracker/internal/gen"
	"github.com/coderfeye13/jobtracker/internal/gmail"
	"github.com/coderfeye13/jobtracker/internal/handler"
	"github.com/coderfeye13/jobtracker/internal/store"
	syncpkg "github.com/coderfeye13/jobtracker/internal/sync"
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

	gmailSvc := initGmail()
	syncer := syncpkg.New(st, gmailSvc, aiClient)
	if gmailSvc != nil {
		go runInboxSyncLoop(syncer)
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

	srv := handler.NewServer(st, aiClient, gmailSvc, syncer)
	gen.RegisterHandlersWithBaseURL(e, srv, "/api/v1")

	log.Fatal(e.Start(":8080"))
}

// initGmail sets up the Gmail client if credentials.json is present in the
// repo root. A missing file is not fatal: the server starts with a nil
// client and the /inbox endpoints return 503 until it's added. On first
// run without a cached token.json, gmail.NewService blocks here printing
// an auth URL and reading the exchange code from stdin.
func initGmail() *gmailv1.Service {
	if _, err := os.Stat("credentials.json"); err != nil {
		log.Println("credentials.json not found — /inbox endpoints will be unavailable")
		return nil
	}
	svc, err := gmail.NewService(context.Background())
	if err != nil {
		log.Printf("Gmail client init failed: %v — /inbox endpoints will be unavailable", err)
		return nil
	}
	log.Println("Gmail client ready")
	return svc
}

// runInboxSyncLoop runs one sync shortly after startup, then on a fixed
// interval. Cleanly stopping this on shutdown is not required for v1.
func runInboxSyncLoop(syncer *syncpkg.Syncer) {
	time.Sleep(10 * time.Second)
	syncOnce(syncer)

	ticker := time.NewTicker(30 * time.Minute)
	for range ticker.C {
		syncOnce(syncer)
	}
}

func syncOnce(syncer *syncpkg.Syncer) {
	res, err := syncer.Run(context.Background(), "")
	if err != nil {
		log.Printf("inbox sync failed: %v", err)
		return
	}
	log.Printf("inbox sync: fetched %d message(s), %d new event(s)", res.Fetched, res.NewEvents)
}
