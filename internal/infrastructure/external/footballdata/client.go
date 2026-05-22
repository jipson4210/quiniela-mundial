package footballdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.football-data.org/v4"

// Client fetches match data from football-data.org.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// FetchMatches returns matches for a date range from football-data.org.
func (c *Client) FetchMatches(ctx context.Context, dateFrom, dateTo string) (*matchesResponse, error) {
	url := fmt.Sprintf("%s/matches?dateFrom=%s&dateTo=%s&competitions=WC", baseURL, dateFrom, dateTo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("footballdata: build request: %w", err)
	}
	req.Header.Set("X-Auth-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("footballdata: http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("footballdata: http %d: %s", resp.StatusCode, string(body))
	}

	var result matchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("footballdata: decode: %w", err)
	}
	return &result, nil
}
