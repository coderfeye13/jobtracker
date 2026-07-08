package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/coderfeye13/jobtracker/internal/ai"
	"github.com/coderfeye13/jobtracker/internal/gen"
	"github.com/coderfeye13/jobtracker/internal/store"
)

type Server struct {
	store *store.Store
	ai    *ai.Client // nil if GEMINI_API_KEY not set
}

func NewServer(st *store.Store, aiClient *ai.Client) *Server {
	return &Server{store: st, ai: aiClient}
}

// ---------------------------------------------------------------------------
// Applications — Phase 1
// ---------------------------------------------------------------------------

func (s *Server) ListApplications(ctx echo.Context, params gen.ListApplicationsParams) error {
	var statusStr *string
	if params.Status != nil {
		v := string(*params.Status)
		statusStr = &v
	}
	apps, err := s.store.List(statusStr)
	if err != nil {
		return err
	}
	out := make([]gen.Application, len(apps))
	for i, a := range apps {
		out[i] = toGen(a)
	}
	return ctx.JSON(http.StatusOK, out)
}

func (s *Server) CreateApplication(ctx echo.Context) error {
	var input gen.ApplicationInput
	if err := ctx.Bind(&input); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{Message: err.Error()})
	}
	if strings.TrimSpace(input.Company) == "" || strings.TrimSpace(input.Position) == "" {
		return ctx.JSON(http.StatusBadRequest, gen.Error{Message: "company and position are required"})
	}
	app := fromInput(input)
	if err := s.store.Create(&app); err != nil {
		return err
	}
	return ctx.JSON(http.StatusCreated, toGen(app))
}

func (s *Server) GetApplication(ctx echo.Context, id int64) error {
	app, err := s.store.Get(id)
	if errors.Is(err, store.ErrNotFound) {
		return ctx.JSON(http.StatusNotFound, gen.Error{Message: "application not found"})
	}
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, toGen(*app))
}

func (s *Server) UpdateApplication(ctx echo.Context, id int64) error {
	app, err := s.store.Get(id)
	if errors.Is(err, store.ErrNotFound) {
		return ctx.JSON(http.StatusNotFound, gen.Error{Message: "application not found"})
	}
	if err != nil {
		return err
	}
	var upd gen.ApplicationUpdate
	if err := ctx.Bind(&upd); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{Message: err.Error()})
	}
	applyUpdate(app, upd)
	if err := s.store.Save(app); err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, toGen(*app))
}

func (s *Server) DeleteApplication(ctx echo.Context, id int64) error {
	err := s.store.Delete(id)
	if errors.Is(err, store.ErrNotFound) {
		return ctx.JSON(http.StatusNotFound, gen.Error{Message: "application not found"})
	}
	if err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Profile — Phase 2
// ---------------------------------------------------------------------------

func (s *Server) GetProfile(ctx echo.Context) error {
	p, err := s.store.GetProfile()
	if errors.Is(err, store.ErrNotFound) {
		return ctx.JSON(http.StatusNotFound, gen.Error{Message: "no CV uploaded yet — save one via PUT /profile"})
	}
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, toGenProfile(*p))
}

func (s *Server) UpdateProfile(ctx echo.Context) error {
	var in gen.Profile
	if err := ctx.Bind(&in); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{Message: err.Error()})
	}
	if strings.TrimSpace(in.CvText) == "" {
		return ctx.JSON(http.StatusBadRequest, gen.Error{Message: "cv_text must not be empty"})
	}
	p, err := s.store.SaveProfile(in.CvText)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, toGenProfile(*p))
}

// ---------------------------------------------------------------------------
// AI — Phase 1 + Phase 2
// ---------------------------------------------------------------------------

func (s *Server) ParseJobPosting(ctx echo.Context) error {
	if s.ai == nil {
		return ctx.JSON(http.StatusServiceUnavailable, gen.Error{Message: "AI not configured: set GEMINI_API_KEY"})
	}
	var body gen.ParseJobPostingJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{Message: err.Error()})
	}
	input, err := s.ai.ParseJob(ctx.Request().Context(), body.RawText, body.Url)
	if errors.Is(err, ai.ErrUnparseable) {
		return ctx.JSON(http.StatusUnprocessableEntity, gen.Error{Message: "text could not be parsed as a job posting"})
	}
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, input)
}

// loadScoringInputs centralizes the shared preconditions of score and
// cover-letter: application exists, CV exists, job_description present.
// Returning (nil, nil, nil) means an HTTP error response was already written.
func (s *Server) loadScoringInputs(ctx echo.Context, applicationID int64) (*store.Application, *store.Profile, error) {
	app, err := s.store.Get(applicationID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, ctx.JSON(http.StatusNotFound, gen.Error{Message: "application not found"})
	}
	if err != nil {
		return nil, nil, err
	}
	prof, err := s.store.GetProfile()
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, ctx.JSON(http.StatusBadRequest, gen.Error{Message: "no CV uploaded yet — save one via PUT /profile first"})
	}
	if err != nil {
		return nil, nil, err
	}
	if app.JobDescription == nil || strings.TrimSpace(*app.JobDescription) == "" {
		return nil, nil, ctx.JSON(http.StatusBadRequest, gen.Error{Message: "application has no job_description to score against"})
	}
	return app, prof, nil
}

func (s *Server) ScoreApplication(ctx echo.Context) error {
	if s.ai == nil {
		return ctx.JSON(http.StatusServiceUnavailable, gen.Error{Message: "AI not configured: set GEMINI_API_KEY"})
	}
	var req gen.ScoreRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{Message: err.Error()})
	}

	app, prof, errResp := s.loadScoringInputs(ctx, req.ApplicationId)
	if app == nil {
		return errResp // error response already written (or a real error)
	}

	res, err := s.ai.ScoreCV(ctx.Request().Context(), prof.CVText, *app.JobDescription)
	if err != nil {
		return err
	}

	// Persist the result on the application so the board can show the badge.
	details, err := json.Marshal(res)
	if err != nil {
		return err
	}
	score := res.Score
	detailsStr := string(details)
	app.FitScore = &score
	app.ScoreDetails = &detailsStr
	if err := s.store.Save(app); err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, gen.ScoreResponse{
		Score:           res.Score,
		MatchedKeywords: res.MatchedKeywords,
		MissingKeywords: res.MissingKeywords,
		Suggestions:     res.Suggestions,
	})
}

func (s *Server) GenerateCoverLetter(ctx echo.Context) error {
	if s.ai == nil {
		return ctx.JSON(http.StatusServiceUnavailable, gen.Error{Message: "AI not configured: set GEMINI_API_KEY"})
	}
	var req gen.CoverLetterRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.Error{Message: err.Error()})
	}

	app, prof, errResp := s.loadScoringInputs(ctx, req.ApplicationId)
	if app == nil {
		return errResp
	}

	letter, err := s.ai.GenerateCoverLetter(
		ctx.Request().Context(),
		prof.CVText,
		*app.JobDescription,
		app.Company,
		app.Position,
		string(req.Language),
		string(req.Tone),
	)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, gen.CoverLetterResponse{CoverLetter: letter})
}
