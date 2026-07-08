package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"google.golang.org/genai"

	"github.com/coderfeye13/jobtracker/internal/gen"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// ErrUnparseable is returned when Gemini cannot extract job details from the text.
var ErrUnparseable = errors.New("could not parse as job posting")

type Client struct {
	inner *genai.Client
}

func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return &Client{inner: c}, nil
}

type parsedJob struct {
	Company        string   `json:"company"`
	Position       string   `json:"position"`
	City           *string  `json:"city,omitempty"`
	Source         *string  `json:"source,omitempty"`
	EmploymentType *string  `json:"employment_type,omitempty"`
	SalaryMin      *float64 `json:"salary_min,omitempty"`
	SalaryMax      *float64 `json:"salary_max,omitempty"`
	SalaryPeriod   *string  `json:"salary_period,omitempty"`
	AppliedAt      *string  `json:"applied_at,omitempty"`
	JobDescription *string  `json:"job_description,omitempty"`
}

var jobSchema = &genai.Schema{
	Type:     genai.TypeObject,
	Required: []string{"company", "position"},
	Properties: map[string]*genai.Schema{
		"company":  {Type: genai.TypeString, Description: "Company name"},
		"position": {Type: genai.TypeString, Description: "Job title / position"},
		"city":     {Type: genai.TypeString, Description: "City or location of the role"},
		"source": {
			Type:        genai.TypeString,
			Description: "Platform where the job was found",
			Enum:        []string{"linkedin", "indeed", "stepstone", "referral", "company_site", "other"},
		},
		"employment_type": {
			Type: genai.TypeString,
			Enum: []string{"werkstudent", "fulltime", "parttime", "internship"},
		},
		"salary_min":    {Type: genai.TypeNumber, Description: "Lower bound of salary range (numeric only)"},
		"salary_max":    {Type: genai.TypeNumber, Description: "Upper bound of salary range (numeric only)"},
		"salary_period": {Type: genai.TypeString, Enum: []string{"hourly", "monthly", "yearly"}},
		"applied_at":    {Type: genai.TypeString, Description: "Application deadline or post date in YYYY-MM-DD format"},
		"job_description": {
			Type:        genai.TypeString,
			Description: "The full raw job posting text, verbatim",
		},
	},
}

var systemInstruction = &genai.Content{
	Parts: []*genai.Part{{Text: `You are a job application assistant. Extract structured information from the raw job posting text provided.
If a field cannot be determined from the text, omit it.
For salary extract numeric values only (no currency symbols).
For employment_type: use "werkstudent" for student/working-student jobs, "fulltime", "parttime", or "internship" as appropriate.
Always copy the full posting verbatim into job_description.`}},
}

func (c *Client) ParseJob(ctx context.Context, rawText string, sourceURL *string) (*gen.ApplicationInput, error) {
	prompt := "Extract job information from this posting:\n\n" + rawText
	if sourceURL != nil {
		prompt = "Source URL: " + *sourceURL + "\n\n" + prompt
	}

	result, err := c.inner.Models.GenerateContent(
		ctx,
		"gemini-2.0-flash",
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			SystemInstruction: systemInstruction,
			ResponseMIMEType:  "application/json",
			ResponseSchema:    jobSchema,
			Temperature:       genai.Ptr[float32](0.1),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini: %w", err)
	}

	if len(result.Candidates) == 0 ||
		result.Candidates[0].Content == nil ||
		len(result.Candidates[0].Content.Parts) == 0 {
		return nil, ErrUnparseable
	}

	raw := result.Candidates[0].Content.Parts[0].Text
	var parsed parsedJob
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, ErrUnparseable
	}
	if parsed.Company == "" || parsed.Position == "" {
		return nil, ErrUnparseable
	}

	return toInput(parsed, rawText), nil
}

func toInput(p parsedJob, rawText string) *gen.ApplicationInput {
	input := gen.ApplicationInput{
		Company:        p.Company,
		Position:       p.Position,
		City:           p.City,
		SalaryMin:      p.SalaryMin,
		SalaryMax:      p.SalaryMax,
		JobDescription: &rawText,
	}
	if p.Source != nil {
		v := gen.ApplicationInputSource(*p.Source)
		input.Source = &v
	}
	if p.EmploymentType != nil {
		v := gen.EmploymentType(*p.EmploymentType)
		input.EmploymentType = &v
	}
	if p.SalaryPeriod != nil {
		v := gen.SalaryPeriod(*p.SalaryPeriod)
		input.SalaryPeriod = &v
	}
	if p.AppliedAt != nil {
		if t, err := time.Parse("2006-01-02", *p.AppliedAt); err == nil {
			input.AppliedAt = &openapi_types.Date{Time: t}
		}
	}
	return &input
}
