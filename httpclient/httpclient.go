package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"log/slog"
)

// ClientConfig holds configuration for the CommonHTTPClient.
type ClientConfig struct {
	BaseURL           *url.URL
	DefaultHeaders    map[string]string
	DisableLogBody    bool
	DisableLogHeaders bool
	DisableLogQuery   bool
	MaxRetries        int
	RetryBackoff      time.Duration
	Logger            *slog.Logger
	HTTPClient        *http.Client
}

// RequestOptions allows per-request customizations.
type RequestOptions struct {
	Path        string
	Method      string
	Headers     map[string]string
	QueryParams map[string]string
	Body        io.Reader
	// Optional Timeout for this request (overrides client default if set)
	Timeout time.Duration
}

// CommonHTTPClient is the wrapper around the standard http.Client.
type CommonHTTPClient struct {
	baseURL           *url.URL
	defaultHeaders    map[string]string
	disableLogBody    bool
	disableLogHeaders bool
	disableLogQuery   bool
	maxRetries        int
	retryBackoff      time.Duration
	logger            *slog.Logger
	client            *http.Client
}

// NewCommonHTTPClient creates a new client with the provided config.
func NewCommonHTTPClient(cfg ClientConfig) *CommonHTTPClient {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &CommonHTTPClient{
		baseURL:           cfg.BaseURL,
		defaultHeaders:    cfg.DefaultHeaders,
		disableLogBody:    cfg.DisableLogBody,
		disableLogHeaders: cfg.DisableLogHeaders,
		disableLogQuery:   cfg.DisableLogQuery,
		maxRetries:        cfg.MaxRetries,
		retryBackoff:      cfg.RetryBackoff,
		logger:            cfg.Logger,
		client:            cfg.HTTPClient,
	}
}

// Do executes an HTTP request with the given options, retries if configured, and logs details.
func (c *CommonHTTPClient) Do(ctx context.Context, opts RequestOptions) (*http.Response, error) {
	// Construct the request URL
	var reqURL *url.URL
	if c.baseURL != nil {
		reqURL = c.baseURL.ResolveReference(&url.URL{Path: opts.Path})
	} else {
		parsed, err := url.Parse(opts.Path)
		if err != nil {
			return nil, err
		}
		reqURL = parsed
	}

	// Add query parameters
	if len(opts.QueryParams) > 0 {
		q := reqURL.Query()
		for k, v := range opts.QueryParams {
			q.Set(k, v)
		}
		reqURL.RawQuery = q.Encode()
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, opts.Method, reqURL.String(), opts.Body)
	if err != nil {
		return nil, err
	}

	// Apply default headers
	for k, v := range c.defaultHeaders {
		req.Header.Set(k, v)
	}

	// Apply request-specific headers
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	// If a per-request timeout is set, create a context with timeout
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// Log the outgoing request
	c.logRequest(req, opts.Body)

	// Perform retries
	var resp *http.Response
	var attempt int
	var lastErr error
	for attempt = 0; attempt <= c.maxRetries; attempt++ {
		resp, lastErr = c.client.Do(req)
		if lastErr == nil && resp.StatusCode < 500 {
			// Successful or non-retriable status
			break
		}
		// If we are here, either an error occurred, or a 5xx was returned
		if attempt < c.maxRetries {
			time.Sleep(c.retryBackoff)
		}
	}

	if lastErr != nil {
		// This is a final error after retries
		c.logger.Error("HTTP request failed", slog.String("url", req.URL.String()), slog.Any("error", lastErr))
		return nil, lastErr
	}

	defer func() {
		// We want to ensure response body can be read for logging.
		// Caller should handle reading the body again if needed.
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	// Read body for logging and then recreate a new ReadCloser for response
	var responseBody []byte
	if resp.Body != nil {
		responseBody, err = io.ReadAll(resp.Body)
		if err != nil {
			c.logger.Error("Error reading response body", slog.String("url", req.URL.String()), slog.Any("error", err))
			return nil, err
		}
		resp.Body = io.NopCloser(bytes.NewReader(responseBody))
	}

	c.logResponse(resp, responseBody)
	return resp, nil
}

// logRequest logs request details based on the client configuration.
func (c *CommonHTTPClient) logRequest(req *http.Request, body io.Reader) {
	var bodyStr string
	if !c.disableLogBody && body != nil {
		// Body might have been consumed; consider buffering the body upstream if needed.
		// For demonstration, we assume body is a type like bytes.Reader or can be re-constructed.
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(body); err == nil {
			bodyStr = buf.String()
		}
		// Recreate the body so it can be sent again
		req.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
	}

	var headers map[string][]string
	if !c.disableLogHeaders {
		headers = req.Header
	}

	query := ""
	if !c.disableLogQuery {
		query = req.URL.RawQuery
	}

	c.logger.Info("Outgoing request",
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.String("query", query),
		slog.Any("headers", headers),
		slog.String("body", bodyStr),
	)
}

// logResponse logs response details based on the client configuration.
func (c *CommonHTTPClient) logResponse(resp *http.Response, responseBody []byte) {
	var headers http.Header
	if !c.disableLogHeaders {
		headers = resp.Header
	}

	var bodyStr string
	if !c.disableLogBody && len(responseBody) > 0 {
		bodyStr = string(responseBody)
	}

	c.logger.Info("Incoming response",
		slog.Int("status_code", resp.StatusCode),
		slog.Any("headers", headers),
		slog.String("body", bodyStr),
	)
}

// Example of an input/output processor - you can adapt this as needed.
// For now, it's a simple helper to decode JSON responses.
func DecodeJSONResponse(resp *http.Response, v interface{}) error {
	if resp.Body == nil {
		return errors.New("no response body")
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	return dec.Decode(v)
}
