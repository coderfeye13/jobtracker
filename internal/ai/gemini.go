package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"google.golang.org/genai"

	"github.com/coderfeye13/jobtracker/internal/gen"
)

// ErrUnparseable is returned when Gemini cannot extract job details from the text.
var ErrUnparseable = errors.New("could not parse as job posting")

// errEmptyResponse: Gemini returned no usable candidate (e.g. safety filter).
var errEmptyResponse = errors.New("gemini: empty response")

type Client struct {
	inner *genai.Client
	model string
}

func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	// Configuration is resolved once at startup.
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}

	return &Client{inner: c, model: model}, nil
}

// firstText safely extracts the first text part of the first candidate.
// The response is three levels deep and every level can be empty
// (e.g. safety filters), so we guard all of them before dereferencing.
func firstText(result *genai.GenerateContentResponse) (string, error) {
	if result == nil ||
		len(result.Candidates) == 0 ||
		result.Candidates[0].Content == nil ||
		len(result.Candidates[0].Content.Parts) == 0 {
		return "", errEmptyResponse
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}

// ---------------------------------------------------------------------------
// ParseJob — Phase 1
// ---------------------------------------------------------------------------

type parsedJob struct {
	Company        string   `json:"company"`
	Position       string   `json:"position"`
	City           *string  `json:"city,omitempty"`
	Source         *string  `json:"source,omitempty"`
	EmploymentType *string  `json:"employment_type,omitempty"`
	SalaryMin      *float64 `json:"salary_min,omitempty"`
	SalaryMax      *float64 `json:"salary_max,omitempty"`
	SalaryPeriod   *string  `json:"salary_period,omitempty"`
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
	},
}

var systemInstruction = &genai.Content{
	Parts: []*genai.Part{{Text: `You are a job application assistant. Extract structured information from the raw job posting text provided.
If a field cannot be determined from the text, omit it.
For salary extract numeric values only (no currency symbols).
For employment_type: use "werkstudent" for student/working-student jobs, "fulltime", "parttime", or "internship" as appropriate.
Strip gender markers like (m/w/d), (f/m/x), (all gender) or ":in" suffixes from the position title.`}},
}

func (c *Client) ParseJob(ctx context.Context, rawText string, sourceURL *string) (*gen.ApplicationInput, error) {
	prompt := "Extract job information from this posting:\n\n" + rawText
	if sourceURL != nil {
		prompt = "Source URL: " + *sourceURL + "\n\n" + prompt
	}

	result, err := c.inner.Models.GenerateContent(
		ctx,
		c.model,
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

	raw, err := firstText(result)
	if err != nil {
		return nil, ErrUnparseable
	}

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
	return &input
}

// ---------------------------------------------------------------------------
// ScoreCV — Phase 2
// ---------------------------------------------------------------------------

// ScoreResult is the AI layer's own type (same reasoning as parsedJob:
// validate at the boundary, stay decoupled from gen).
type ScoreResult struct {
	Score           int      `json:"score"`
	MatchedKeywords []string `json:"matched_keywords"`
	MissingKeywords []string `json:"missing_keywords"`
	Suggestions     []string `json:"suggestions"`
}

var scoreSchema = &genai.Schema{
	Type:     genai.TypeObject,
	Required: []string{"score", "matched_keywords", "missing_keywords", "suggestions"},
	Properties: map[string]*genai.Schema{
		"score": {
			Type:        genai.TypeInteger,
			Description: "Overall fit score from 0 (no match) to 100 (perfect match)",
		},
		"matched_keywords": {
			Type:        genai.TypeArray,
			Items:       &genai.Schema{Type: genai.TypeString},
			Description: "Skills/technologies required by the posting AND present in the CV",
		},
		"missing_keywords": {
			Type:        genai.TypeArray,
			Items:       &genai.Schema{Type: genai.TypeString},
			Description: "Skills/technologies required by the posting but NOT found in the CV",
		},
		"suggestions": {
			Type:        genai.TypeArray,
			Items:       &genai.Schema{Type: genai.TypeString},
			Description: "3-5 concrete, actionable suggestions to improve the CV for THIS posting",
		},
	},
}

var scoreInstruction = &genai.Content{
	Parts: []*genai.Part{{Text: `You are an experienced technical recruiter evaluating how well a candidate's CV matches a specific job posting.
Score honestly: 80+ only for strong matches, 50-79 for partial matches, below 50 for weak matches.
Base keywords strictly on the posting's stated requirements. Only list a keyword as matched if there is clear evidence in the CV.
Do not invent skills that are not in the CV. Language requirements (e.g. German level) count as keywords too.
Keep each suggestion short, specific and actionable.`}},
}

func (c *Client) ScoreCV(ctx context.Context, cvText, jobDescription string) (*ScoreResult, error) {
	prompt := "JOB POSTING:\n" + jobDescription + "\n\nCANDIDATE CV:\n" + cvText

	result, err := c.inner.Models.GenerateContent(
		ctx,
		c.model,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			SystemInstruction: scoreInstruction,
			ResponseMIMEType:  "application/json",
			ResponseSchema:    scoreSchema,
			Temperature:       genai.Ptr[float32](0.1), // evaluation = determinism task
		},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini: %w", err)
	}

	raw, err := firstText(result)
	if err != nil {
		return nil, err
	}

	var res ScoreResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, fmt.Errorf("gemini: invalid score payload: %w", err)
	}
	// Boundary validation: never trust the model to stay in range.
	if res.Score < 0 {
		res.Score = 0
	}
	if res.Score > 100 {
		res.Score = 100
	}
	return &res, nil
}

// ---------------------------------------------------------------------------
// GenerateCoverLetter — Phase 2
// ---------------------------------------------------------------------------

var coverLetterInstruction = &genai.Content{
	Parts: []*genai.Part{{Text: `You write tailored cover letters for job applications.
Rules:
- Write in the requested LANGUAGE. For German use proper business-letter conventions (Anrede such as "Sehr geehrte Damen und Herren," and Grussformel "Mit freundlichen Gruessen" with the candidate's name from the CV).
- Match the requested TONE: formal = conservative business style; warm = personable but professional; concise = short and direct.
- Ground every claim in the CV. Never invent experience, skills, degrees or numbers that are not in the CV.
- Reference 2-3 specific requirements from the posting and connect them to concrete evidence from the CV.
- Length: 250-350 words. Output ONLY the letter body text: no subject line, no addresses, no markdown, no placeholders like [Company].`}},
}

func (c *Client) GenerateCoverLetter(ctx context.Context, cvText, jobDescription, company, position, language, tone string) (string, error) {
	prompt := fmt.Sprintf(
		"COMPANY: %s\nPOSITION: %s\nLANGUAGE: %s\nTONE: %s\n\nJOB POSTING:\n%s\n\nCANDIDATE CV:\n%s",
		company, position, language, tone, jobDescription, cvText,
	)

	result, err := c.inner.Models.GenerateContent(
		ctx,
		c.model,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			SystemInstruction: coverLetterInstruction,
			// Writing task, not extraction: some creativity is desirable.
			Temperature: genai.Ptr[float32](0.7),
		},
	)
	if err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}

	return firstText(result)
}
