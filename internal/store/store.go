package store

import (
	"errors"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ErrNotFound is returned when a record does not exist.
// Handler katmani GORM detayini bilmesin diye kendi hatamiza ceviriyoruz.
var ErrNotFound = errors.New("not found")

type Application struct {
	ID             int64 `gorm:"primaryKey"`
	Company        string
	Position       string
	City           *string
	Source         *string
	URL            *string
	EmploymentType *string
	SalaryMin      *float64
	SalaryMax      *float64
	SalaryPeriod   *string
	Status         string `gorm:"default:saved;index"`
	AppliedAt      *time.Time
	Notes          *string
	JobDescription *string
	// Phase 2: set by POST /ai/score, never by the client directly.
	FitScore     *int
	ScoreDetails *string // JSON string: matched/missing keywords + suggestions
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Profile holds the user's CV as raw text. Single-user app -> singleton row
// pattern: there is at most one Profile and its ID is always 1.
type Profile struct {
	ID        int64 `gorm:"primaryKey"`
	CVText    string
	UpdatedAt time.Time
}

// InboxEvent is one Gmail message the classifier has looked at. Kind
// "irrelevant" events are stored (dismissed) too, so re-syncing never
// re-classifies the same message (see HasInboxEvent).
type InboxEvent struct {
	ID             int64  `gorm:"primaryKey"`
	GmailMessageID string `gorm:"uniqueIndex"`
	ReceivedAt     time.Time
	From           string
	Subject        string
	Kind           string
	Summary        string
	ApplicationID  *int64
	// SuggestedStatus/Confidence are only set for kind=application_update.
	SuggestedStatus *string
	Confidence      *float64
	Dismissed       bool
	CreatedAt       time.Time
}

type Store struct {
	db *gorm.DB
}

func New(path string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// AutoMigrate adds the two new Application columns and creates the
	// profiles table on first run after this change. Existing data is kept.
	if err := db.AutoMigrate(&Application{}, &Profile{}, &InboxEvent{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) List(status *string) ([]Application, error) {
	var apps []Application
	q := s.db.Order("updated_at DESC")
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	return apps, q.Find(&apps).Error
}

func (s *Store) Get(id int64) (*Application, error) {
	var app Application
	err := s.db.First(&app, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *Store) Create(app *Application) error {
	return s.db.Create(app).Error
}

// Save writes the full record back (load-modify-save pattern).
func (s *Store) Save(app *Application) error {
	return s.db.Save(app).Error
}

func (s *Store) Delete(id int64) error {
	res := s.db.Delete(&Application{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// GetProfile returns the singleton profile row, or ErrNotFound if the user
// has not uploaded a CV yet.
func (s *Store) GetProfile() (*Profile, error) {
	var p Profile
	err := s.db.First(&p, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SaveProfile upserts the singleton row: because ID is fixed to 1,
// gorm's Save updates the row if it exists and inserts it otherwise.
func (s *Store) SaveProfile(cvText string) (*Profile, error) {
	p := Profile{ID: 1, CVText: cvText}
	if err := s.db.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// ---------------------------------------------------------------------------
// InboxEvent — Phase 3
// ---------------------------------------------------------------------------

// CreateInboxEvent inserts a new event. Two sync runs can race on the same
// Gmail message (HasInboxEvent is a best-effort pre-check, not a lock), so
// a unique-constraint violation on gmail_message_id is swallowed: the event
// already exists, which is exactly what we wanted.
func (s *Store) CreateInboxEvent(e *InboxEvent) error {
	err := s.db.Create(e).Error
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return nil
	}
	return err
}

// ListInboxEvents returns events newest-received first, optionally
// filtered by kind and excluding dismissed events unless requested.
func (s *Store) ListInboxEvents(kind *string, includeDismissed bool) ([]InboxEvent, error) {
	var events []InboxEvent
	q := s.db.Order("received_at DESC")
	if kind != nil {
		q = q.Where("kind = ?", *kind)
	}
	if !includeDismissed {
		q = q.Where("dismissed = ?", false)
	}
	return events, q.Find(&events).Error
}

func (s *Store) GetInboxEvent(id int64) (*InboxEvent, error) {
	var e InboxEvent
	err := s.db.First(&e, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// DismissInboxEvent hides an event from the default list.
func (s *Store) DismissInboxEvent(id int64) error {
	res := s.db.Model(&InboxEvent{}).Where("id = ?", id).Update("dismissed", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// HasInboxEvent is a cheap dedupe check so sync doesn't spend a Gemini
// call re-classifying a message it has already seen.
func (s *Store) HasInboxEvent(gmailMessageID string) (bool, error) {
	var count int64
	err := s.db.Model(&InboxEvent{}).Where("gmail_message_id = ?", gmailMessageID).Count(&count).Error
	return count > 0, err
}
