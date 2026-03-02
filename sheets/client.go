package sheets

import (
	"context"
	"fmt"

	"google.golang.org/api/sheets/v4"
)

type Client struct {
	svc           *sheets.Service
	SpreadsheetID string
}

func NewClient(ctx context.Context, spreadsheetID string) (*Client, error) {
	svc, err := NewServiceFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{
		svc:           svc,
		SpreadsheetID: spreadsheetID,
	}, nil
}

func (c *Client) Service() *sheets.Service {
	return c.svc
}

func (c *Client) SpreadsheetURL() string {
	// handy for logs/debug (don’t rely on this for auth)
	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit", c.SpreadsheetID)
}
