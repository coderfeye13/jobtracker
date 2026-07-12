package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

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

// ---------------------------------------------------------------------------
// ClassifyEmail — Phase 3
// ---------------------------------------------------------------------------

// ApplicationSummary is the minimal view of a tracked application the
// classifier needs to decide whether an email is about it.
type ApplicationSummary struct {
	ID       int64
	Company  string
	Position string
	Status   string
}

// EmailClassification is the AI layer's own type (validated at the
// boundary, same reasoning as ScoreResult).
type EmailClassification struct {
	Kind            string  `json:"kind"`
	Summary         string  `json:"summary"`
	ApplicationID   int64   `json:"application_id"`
	SuggestedStatus string  `json:"suggested_status"`
	Confidence      float64 `json:"confidence"`
}

var classifySchema = &genai.Schema{
	Type:     genai.TypeObject,
	Required: []string{"kind", "summary", "application_id", "suggested_status", "confidence"},
	Properties: map[string]*genai.Schema{
		"kind": {
			Type: genai.TypeString,
			Enum: []string{"job_alert", "application_update", "irrelevant"},
		},
		"summary": {
			Type:        genai.TypeString,
			Description: "1-3 sentences. For job_alert: which position(s) in the mail match the CV profile and why; mention city if present. For application_update: what changed.",
		},
		"application_id": {
			Type:        genai.TypeInteger,
			Description: "ID of the candidate application this email is about. Must be one of the given candidate IDs, or 0 if none / not an application_update.",
		},
		"suggested_status": {
			Type:        genai.TypeString,
			Description: "New status to suggest for the matched application. Empty string if not applicable.",
			Enum:        []string{"none", "applied", "interview", "offer", "rejected", "ghosted"},
		},
		"confidence": {
			Type:        genai.TypeNumber,
			Description: "Confidence in this classification, 0 (guess) to 1 (certain)",
		},
	},
}

var classifyInstruction = &genai.Content{
	Parts: []*genai.Part{{Text: `You are an email triage assistant for a job search tracker. Classify the email and, if relevant, connect it to one of the candidate's tracked applications.

KINDS:
- job_alert: a job board or company sent one or more open positions. Set summary to which position(s) in the mail are a good match for the candidate's CV profile and why; mention city if the mail states one. If none of the listed positions are a reasonable match, use "irrelevant" instead.
- application_update: the email reports on the status of ONE SPECIFIC application the candidate already submitted (e.g. rejection, interview/assessment invite, offer). Only use this kind if the company or position clearly appears in the email AND matches one of the candidate applications listed in the prompt — never guess application_id otherwise.
- irrelevant: marketing, newsletters, unrelated account/notification emails, or anything not about the candidate's own job search.

STATUS RULES (application_update only):
- Clear rejection language -> suggested_status "rejected".
- Invitation to interview, phone screen, technical assessment, or next round -> "interview".
- Job offer extended -> "offer".
- If the update does not clearly map to applied/interview/offer/rejected/ghosted, use suggested_status "none".

Emails about the candidate's OWN applications (confirmations, rejections, interview invitations) are application_update even when they match none of the open applications in the list; set application_id to 0 in that case.

Classify conservatively: when unsure, prefer "irrelevant" over a wrong match, and never match an application unless its company or position is clearly present in the email. application_id must be 0 unless kind is application_update with a clear match.`}},
}

// ClassifyEmail asks Gemini to classify a single email against the
// candidate's currently open (non-terminal) applications. body is
// truncated to a few thousand characters before being sent.
func (c *Client) ClassifyEmail(ctx context.Context, from, subject, body string, candidates []ApplicationSummary, cvSummary string) (*EmailClassification, error) {
	const maxBodyChars = 4000
	if len(body) > maxBodyChars {
		body = body[:maxBodyChars]
	}

	var candidateLines strings.Builder
	if len(candidates) == 0 {
		candidateLines.WriteString("(none — candidate has no open applications)")
	}
	for _, a := range candidates {
		fmt.Fprintf(&candidateLines, "- id=%d company=%q position=%q status=%s\n", a.ID, a.Company, a.Position, a.Status)
	}

	prompt := fmt.Sprintf(
		"FROM: %s\nSUBJECT: %s\n\nBODY:\n%s\n\nCANDIDATE'S OPEN APPLICATIONS:\n%s\nCANDIDATE'S CV:\n%s",
		from, subject, body, candidateLines.String(), cvSummary,
	)

	result, err := c.inner.Models.GenerateContent(
		ctx,
		c.model,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			SystemInstruction: classifyInstruction,
			ResponseMIMEType:  "application/json",
			ResponseSchema:    classifySchema,
			Temperature:       genai.Ptr[float32](0.1),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini: %w", err)
	}

	raw, err := firstText(result)
	if err != nil {
		return nil, err
	}

	var res EmailClassification
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, fmt.Errorf("gemini: invalid classification payload: %w", err)
	}

	// Boundary validation: never trust the model to stay in range or
	// respect the candidate list.

	// suggested_status: whitelist. "none" is a schema-only sentinel (empty
	// enum values are rejected by Gemini); anything outside the real
	// statuses collapses to "" = no suggestion.
	switch res.SuggestedStatus {
	case "applied", "interview", "offer", "rejected", "ghosted":
		// valid suggestion, keep it
	default:
		res.SuggestedStatus = ""
	}

	switch res.Kind {
	case "job_alert", "application_update", "irrelevant":
	default:
		res.Kind = "irrelevant"
	}
	if res.Confidence < 0 {
		res.Confidence = 0
	}
	if res.Confidence > 1 {
		res.Confidence = 1
	}
	if res.Kind != "application_update" {
		res.ApplicationID = 0
		res.SuggestedStatus = ""
	} else if !isCandidateID(candidates, res.ApplicationID) {
		res.ApplicationID = 0
		res.SuggestedStatus = ""
	}

	return &res, nil
}

func isCandidateID(candidates []ApplicationSummary, id int64) bool {
	for _, a := range candidates {
		if a.ID == id {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// TailorCV — Phase 2.5
// ---------------------------------------------------------------------------

// TailorResult is the AI layer's own type (same boundary-validation
// reasoning as parsedJob / ScoreResult / EmailClassification).
type TailorResult struct {
	TailoredCV string   `json:"tailored_cv"`
	Changes    []string `json:"changes"`
}

var tailorSchema = &genai.Schema{
	Type:     genai.TypeObject,
	Required: []string{"tailored_cv", "changes"},
	Properties: map[string]*genai.Schema{
		"tailored_cv": {
			Type:        genai.TypeString,
			Description: "The adapted CV as plain text, same overall structure as the original",
		},
		"changes": {
			Type:        genai.TypeArray,
			Items:       &genai.Schema{Type: genai.TypeString},
			Description: "One short line per modification made (reorder, rephrase, cut, terminology alignment) so a human can review every change",
		},
	},
}

// The four guardrails recorded in CLAUDE.md, verbatim as prompt rules.
var tailorInstruction = &genai.Content{
	Parts: []*genai.Part{{Text: `You adapt a candidate's CV to one specific job posting. You are an editor, not an author.

RULES (all mandatory):
1. NO FABRICATION. You may reorder, rephrase, emphasize and cut. You may NEVER add skills, tools, employers, dates, degrees, metrics or responsibilities that are not present in the CV. If a keyword from the posting has no evidence in the CV, do not insert it.
2. TERMINOLOGY ALIGNMENT. If the CV contains evidence for a posting keyword under a different name, align the wording to the posting's term (e.g. CV "GitLab CI/CD pipelines" vs posting "Build-Automatisierung"). This is renaming existing evidence, never inventing new evidence.
3. REORDER FREEDOM. Move the most relevant experience, projects and skills first; shorten or drop items irrelevant to this posting. Keep the CV's section structure (education, experience, skills, projects) recognizable.
4. NATURAL TONE. Plain, factual, human. Keep the CV's existing voice and sentence rhythm. Forbidden: buzzword inflation ("spearheaded", "leveraged", "passionate", "results-driven", "synergy"), superlatives, and any phrasing the original author would not naturally write. A rephrased bullet must stay verifiable in an interview: if the candidate couldn't defend the sentence word-by-word, don't write it. When in doubt, prefer the CV's original wording over a "better-sounding" alternative.

OUTPUT: write the tailored CV in the requested LANGUAGE (translate faithfully if it differs from the CV's language — translation is not fabrication). In "changes", list every modification you made, one short line each, so the candidate can audit you. If you made no change of a given kind, don't pad the list.`}},
}

// TailorCV adapts the stored CV's content to one posting under the four
// guardrails above. Returns the adapted text plus an auditable change log.
func (c *Client) TailorCV(ctx context.Context, cvText, jobDescription, company, position, language string) (*TailorResult, error) {
	prompt := fmt.Sprintf(
		"COMPANY: %s\nPOSITION: %s\nLANGUAGE: %s\n\nJOB POSTING:\n%s\n\nCANDIDATE CV:\n%s",
		company, position, language, jobDescription, cvText,
	)

	result, err := c.inner.Models.GenerateContent(
		ctx,
		c.model,
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			SystemInstruction: tailorInstruction,
			ResponseMIMEType:  "application/json",
			ResponseSchema:    tailorSchema,
			// Between extraction (0.1) and letter-writing (0.7): rephrasing
			// needs some freedom, fabrication resistance needs restraint.
			Temperature: genai.Ptr[float32](0.3),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini: %w", err)
	}

	raw, err := firstText(result)
	if err != nil {
		return nil, err
	}

	var res TailorResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, fmt.Errorf("gemini: invalid tailor payload: %w", err)
	}
	// Boundary validation: an empty CV means the model failed the task.
	if strings.TrimSpace(res.TailoredCV) == "" {
		return nil, fmt.Errorf("gemini: empty tailored CV")
	}
	if res.Changes == nil {
		res.Changes = []string{}
	}
	return &res, nil
}
