package ai

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

var (
	ErrInvalidURL     = errors.New("invalid url")
	ErrFetchFailed    = errors.New("could not fetch url")
	ErrPageUnreadable = errors.New("page has no readable job text")
)

// FetchJobText downloads a public page and extracts its main text content.
func FetchJobText(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", ErrInvalidURL
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", ErrInvalidURL
	}
	// Bazi siteler UA'siz istekleri reddeder.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; jobtracker-selfhosted)")

	resp, err := client.Do(req)
	if err != nil {
		return "", ErrFetchFailed
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", ErrFetchFailed
	}

	article, err := readability.FromReader(resp.Body, u)
	if err != nil {
		return "", ErrPageUnreadable
	}
	text := strings.TrimSpace(article.TextContent)
	// Sezgisel esik: login duvari / JS-only sayfalar cok az metin dondurur.
	if len(text) < 200 {
		return "", ErrPageUnreadable
	}
	return text, nil
}
