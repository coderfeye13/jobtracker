// Package sync fetches recent Gmail messages, classifies each with Gemini,
// and stores the result as an InboxEvent. It is shared by the
// POST /inbox/sync endpoint and the periodic background loop in
// cmd/server/main.go, so a single Syncer instance (and its mutex) must be
// used by both — that's what keeps the two from ever running concurrently.
package sync

import (
	"context"
	"errors"
	"log"
	syncpkg "sync"

	gmailv1 "google.golang.org/api/gmail/v1"

	"github.com/coderfeye13/jobtracker/internal/ai"
	"github.com/coderfeye13/jobtracker/internal/gmail"
	"github.com/coderfeye13/jobtracker/internal/store"
)

// ErrNotConfigured means the Gmail or AI client is missing, so Run cannot
// do anything. Callers that already gate on nil clients shouldn't normally
// see this — it's a last-line safety net for the background loop.
var ErrNotConfigured = errors.New("sync: gmail or ai client not configured")

// openStatuses are the "still active" application statuses worth matching
// inbox mail against; terminal states (rejected/ghosted/offer-closed-out)
// are excluded.
var openStatuses = map[string]bool{
	"saved":     true,
	"applied":   true,
	"interview": true,
}

// Syncer runs one sync pass: fetch -> classify -> persist. Safe for
// concurrent use — TryLock ensures overlapping calls to Run are no-ops
// rather than racing on the same Gmail messages.
type Syncer struct {
	store *store.Store
	gmail *gmailv1.Service // nil if credentials.json is missing
	ai    *ai.Client       // nil if GEMINI_API_KEY is unset

	mu syncpkg.Mutex
}

func New(st *store.Store, gmailSvc *gmailv1.Service, aiClient *ai.Client) *Syncer {
	return &Syncer{store: st, gmail: gmailSvc, ai: aiClient}
}

// Result is the outcome of one sync run.
type Result struct {
	Fetched   int
	NewEvents int
}

// Run fetches recent mail (via query, or the package defaults when empty),
// classifies any message not already seen, and persists new InboxEvents.
// If another Run is already in flight, this call returns immediately with
// a zero Result instead of blocking or running concurrently.
func (sy *Syncer) Run(ctx context.Context, query string) (Result, error) {
	if !sy.mu.TryLock() {
		log.Println("inbox sync: a run is already in progress, skipping")
		return Result{}, nil
	}
	defer sy.mu.Unlock()

	if sy.gmail == nil || sy.ai == nil {
		return Result{}, ErrNotConfigured
	}

	messages, err := gmail.FetchRecent(ctx, sy.gmail, query)
	if err != nil {
		return Result{}, err
	}

	candidates, err := sy.openApplications()
	if err != nil {
		return Result{}, err
	}
	cvText, err := sy.cvText()
	if err != nil {
		return Result{}, err
	}

	result := Result{Fetched: len(messages)}
	for _, m := range messages {
		seen, err := sy.store.HasInboxEvent(m.ID)
		if err != nil {
			log.Printf("inbox sync: dedupe check failed for message %s: %v", m.ID, err)
			continue
		}
		if seen {
			continue
		}

		cls, err := sy.ai.ClassifyEmail(ctx, m.From, m.Subject, m.Body, candidates, cvText)
		if err != nil {
			log.Printf("inbox sync: classification failed for message %s: %v", m.ID, err)
			continue
		}

		event := &store.InboxEvent{
			GmailMessageID: m.ID,
			ReceivedAt:     m.ReceivedAt,
			From:           m.From,
			Subject:        m.Subject,
			Kind:           cls.Kind,
			Summary:        cls.Summary,
			// Nothing to act on for an irrelevant message — keep it out of
			// the default inbox list from the moment it's created.
			Dismissed: cls.Kind == "irrelevant",
		}
		if cls.ApplicationID != 0 {
			id := cls.ApplicationID
			event.ApplicationID = &id
		}
		if cls.SuggestedStatus != "" {
			status := cls.SuggestedStatus
			event.SuggestedStatus = &status
		}
		confidence := cls.Confidence
		event.Confidence = &confidence

		if err := sy.store.CreateInboxEvent(event); err != nil {
			log.Printf("inbox sync: storing event for message %s: %v", m.ID, err)
			continue
		}
		result.NewEvents++
	}

	return result, nil
}

func (sy *Syncer) openApplications() ([]ai.ApplicationSummary, error) {
	apps, err := sy.store.List(nil)
	if err != nil {
		return nil, err
	}
	out := make([]ai.ApplicationSummary, 0, len(apps))
	for _, a := range apps {
		if !openStatuses[a.Status] {
			continue
		}
		out = append(out, ai.ApplicationSummary{
			ID:       a.ID,
			Company:  a.Company,
			Position: a.Position,
			Status:   a.Status,
		})
	}
	return out, nil
}

// cvText returns the stored CV, or "" if none has been uploaded yet —
// classification still works without one, just with less context.
func (sy *Syncer) cvText() (string, error) {
	prof, err := sy.store.GetProfile()
	if errors.Is(err, store.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return prof.CVText, nil
}
