package handler

import (
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/coderfeye13/jobtracker/internal/gen"
	"github.com/coderfeye13/jobtracker/internal/store"
)

// toGen: DB modeli -> API tipi
func toGen(a store.Application) gen.Application {
	out := gen.Application{
		Id:             a.ID,
		Company:        a.Company,
		Position:       a.Position,
		City:           a.City,
		Url:            a.URL,
		Notes:          a.Notes,
		JobDescription: a.JobDescription,
		SalaryMin:      a.SalaryMin,
		SalaryMax:      a.SalaryMax,
		FitScore:       a.FitScore,     // Phase 2
		ScoreDetails:   a.ScoreDetails, // Phase 2
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
	status := gen.ApplicationStatus(a.Status)
	out.Status = &status
	if a.Source != nil {
		v := gen.ApplicationSource(*a.Source)
		out.Source = &v
	}
	if a.EmploymentType != nil {
		v := gen.EmploymentType(*a.EmploymentType)
		out.EmploymentType = &v
	}
	if a.SalaryPeriod != nil {
		v := gen.SalaryPeriod(*a.SalaryPeriod)
		out.SalaryPeriod = &v
	}
	if a.AppliedAt != nil {
		out.AppliedAt = &openapi_types.Date{Time: *a.AppliedAt}
	}
	return out
}

// fromInput: API input -> DB modeli (create icin)
func fromInput(in gen.ApplicationInput) store.Application {
	app := store.Application{
		Company:        in.Company,
		Position:       in.Position,
		City:           in.City,
		URL:            in.Url,
		Notes:          in.Notes,
		JobDescription: in.JobDescription,
		SalaryMin:      in.SalaryMin,
		SalaryMax:      in.SalaryMax,
	}
	if in.Status != nil {
		app.Status = string(*in.Status)
	}
	// Status bos kalirsa GORM'daki default:saved devreye girer
	if in.Source != nil {
		v := string(*in.Source)
		app.Source = &v
	}
	if in.EmploymentType != nil {
		v := string(*in.EmploymentType)
		app.EmploymentType = &v
	}
	if in.SalaryPeriod != nil {
		v := string(*in.SalaryPeriod)
		app.SalaryPeriod = &v
	}
	if in.AppliedAt != nil {
		t := in.AppliedAt.Time
		app.AppliedAt = &t
	}
	return app
}

// applyUpdate: PATCH semantigi — nil olan alanlara dokunma.
// FitScore/ScoreDetails bilerek yok: o alanlari sadece /ai/score yazar.
func applyUpdate(app *store.Application, upd gen.ApplicationUpdate) {
	if upd.Company != nil {
		app.Company = *upd.Company
	}
	if upd.Position != nil {
		app.Position = *upd.Position
	}
	if upd.City != nil {
		app.City = upd.City
	}
	if upd.Url != nil {
		app.URL = upd.Url
	}
	if upd.Notes != nil {
		app.Notes = upd.Notes
	}
	if upd.JobDescription != nil {
		app.JobDescription = upd.JobDescription
	}
	if upd.SalaryMin != nil {
		app.SalaryMin = upd.SalaryMin
	}
	if upd.SalaryMax != nil {
		app.SalaryMax = upd.SalaryMax
	}
	if upd.Status != nil {
		app.Status = string(*upd.Status)
	}
	if upd.Source != nil {
		v := string(*upd.Source)
		app.Source = &v
	}
	if upd.EmploymentType != nil {
		v := string(*upd.EmploymentType)
		app.EmploymentType = &v
	}
	if upd.SalaryPeriod != nil {
		v := string(*upd.SalaryPeriod)
		app.SalaryPeriod = &v
	}
	if upd.AppliedAt != nil {
		t := upd.AppliedAt.Time
		app.AppliedAt = &t
	}
}

// toGenProfile: DB Profile -> API Profile
func toGenProfile(p store.Profile) gen.Profile {
	t := p.UpdatedAt
	return gen.Profile{
		CvText:    p.CVText,
		UpdatedAt: &t,
	}
}

// toGenInboxEvent: DB InboxEvent -> API InboxEvent
func toGenInboxEvent(e store.InboxEvent) gen.InboxEvent {
	out := gen.InboxEvent{
		Id:             e.ID,
		GmailMessageId: e.GmailMessageID,
		ReceivedAt:     e.ReceivedAt,
		From:           e.From,
		Subject:        e.Subject,
		Kind:           gen.InboxEventKind(e.Kind),
		Summary:        e.Summary,
		ApplicationId:  e.ApplicationID,
		Confidence:     e.Confidence,
		Dismissed:      e.Dismissed,
	}
	if e.SuggestedStatus != nil {
		v := gen.ApplicationStatus(*e.SuggestedStatus)
		out.SuggestedStatus = &v
	}
	return out
}
