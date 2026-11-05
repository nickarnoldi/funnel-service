package everflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	client, err := NewClient(&ClientConfig{
		APIKey:             "test-api-key",
		APIEndpoint:        server.URL,
		APITimeout:         time.Second,
		RateLimitPerSecond: 1000,
		DefaultPageSize:    2,
		Logger:             logger,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestNewClient(t *testing.T) {
	cfg := &ClientConfig{APIKey: "k"}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if cfg.APIEndpoint != DefaultAPIEndpoint {
		t.Errorf("APIEndpoint = %q", cfg.APIEndpoint)
	}
	if cfg.APITimeout != DefaultAPITimeout {
		t.Errorf("APITimeout = %s", cfg.APITimeout)
	}
	if cfg.DefaultPageSize != DefaultPageSize {
		t.Errorf("DefaultPageSize = %d", cfg.DefaultPageSize)
	}
	if cfg.RateLimitPerSecond != DefaultRateLimitPerSec {
		t.Errorf("RateLimitPerSecond = %d", cfg.RateLimitPerSecond)
	}
	if client.httpClient == nil || client.limiter == nil {
		t.Fatal("client was not initialized")
	}

	if _, err := NewClient(&ClientConfig{}); err == nil || !strings.Contains(err.Error(), "APIKey is required") {
		t.Fatalf("empty API key error = %v", err)
	}
}

func TestGetAdvertisersPaging(t *testing.T) {
	t.Run("stops on short page", func(t *testing.T) {
		var calls atomic.Int32
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			n := calls.Add(1)
			if r.Header.Get("X-Eflow-API-Key") != "test-api-key" {
				t.Errorf("api key = %q", r.Header.Get("X-Eflow-API-Key"))
			}
			if r.URL.Query().Get("page_size") != "2" {
				t.Errorf("page_size = %q", r.URL.Query().Get("page_size"))
			}
			if got := r.URL.Query().Get("page"); got != fmt.Sprint(n) {
				t.Errorf("page = %q, want %d", got, n)
			}
			type row struct {
				NetworkAdvertiserID int    `json:"network_advertiser_id"`
				Name                string `json:"name"`
			}
			var body struct {
				Advertisers []row `json:"advertisers"`
				Paging      struct {
					TotalCount int `json:"total_count"`
				} `json:"paging"`
			}
			body.Paging.TotalCount = 3
			switch n {
			case 1:
				body.Advertisers = []row{{NetworkAdvertiserID: 1, Name: "a"}, {NetworkAdvertiserID: 2, Name: "b"}}
			case 2:
				body.Advertisers = []row{{NetworkAdvertiserID: 3, Name: "c"}}
			default:
				t.Errorf("unexpected page %d", n)
			}
			_ = json.NewEncoder(w).Encode(body)
		})

		got, err := client.GetAdvertisers(context.Background(), AdvertisersRequestParams{})
		if err != nil {
			t.Fatalf("GetAdvertisers: %v", err)
		}
		if calls.Load() != 2 {
			t.Fatalf("calls = %d", calls.Load())
		}
		if len(*got) != 3 || (*got)[2].Name != "c" {
			t.Fatalf("advertisers = %+v", *got)
		}
	})

	t.Run("stops when total count is reached", func(t *testing.T) {
		var calls atomic.Int32
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			n := calls.Add(1)
			if n > 1 {
				t.Errorf("requested page %d after total_count was reached", n)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"advertisers": []map[string]any{
					{"network_advertiser_id": 1, "name": "a"},
					{"network_advertiser_id": 2, "name": "b"},
				},
				"paging": map[string]any{"total_count": 2},
			})
		})

		got, err := client.GetAdvertisers(context.Background(), AdvertisersRequestParams{})
		if err != nil {
			t.Fatalf("GetAdvertisers: %v", err)
		}
		if calls.Load() != 1 || len(*got) != 2 {
			t.Fatalf("calls = %d, rows = %d", calls.Load(), len(*got))
		}
	})
}

func TestListRequestBodies(t *testing.T) {
	t.Run("offers", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/networks/offerstable" {
				t.Errorf("request %s %s", r.Method, r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"search_terms":null,"filters":null}` {
				t.Errorf("body = %s", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"offers": []any{},
				"paging": map[string]any{"total_count": 0},
			})
		})
		if _, err := client.GetOffers(context.Background(), OffersRequestParams{}); err != nil {
			t.Fatalf("GetOffers: %v", err)
		}
	})

	t.Run("affiliates", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/networks/affiliatestable" {
				t.Errorf("request %s %s", r.Method, r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{}` {
				t.Errorf("body = %s", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"affiliates": []any{},
				"paging":     map[string]any{"total_count": 0},
			})
		})
		if _, err := client.GetAffiliates(context.Background(), GetAffiliatesParams{}); err != nil {
			t.Fatalf("GetAffiliates: %v", err)
		}
	})
}

func TestGetConversionsByDateTime(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/networks/reporting/conversions" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		var body ConversionRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.From != "2024-07-29 00:00:00" || body.To != "2024-07-29 01:00:00" {
			t.Errorf("range = %s .. %s", body.From, body.To)
		}
		if body.TimezoneID != 67 || body.CurrencyID != "USD" || !body.ShowConversions || !body.ShowEvents {
			t.Errorf("body = %+v", body)
		}
		resp := ConversionResponseBody{Paging: PagingInfo{TotalCount: 3}}
		switch n {
		case 1:
			resp.Conversions = []ConversionData{{ConversionID: "c1"}, {ConversionID: "c2"}}
		case 2:
			resp.Conversions = []ConversionData{{ConversionID: "c3"}}
		default:
			t.Errorf("unexpected page %d", n)
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	got, err := client.GetConversionsByDateTime(context.Background(), GetConversionsByDateTimeParams{
		FromDate:   "2024-07-29 00:00:00",
		ToDate:     "2024-07-29 01:00:00",
		TimezoneID: 67,
	})
	if err != nil {
		t.Fatalf("GetConversionsByDateTime: %v", err)
	}
	if calls.Load() != 2 || len(got) != 3 || got[2].ConversionID != "c3" {
		t.Fatalf("calls = %d, conversions = %+v", calls.Load(), got)
	}
}

func TestGetConversionsDefaultsTimezone(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body ConversionRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.TimezoneID != DefaultTimezoneID {
			t.Errorf("timezone = %d", body.TimezoneID)
		}
		_ = json.NewEncoder(w).Encode(ConversionResponseBody{})
	})
	_, err := client.GetConversionsByDateTime(context.Background(), GetConversionsByDateTimeParams{
		FromDate: "2024-07-29 00:00:00",
		ToDate:   "2024-07-29 01:00:00",
	})
	if err != nil {
		t.Fatalf("GetConversionsByDateTime: %v", err)
	}
}

func TestGetClicksByDateTime(t *testing.T) {
	t.Run("reverses a short page into chronological order", func(t *testing.T) {
		var calls atomic.Int32
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			_ = json.NewEncoder(w).Encode(ClickResponseBody{Clicks: []ClickData{
				{TransactionID: "new", UnixTimestamp: 300},
				{TransactionID: "old", UnixTimestamp: 100},
			}})
		})
		got, err := client.GetClicksByDateTime(context.Background(), GetClicksByDateTimeParams{
			FromDateTime: "2024-07-29 00:00:00",
			ToDateTime:   "2024-07-29 01:00:00",
			TimezoneID:   67,
		})
		if err != nil {
			t.Fatalf("GetClicksByDateTime: %v", err)
		}
		if calls.Load() != 1 || len(*got) != 2 || (*got)[0].TransactionID != "old" || (*got)[1].TransactionID != "new" {
			t.Fatalf("calls = %d, clicks = %+v", calls.Load(), *got)
		}
	})

	t.Run("walks a full page and drops the boundary duplicate", func(t *testing.T) {
		var calls atomic.Int32
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			n := calls.Add(1)
			var body ClickRequestBody
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
			}
			switch n {
			case 1:
				if body.To != "2024-07-29 01:00:00" {
					t.Errorf("first to = %q", body.To)
				}
				clicks := make([]ClickData, clickStreamPageSize)
				for i := range clicks {
					clicks[i] = ClickData{TransactionID: fmt.Sprintf("t%05d", i), UnixTimestamp: 2_000}
				}
				clicks[clickStreamPageSize-1] = ClickData{TransactionID: "boundary", UnixTimestamp: 1_000}
				_ = json.NewEncoder(w).Encode(ClickResponseBody{Clicks: clicks})
			case 2:
				wantTo := time.Unix(1_000, 0).UTC().Format("2006-01-02 15:04:05")
				if body.To != wantTo {
					t.Errorf("second to = %q, want %q", body.To, wantTo)
				}
				if body.From != "2024-07-29 00:00:00" || body.TimezoneID != 67 {
					t.Errorf("second body = %+v", body)
				}
				_ = json.NewEncoder(w).Encode(ClickResponseBody{Clicks: []ClickData{
					{TransactionID: "boundary", UnixTimestamp: 1_000},
					{TransactionID: "older", UnixTimestamp: 900},
				}})
			default:
				t.Errorf("unexpected request %d", n)
			}
		})

		got, err := client.GetClicksByDateTime(context.Background(), GetClicksByDateTimeParams{
			FromDateTime: "2024-07-29 00:00:00",
			ToDateTime:   "2024-07-29 01:00:00",
			TimezoneID:   67,
		})
		if err != nil {
			t.Fatalf("GetClicksByDateTime: %v", err)
		}
		if calls.Load() != 2 {
			t.Fatalf("calls = %d", calls.Load())
		}
		clicks := *got
		if len(clicks) != clickStreamPageSize+1 {
			t.Fatalf("len = %d", len(clicks))
		}
		if clicks[0].TransactionID != "older" || clicks[1].TransactionID != "boundary" {
			t.Fatalf("start = %s, %s", clicks[0].TransactionID, clicks[1].TransactionID)
		}
		if clicks[len(clicks)-1].TransactionID != "t00000" {
			t.Fatalf("end = %s", clicks[len(clicks)-1].TransactionID)
		}
		boundary := 0
		for _, click := range clicks {
			if click.TransactionID == "boundary" {
				boundary++
			}
		}
		if boundary != 1 {
			t.Fatalf("boundary count = %d", boundary)
		}
	})
}

func TestGetClickByID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/networks/reporting/clicks/abc123" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ClickData{TransactionID: "abc123"})
	})
	got, err := client.GetClickByID(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetClickByID: %v", err)
	}
	if got.TransactionID != "abc123" {
		t.Fatalf("click = %+v", got)
	}
	if _, err := client.GetClickByID(context.Background(), ""); err == nil {
		t.Fatal("empty transaction id returned a click")
	}
}

func TestRequestErrors(t *testing.T) {
	t.Run("rate limit", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"slow down"}`))
		})
		_, err := client.GetCategories(context.Background())
		if !errors.Is(err, ErrRateLimited) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("status message", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"no thanks"}`))
		})
		_, err := client.GetCategories(context.Background())
		if err == nil || !strings.Contains(err.Error(), "everflow API error (status 500): no thanks") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		_, err := client.GetCategories(context.Background())
		if err == nil || !strings.Contains(err.Error(), "empty response body") {
			t.Fatalf("error = %v", err)
		}
	})
}
