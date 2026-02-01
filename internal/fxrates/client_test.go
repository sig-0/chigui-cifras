package fxrates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sig-0/fxrates/storage/types"
)

func TestClient_Rate(t *testing.T) {
	t.Parallel()

	var (
		asOf      = time.Date(2026, time.January, 2, 15, 4, 0, 0, time.UTC)
		fetchedAt = time.Date(2026, time.January, 2, 15, 5, 0, 0, time.UTC)

		expected = PageExchangeRate{
			Results: []ExchangeRate{{
				Base:      types.CurrencyUSD,
				Target:    types.CurrencyVES,
				Rate:      42,
				RateType:  types.RateTypeMID,
				Source:    types.SourceBCV,
				AsOf:      asOf,
				FetchedAt: fetchedAt,
			}},
			Total: 1,
		}
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/rates/USD/VES", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(expected))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, time.Second)

	resp, err := client.Rate(context.Background(), "USD", "VES")

	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, expected.Total, resp.Total)
	assert.Equal(t, expected.Results[0], resp.Results[0])
}

func TestClient_Rates(t *testing.T) {
	t.Parallel()

	var (
		asOf = time.Date(2026, time.January, 3, 10, 0, 0, 0, time.UTC)

		expected = PageExchangeRate{
			Results: []ExchangeRate{{
				Base:      types.CurrencyUSD,
				Target:    types.CurrencyVES,
				Rate:      50,
				RateType:  types.RateTypeMID,
				Source:    types.SourceBCV,
				AsOf:      asOf,
				FetchedAt: asOf,
			}},
			Total: 1,
		}
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/rates/USD", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(expected))
	}))

	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, time.Second)

	resp, err := client.Rates(context.Background(), "USD")

	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, expected.Total, resp.Total)
	assert.Equal(t, expected.Results[0], resp.Results[0])
}

func TestClient_Sources(t *testing.T) {
	t.Parallel()

	expected := SourcesResponse{
		Results: []types.Source{
			types.SourceBCV,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sources", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(expected))
	}))

	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, time.Second)

	resp, err := client.Sources(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expected, *resp)
}

func TestClient_Currencies(t *testing.T) {
	t.Parallel()

	expected := CurrenciesResponse{
		Results: []types.Currency{
			types.CurrencyUSD,
			types.CurrencyVES,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/currencies", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(expected))
	}))

	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, time.Second)

	resp, err := client.Currencies(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expected, *resp)
}

func TestClient_Health(t *testing.T) {
	t.Parallel()

	t.Run("healthy", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/health", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}))

		t.Cleanup(srv.Close)

		client := NewClient(srv.URL, time.Second)

		err := client.Health(context.Background())

		assert.NoError(t, err)
	})

	t.Run("unhealthy", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/health", r.URL.Path)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))

		t.Cleanup(srv.Close)

		client := NewClient(srv.URL, time.Second)

		err := client.Health(context.Background())

		require.Error(t, err)
		assert.ErrorContains(t, err, "unhealthy status code")
	})
}

func TestClient_Errors(t *testing.T) {
	t.Parallel()

	t.Run("unexpected status code", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))

		t.Cleanup(srv.Close)

		client := NewClient(srv.URL, time.Second)

		_, err := client.Rates(context.Background(), "USD")

		require.Error(t, err)
		assert.ErrorContains(t, err, "unexpected status code")
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = w.Write([]byte("{"))
		}))

		t.Cleanup(srv.Close)

		client := NewClient(srv.URL, time.Second)

		_, err := client.Rates(context.Background(), "USD")

		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to decode response")
	})
}
