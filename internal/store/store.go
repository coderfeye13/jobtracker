package store

import (
	"errors"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ErrNotFound is returned when a record does not exist.
// Handler katmanı GORM detayını bilmesin diye kendi hatamıza çeviriyoruz.
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
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Store struct {
	db *gorm.DB
}

func New(path string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Application{}); err != nil {
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

// Save writes the full record back (load-modify-save pattern for PATCH).
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
