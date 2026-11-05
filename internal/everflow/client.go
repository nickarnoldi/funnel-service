// Package everflow is a small client for the Everflow endpoints this service syncs.
// Each request is a single attempt. Callers retry by running the sync again.
package everflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

const (
	DefaultAPIEndpoint     = "https://api.eflow.team/v1"
	DefaultAPITimeout      = 60 * time.Second
	DefaultRateLimitPerSec = 5
	DefaultPageSize        = 1000
	MaxPageSize            = 2000
	DefaultTimezoneID      = 67 // UTC
	DefaultCurrencyID      = "USD"

	// The click stream returns at most this many rows, newest first, with no paging object.
	// A full page means older rows still exist.
	clickStreamPageSize = 10_000
)

var everflowTimezones = map[int]string{
	67: "UTC",
}

// ErrRateLimited is returned when Everflow responds with HTTP 429.
var ErrRateLimited = errors.New("everflow API rate limit exceeded")

// ClientConfig configures an Everflow API client.
// Zero values other than APIKey are replaced by the defaults in NewClient.
type ClientConfig struct {
	APIEndpoint        string
	APIKey             string
	APITimeout         time.Duration
	RateLimitPerSecond int
	DefaultPageSize    int
	Logger             *logrus.Logger
	HTTPClient         *http.Client
}

// Client calls the Everflow API.
type Client struct {
	config     *ClientConfig
	httpClient *http.Client
	limiter    *rate.Limiter
}

// NewClient validates config, applies defaults, and returns a client.
func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("invalid config: APIKey is required")
	}
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: config.APITimeout}
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		limiter:    rate.NewLimiter(rate.Limit(config.RateLimitPerSecond), config.RateLimitPerSecond),
	}, nil
}

func (c *ClientConfig) validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("APIKey is required")
	}
	if c.APIEndpoint == "" {
		c.APIEndpoint = DefaultAPIEndpoint
	}
	if _, err := url.ParseRequestURI(c.APIEndpoint); err != nil {
		return fmt.Errorf("invalid APIEndpoint: %w", err)
	}
	if c.APITimeout <= 0 {
		c.APITimeout = DefaultAPITimeout
	}
	if c.RateLimitPerSecond <= 0 {
		c.RateLimitPerSecond = DefaultRateLimitPerSec
	}
	if c.DefaultPageSize <= 0 {
		c.DefaultPageSize = DefaultPageSize
	}
	if c.DefaultPageSize > MaxPageSize {
		c.DefaultPageSize = MaxPageSize
	}
	if c.Logger == nil {
		c.Logger = logrus.New()
		c.Logger.SetFormatter(&logrus.JSONFormatter{})
		c.Logger.SetLevel(logrus.ErrorLevel)
	}
	return nil
}

func (c *Client) pageSize(requested int) int {
	size := c.config.DefaultPageSize
	if requested > 0 {
		size = requested
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	return size
}

// do performs one rate-limited request and returns the response body.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("context cancelled while waiting for rate limiter: %w", err)
	}

	fullURL := c.config.APIEndpoint + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("X-Eflow-API-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.config.Logger.WithError(err).WithField("path", path).Error("everflow request failed")
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		c.config.Logger.WithField("path", path).Error("everflow rate limit exceeded")
		return nil, ErrRateLimited
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := "invalid response status code: " + strconv.Itoa(resp.StatusCode)
		var parsed struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &parsed) == nil && parsed.Message != "" {
			msg = parsed.Message
		}
		err := fmt.Errorf("everflow API error (status %d): %s", resp.StatusCode, msg)
		c.config.Logger.WithError(err).WithField("path", path).Error("everflow request failed")
		return nil, err
	}
	if len(respBody) == 0 {
		err := fmt.Errorf("received empty response body with status code %d", resp.StatusCode)
		c.config.Logger.WithError(err).WithField("path", path).Error("everflow request failed")
		return nil, err
	}
	return respBody, nil
}

// paginate walks a 1-based Everflow list. It stops on a short page, or when
// totalCount is positive and that many rows have been read.
func (c *Client) paginate(ctx context.Context, method, path string, body any, pageSize int, consume func([]byte) (int, int, error)) error {
	page := 1
	fetched := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("page_size", strconv.Itoa(pageSize))

		respBody, err := c.do(ctx, method, path, query, body)
		if err != nil {
			return err
		}
		n, total, err := consume(respBody)
		if err != nil {
			return err
		}
		fetched += n
		if n < pageSize || (total > 0 && fetched >= total) {
			return nil
		}
		page++
	}
}

// GetCategories returns every category. The categories endpoint is not paged.
func (c *Client) GetCategories(ctx context.Context) (*[]CategoryData, error) {
	respBody, err := c.do(ctx, http.MethodGet, "/networks/categories", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching categories: %w", err)
	}
	var response CategoriesResponseBody
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling categories: %w", err)
	}
	return &response.Categories, nil
}

// GetAdvertisers returns every advertiser page.
func (c *Client) GetAdvertisers(ctx context.Context, params AdvertisersRequestParams) (*[]AdvertiserData, error) {
	var all []AdvertiserData
	err := c.paginate(ctx, http.MethodGet, "/networks/advertisers", nil, c.pageSize(params.PageSize), func(respBody []byte) (int, int, error) {
		var response AdvertisersResponseBody
		if err := json.Unmarshal(respBody, &response); err != nil {
			return 0, 0, fmt.Errorf("error unmarshaling advertisers: %w", err)
		}
		all = append(all, response.Advertisers...)
		return len(response.Advertisers), response.Paging.TotalCount, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching advertisers: %w", err)
	}
	return &all, nil
}

// GetOfferURLs returns every offer URL page.
func (c *Client) GetOfferURLs(ctx context.Context, params OfferURLsRequestParams) (*[]OfferURLData, error) {
	var all []OfferURLData
	err := c.paginate(ctx, http.MethodGet, "/networks/offerurls", nil, c.pageSize(params.PageSize), func(respBody []byte) (int, int, error) {
		var response OfferURLsResponseBody
		if err := json.Unmarshal(respBody, &response); err != nil {
			return 0, 0, fmt.Errorf("error unmarshaling offer urls: %w", err)
		}
		all = append(all, response.URLs...)
		return len(response.URLs), response.Paging.TotalCount, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching offer urls: %w", err)
	}
	return &all, nil
}

// GetOffers returns every offer page. An empty params value posts null search terms and filters.
func (c *Client) GetOffers(ctx context.Context, params OffersRequestParams) (*[]OfferData, error) {
	body := OffersRequestBody{
		SearchTerms: params.SearchTerms,
		Filters:     params.Filters,
	}
	var all []OfferData
	err := c.paginate(ctx, http.MethodPost, "/networks/offerstable", body, c.pageSize(params.PageSize), func(respBody []byte) (int, int, error) {
		var response OffersResponseBody
		if err := json.Unmarshal(respBody, &response); err != nil {
			return 0, 0, fmt.Errorf("error unmarshaling offers: %w", err)
		}
		all = append(all, response.Offers...)
		return len(response.Offers), response.Paging.TotalCount, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching offers: %w", err)
	}
	return &all, nil
}

// GetAffiliates returns every affiliate page. The request body is an empty JSON object.
func (c *Client) GetAffiliates(ctx context.Context, params GetAffiliatesParams) (*[]AffiliateData, error) {
	var all []AffiliateData
	err := c.paginate(ctx, http.MethodPost, "/networks/affiliatestable", struct{}{}, c.pageSize(params.PageSize), func(respBody []byte) (int, int, error) {
		var response AffiliatesResponseBody
		if err := json.Unmarshal(respBody, &response); err != nil {
			return 0, 0, fmt.Errorf("error unmarshaling affiliates: %w", err)
		}
		all = append(all, response.Affiliates...)
		return len(response.Affiliates), response.Paging.TotalCount, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching affiliates: %w", err)
	}
	return &all, nil
}

// GetConversionsByDateTime returns conversions in [from, to].
// Timezone 0 uses DefaultTimezoneID. An empty currency uses USD.
// show_conversions and show_events default to true.
func (c *Client) GetConversionsByDateTime(ctx context.Context, params GetConversionsByDateTimeParams) ([]ConversionData, error) {
	if params.FromDate == "" || params.ToDate == "" {
		return nil, fmt.Errorf("FromDate and ToDate are required")
	}
	if _, err := time.Parse("2006-01-02 15:04:05", params.FromDate); err != nil {
		return nil, fmt.Errorf("invalid FromDate format (expect YYYY-MM-DD HH:MM:SS): %w", err)
	}
	if _, err := time.Parse("2006-01-02 15:04:05", params.ToDate); err != nil {
		return nil, fmt.Errorf("invalid ToDate format (expect YYYY-MM-DD HH:MM:SS): %w", err)
	}

	timezoneID := DefaultTimezoneID
	if params.TimezoneID != 0 {
		timezoneID = params.TimezoneID
	}
	currencyID := DefaultCurrencyID
	if params.CurrencyID != "" {
		currencyID = params.CurrencyID
	}
	showConversions := true
	if params.ShowConversions != nil {
		showConversions = *params.ShowConversions
	}
	showEvents := true
	if params.ShowEvents != nil {
		showEvents = *params.ShowEvents
	}

	body := ConversionRequestBody{
		From:            params.FromDate,
		To:              params.ToDate,
		TimezoneID:      timezoneID,
		CurrencyID:      currencyID,
		ShowConversions: showConversions,
		ShowEvents:      showEvents,
	}

	var all []ConversionData
	page := 0
	err := c.paginate(ctx, http.MethodPost, "/networks/reporting/conversions", body, c.pageSize(params.PageSize), func(respBody []byte) (int, int, error) {
		page++
		var response ConversionResponseBody
		if err := json.Unmarshal(respBody, &response); err != nil {
			return 0, 0, fmt.Errorf("error unmarshaling conversions page %d: %w", page, err)
		}
		all = append(all, response.Conversions...)
		return len(response.Conversions), response.Paging.TotalCount, nil
	})
	if err != nil {
		if errors.Is(err, ErrRateLimited) {
			return nil, err
		}
		return nil, fmt.Errorf("error fetching conversions: %w", err)
	}
	return all, nil
}

// GetClickByID loads one click by its transaction id.
func (c *Client) GetClickByID(ctx context.Context, transactionID string) (*ClickData, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("TransactionID is required")
	}
	respBody, err := c.do(ctx, http.MethodGet, "/networks/reporting/clicks/"+transactionID, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching click: %w", err)
	}
	var click ClickData
	if err := json.Unmarshal(respBody, &click); err != nil {
		return nil, fmt.Errorf("error unmarshaling click: %w", err)
	}
	return &click, nil
}

// GetClicksByDateTime walks the click stream for a time range.
// Everflow returns rows newest first and caps a response at clickStreamPageSize.
// A full page is followed by another request whose `to` is the oldest timestamp
// in that page, formatted in the requested timezone. Transaction ids already
// seen at that timestamp are skipped. The returned slice is oldest first.
func (c *Client) GetClicksByDateTime(ctx context.Context, params GetClicksByDateTimeParams) (*[]ClickData, error) {
	if params.FromDateTime == "" || params.ToDateTime == "" {
		return nil, fmt.Errorf("FromDateTime and ToDateTime are required")
	}
	if params.TimezoneID == 0 {
		return nil, fmt.Errorf("TimezoneID is required")
	}
	location, err := timezoneLocation(params.TimezoneID)
	if err != nil {
		return nil, fmt.Errorf("error getting timezone location: %w", err)
	}

	var clicks []ClickData
	var boundaryTimestamp int64
	var boundaryIDs map[string]bool
	toDate := params.ToDateTime

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		body := ClickRequestBody{
			From:       params.FromDateTime,
			To:         toDate,
			TimezoneID: params.TimezoneID,
		}
		respBody, err := c.do(ctx, http.MethodPost, "/networks/reporting/clicks/stream", nil, body)
		if err != nil {
			return nil, fmt.Errorf("error fetching click stream: %w", err)
		}
		var response ClickResponseBody
		if err := json.Unmarshal(respBody, &response); err != nil {
			return nil, fmt.Errorf("error unmarshaling click stream: %w", err)
		}

		fullPage := len(response.Clicks) == clickStreamPageSize
		var oldestTimestamp int64
		if fullPage {
			oldestTimestamp = response.Clicks[len(response.Clicks)-1].UnixTimestamp
		}
		nextBoundary := make(map[string]bool)

		for _, click := range response.Clicks {
			if boundaryTimestamp > 0 && click.UnixTimestamp == boundaryTimestamp && boundaryIDs[click.TransactionID] {
				continue
			}
			if oldestTimestamp != 0 && click.UnixTimestamp == oldestTimestamp {
				nextBoundary[click.TransactionID] = true
			}
			clicks = append(clicks, click)
		}
		if !fullPage {
			break
		}

		toDate = time.Unix(oldestTimestamp, 0).In(location).Format("2006-01-02 15:04:05")
		boundaryIDs = nextBoundary
		boundaryTimestamp = oldestTimestamp
	}

	slices.Reverse(clicks)
	return &clicks, nil
}

func timezoneLocation(timezoneID int) (*time.Location, error) {
	name, ok := everflowTimezones[timezoneID]
	if !ok {
		return nil, fmt.Errorf("timezone ID %d not found in mapping", timezoneID)
	}
	return time.LoadLocation(name)
}
