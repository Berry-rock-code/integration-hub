package sheets

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// NewServiceFromEnv creates a Sheets client using:
//
// 1) GOOGLE_SHEETS_CREDENTIALS_JSON
// 2) GOOGLE_APPLICATION_CREDENTIALS (file path)
// 3) Application Default Credentials (Cloud Run / GCE / gcloud auth)
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

	// Cloud Run / GCP default credentials fallback
	return sheets.NewService(ctx, option.WithScopes(sheets.SpreadsheetsScope))
}
