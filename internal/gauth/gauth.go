package gauth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	sheetsapi "google.golang.org/api/sheets/v4"
)

// NewClient returns an HTTP client and token source authenticated as the
// service account in credentialsPath.
// The HTTP client is suitable for REST-based Google APIs (e.g. Sheets).
// The token source is suitable for gRPC-based Google APIs (e.g. Cloud Vision).
func NewClient(ctx context.Context, credentialsPath string) (*http.Client, oauth2.TokenSource, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read credentials file: %w", err)
	}

	cfg, err := google.JWTConfigFromJSON(b,
		sheetsapi.SpreadsheetsScope,
		"https://www.googleapis.com/auth/cloud-vision",
	)
	if err != nil {
		return nil, nil, fmt.Errorf("parse service account credentials: %w", err)
	}

	ts := cfg.TokenSource(ctx)
	return oauth2.NewClient(ctx, ts), ts, nil
}
