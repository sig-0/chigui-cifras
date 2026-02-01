package fxrates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client is an HTTP client for the fxrates API
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new fxrates API client
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Rate fetches the exchange rate for a specific currency pair
func (c *Client) Rate(ctx context.Context, base, target string) (*PageExchangeRate, error) {
	path := fmt.Sprintf("/v1/rates/%s/%s", base, target)

	return get[PageExchangeRate](ctx, c, path)
}

// Rates fetches all exchange rates for a base currency
func (c *Client) Rates(ctx context.Context, base string) (*PageExchangeRate, error) {
	path := fmt.Sprintf("/v1/rates/%s", base)

	return get[PageExchangeRate](ctx, c, path)
}

// Sources fetches the list of available rate sources
func (c *Client) Sources(ctx context.Context) (*SourcesResponse, error) {
	return get[SourcesResponse](ctx, c, "/v1/sources")
}

// Currencies fetches the list of supported currencies
func (c *Client) Currencies(ctx context.Context) (*CurrenciesResponse, error) {
	return get[CurrenciesResponse](ctx, c, "/v1/currencies")
}

// Health checks if the API is healthy
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", http.NoBody)
	if err != nil {
		return fmt.Errorf("unable to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	return nil
}

func get[T any](ctx context.Context, c *Client, path string) (*T, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("unable to parse url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result T
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("unable to decode response: %w", err)
	}

	return &result, nil
}
