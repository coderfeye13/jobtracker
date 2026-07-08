package store

import (
	"errors"
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
	if err := db.AutoMigrate(&Application{}, &Profile{}); err != nil {
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
