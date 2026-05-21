package openfootball

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://raw.githubusercontent.com/openfootball/worldcup.json/master/2026/worldcup.json"

// Client fetches the openfootball worldcup.json file.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client with the default worldcup.json URL.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchWorldCupJSON downloads and returns the raw worldcup.json bytes.
func (c *Client) FetchWorldCupJSON(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openfootball: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openfootball: http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openfootball: http %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// Decode parses raw JSON bytes into the DTO.
func Decode(raw []byte) (*worldcupDTO, error) {
	var wc worldcupDTO
	if err := json.Unmarshal(raw, &wc); err != nil {
		return nil, fmt.Errorf("openfootball: decode: %w", err)
	}
	return &wc, nil
}
