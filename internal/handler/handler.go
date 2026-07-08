package handler

import (
	"errors"
	"net/http"

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
