package sheets

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Env vars supported:
// - GOOGLE_APPLICATION_CREDENTIALS: path to service account JSON file
// - GOOGLE_SHEETS_CREDENTIALS_JSON: raw JSON string (optional alternative)
func NewServiceFromEnv(ctx context.Context) (*sheets.Service, error) {
	if jsonStr := strings.TrimSpace(os.Getenv("GOOGLE_SHEETS_CREDENTIALS_JSON")); jsonStr != "" {
		return sheets.NewService(ctx, option.WithCredentialsJSON([]byte(jsonStr)))
	}

	if path := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read GOOGLE_APPLICATION_CREDENTIALS file: %w", err)
		}
		return sheets.NewService(ctx, option.WithCredentialsJSON(b))
	}

	return nil, fmt.Errorf("missing credentials: set GOOGLE_APPLICATION_CREDENTIALS (file path) or GOOGLE_SHEETS_CREDENTIALS_JSON (raw json)")
}
