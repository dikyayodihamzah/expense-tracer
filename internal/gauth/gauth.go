package gauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	sheetsapi "google.golang.org/api/sheets/v4"
)

// NewClient returns an OAuth2 HTTP client and token source authenticated as the user.
// The HTTP client is suitable for REST-based Google APIs (e.g. Sheets).
// The token source is suitable for gRPC-based Google APIs (e.g. Cloud Vision).
// On first run it opens a browser consent flow and saves the token to tokenPath.
func NewClient(ctx context.Context, credentialsPath, tokenPath string) (*http.Client, oauth2.TokenSource, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read credentials file: %w", err)
	}

	cfg, err := google.ConfigFromJSON(b,
		sheetsapi.SpreadsheetsScope,
		"https://www.googleapis.com/auth/cloud-vision",
	)
	if err != nil {
		return nil, nil, fmt.Errorf("parse credentials: %w", err)
	}

	token, err := loadToken(tokenPath)
	if err != nil {
		token, err = fetchToken(cfg, tokenPath)
		if err != nil {
			return nil, nil, fmt.Errorf("oauth2 token: %w", err)
		}
	}

	ts := cfg.TokenSource(ctx, token)
	return oauth2.NewClient(ctx, ts), ts, nil
}

func loadToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var t oauth2.Token
	if err := json.NewDecoder(f).Decode(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

func fetchToken(cfg *oauth2.Config, savePath string) (*oauth2.Token, error) {
	cfg.RedirectURL = "urn:ietf:wg:oauth:2.0:oob"
	authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	log.Println("Open this URL in your browser to authorise the app:")
	log.Println(authURL)

	fmt.Print("Paste the authorisation code here: ")
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("read auth code: %w", err)
	}

	token, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("exchange auth code: %w", err)
	}

	f, err := os.Create(savePath)
	if err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		return nil, fmt.Errorf("encode token: %w", err)
	}
	log.Printf("token saved to %s", savePath)
	return token, nil
}
