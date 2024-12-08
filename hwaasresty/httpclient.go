package hwaasresty

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"time"

	"log/slog"

	"github.com/go-resty/resty/v2"
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
	HTTPTimeout       time.Duration
}

// RequestOptions allows per-request customizations.
type RequestOptions struct {
	Path        string
	Method      string
	Headers     map[string]string
	QueryParams map[string]string
	Body        io.Reader
	Timeout     time.Duration
}

// CommonHTTPClient is the wrapper around resty.Client.
type CommonHTTPClient struct {
	client            *resty.Client
	baseURL           *url.URL
	defaultHeaders    map[string]string
	disableLogBody    bool
	disableLogHeaders bool
	disableLogQuery   bool
	logger            *slog.Logger
}

// NewCommonHTTPClient creates a new client with the provided config.
func NewCommonHTTPClient(cfg ClientConfig) *CommonHTTPClient {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	client := resty.New()

	if cfg.HTTPTimeout > 0 {
		client.SetTimeout(cfg.HTTPTimeout)
	} else {
		client.SetTimeout(30 * time.Second)
	}

	if cfg.MaxRetries > 0 {
		client.
			SetRetryCount(cfg.MaxRetries).
			SetRetryWaitTime(cfg.RetryBackoff).
			SetRetryAfter(func(client *resty.Client, resp *resty.Response) (time.Duration, error) {
				// Simple retry after logic: always wait RetryBackoff between tries
				return cfg.RetryBackoff, nil
			})
	}

	// Base URL set at resty level if provided
	if cfg.BaseURL != nil {
		client.SetBaseURL(cfg.BaseURL.String())
	}

	commonClient := &CommonHTTPClient{
		client:            client,
		baseURL:           cfg.BaseURL,
		defaultHeaders:    cfg.DefaultHeaders,
		disableLogBody:    cfg.DisableLogBody,
		disableLogHeaders: cfg.DisableLogHeaders,
		disableLogQuery:   cfg.DisableLogQuery,
		logger:            cfg.Logger,
	}

	// Set hooks for logging
	commonClient.client.OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		commonClient.logRequest(r)
		return nil
	})

	commonClient.client.OnAfterResponse(func(c *resty.Client, r *resty.Response) error {
		commonClient.logResponse(r)
		return nil
	})

	return commonClient
}

// Do executes an HTTP request with the given options.
func (c *CommonHTTPClient) Do(ctx context.Context, opts RequestOptions) (*resty.Response, error) {
	req := c.client.R().SetContext(ctx)

	// Set headers
	for k, v := range c.defaultHeaders {
		req.SetHeader(k, v)
	}

	for k, v := range opts.Headers {
		req.SetHeader(k, v)
	}

	// Set query params
	if len(opts.QueryParams) > 0 {
		req.SetQueryParams(opts.QueryParams)
	}

	// If a per-request timeout is set, configure a context-based timeout
	var cancel func()
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
		req.SetContext(ctx)
	}

	// If there is a body, we need to read it fully to set in resty
	// Resty expects either a reader or body directly.
	var bodyBytes []byte
	if opts.Body != nil {
		b, err := io.ReadAll(opts.Body)
		if err != nil {
			return nil, err
		}
		bodyBytes = b
		req.SetBody(bodyBytes)
	}

	// Perform request by Method
	var resp *resty.Response
	var err error
	switch opts.Method {
	case "GET":
		resp, err = req.Get(opts.Path)
	case "POST":
		resp, err = req.Post(opts.Path)
	case "PUT":
		resp, err = req.Put(opts.Path)
	case "DELETE":
		resp, err = req.Delete(opts.Path)
	case "PATCH":
		resp, err = req.Patch(opts.Path)
	case "HEAD":
		resp, err = req.Head(opts.Path)
	default:
		return nil, errors.New("unsupported method")
	}

	if err != nil {
		c.logger.Error("HTTP request failed", slog.String("url", resp.Request.URL), slog.Any("error", err))
		return nil, err
	}

	return resp, nil
}

// logRequest logs request details before sending it.
func (c *CommonHTTPClient) logRequest(r *resty.Request) {
	var headers map[string][]string
	if !c.disableLogHeaders {
		// Convert resty headers type to map[string][]string
		headers = make(map[string][]string)
		for k, v := range r.Header {
			headers[k] = v
		}
	}

	queryStr := ""
	if !c.disableLogQuery {
		queryValues := r.QueryParam
		if queryValues != nil {
			queryBytes, _ := json.Marshal(queryValues)
			queryStr = string(queryBytes)
		}
	}

	bodyStr := ""
	if !c.disableLogBody && r.RawRequest != nil && r.RawRequest.Body != nil {
		// We already have the body in r.body (bytes)
		if body, ok := r.Body.(string); ok {
			bodyStr = body
		} else if b, ok := r.Body.([]byte); ok {
			bodyStr = string(b)
		}
	}

	c.logger.Info("Outgoing request",
		slog.String("method", r.Method),
		slog.String("url", r.URL),
		slog.String("query", queryStr),
		slog.Any("headers", headers),
		slog.String("body", bodyStr),
	)
}

// logResponse logs response details after receiving it.
func (c *CommonHTTPClient) logResponse(resp *resty.Response) {
	var headers map[string][]string
	if !c.disableLogHeaders {
		headers = make(map[string][]string)
		for k, v := range resp.Header() {
			headers[k] = v
		}
	}

	var bodyStr string
	if !c.disableLogBody && resp.Body() != nil {
		bodyStr = string(resp.Body())
	}

	c.logger.Info("Incoming response",
		slog.Int("status_code", resp.StatusCode()),
		slog.Any("headers", headers),
		slog.String("body", bodyStr),
	)
}

// Example of an input/output processor
func DecodeJSONResponse(resp *resty.Response, v interface{}) error {
	if resp.Body() == nil {
		return errors.New("no response body")
	}
	return json.Unmarshal(resp.Body(), v)
}
