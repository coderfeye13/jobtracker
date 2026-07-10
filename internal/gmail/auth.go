package gmail

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	credentialsFile = "credentials.json"
	tokenFile       = "token.json"
)

// NewService builds a Gmail API client via the OAuth2 desktop-app flow.
// credentials.json (OAuth client ID, "Desktop app" type) must exist in the
// working directory. On first run, with no token.json yet, it prints the
// consent URL to the console and reads the exchange code from stdin —
// no local callback server needed. The resulting token is cached to
// token.json so later runs start silently.
func NewService(ctx context.Context) (*gmail.Service, error) {
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("gmail: reading %s: %w", credentialsFile, err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("gmail: parsing client secret: %w", err)
	}

	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok, err = tokenFromConsole(config)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenFile, tok); err != nil {
			return nil, err
		}
	}

	client := config.Client(ctx, tok)
	return gmail.NewService(ctx, option.WithHTTPClient(client))
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, err
	}
	return tok, nil
}

// tokenFromConsole runs the manual desktop-flow exchange: print the URL,
// block on stdin for the code the user pastes back after approving access.
func tokenFromConsole(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("Gmail authorization required. Open this URL in your browser:\n\n%s\n\nPaste the authorization code here: ", authURL)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("gmail: reading auth code: %w", err)
	}
	code := strings.TrimSpace(line)

	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("gmail: exchanging auth code: %w", err)
	}
	return tok, nil
}

func saveToken(file string, token *oauth2.Token) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("gmail: caching oauth token: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("gmail: writing %s: %w", file, err)
	}
	fmt.Printf("Gmail token saved to %s\n", file)
	return nil
}
