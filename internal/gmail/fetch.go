package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

// defaultQuery mirrors a Gmail search box query: recent, not promotions/social.
const defaultQuery = "newer_than:3d category:(primary OR updates)"

// Message is the subset of a Gmail message the classifier needs.
type Message struct {
	ID         string
	From       string
	Subject    string
	ReceivedAt time.Time
	Body       string // best-effort plain text
}

// FetchRecent lists messages matching query and returns their headers plus
// a plain-text body. An empty query falls back to GMAIL_QUERY, then to
// defaultQuery.
func FetchRecent(ctx context.Context, svc *gmail.Service, query string) ([]Message, error) {
	if query == "" {
		query = os.Getenv("GMAIL_QUERY")
	}
	if query == "" {
		query = defaultQuery
	}

	list, err := svc.Users.Messages.List("me").Q(query).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("gmail: listing messages: %w", err)
	}

	messages := make([]Message, 0, len(list.Messages))
	for _, ref := range list.Messages {
		full, err := svc.Users.Messages.Get("me", ref.Id).Format("full").Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("gmail: fetching message %s: %w", ref.Id, err)
		}
		messages = append(messages, toMessage(full))
	}
	return messages, nil
}

func toMessage(m *gmail.Message) Message {
	msg := Message{ID: m.Id}
	if m.Payload != nil {
		for _, h := range m.Payload.Headers {
			switch h.Name {
			case "From":
				msg.From = h.Value
			case "Subject":
				msg.Subject = h.Value
			}
		}
		msg.Body = extractBody(m.Payload)
	}
	if m.InternalDate != 0 {
		msg.ReceivedAt = time.UnixMilli(m.InternalDate)
	}
	return msg
}

// extractBody walks the MIME part tree for a text/plain part; if none
// exists it falls back to text/html with tags stripped.
func extractBody(part *gmail.MessagePart) string {
	if data := findPart(part, "text/plain"); data != "" {
		return decodeBody(data)
	}
	if data := findPart(part, "text/html"); data != "" {
		return stripHTML(decodeBody(data))
	}
	return ""
}

// findPart returns the raw (still base64url-encoded) body data of the
// first part matching mimeType, depth-first.
func findPart(part *gmail.MessagePart, mimeType string) string {
	if part == nil {
		return ""
	}
	if part.MimeType == mimeType && part.Body != nil && part.Body.Data != "" {
		return part.Body.Data
	}
	for _, p := range part.Parts {
		if data := findPart(p, mimeType); data != "" {
			return data
		}
	}
	return ""
}

func decodeBody(data string) string {
	raw, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		// Gmail's base64url data is unpadded more often than not.
		raw, err = base64.RawURLEncoding.DecodeString(data)
		if err != nil {
			return ""
		}
	}
	return string(raw)
}

var (
	scriptStyleRe = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	anyTagRe      = regexp.MustCompile(`<[^>]*>`)
	whitespaceR   = regexp.MustCompile(`\s+`)
)

// stripHTML removes markup and collapses whitespace, giving a rough
// plain-text fallback for HTML-only emails.
func stripHTML(html string) string {
	text := scriptStyleRe.ReplaceAllString(html, " ")
	text = anyTagRe.ReplaceAllString(text, " ")
	text = whitespaceR.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
