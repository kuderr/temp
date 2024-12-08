package httpclient2

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// AuthMethod defines different types of authentication
type AuthMethod string

const (
	AuthNone   AuthMethod = "none"
	AuthBasic  AuthMethod = "basic"
	AuthBearer AuthMethod = "bearer"
	AuthApiKey AuthMethod = "apikey"
	AuthOAuth  AuthMethod = "oauth"
)

// ClientOption allows configuring the HTTP client
type ClientOption func(*Client)

// Client represents a configurable HTTP client
type Client struct {
	httpClient     *http.Client
	baseURL        string
	defaultHeaders map[string]string
	authMethod     AuthMethod
	authConfig     map[string]string
}

// New creates a new HTTP client with optional configurations
func New(options ...ClientOption) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		defaultHeaders: make(map[string]string),
		authMethod:     AuthNone,
		authConfig:     make(map[string]string),
	}

	// Apply provided options
	for _, opt := range options {
		opt(client)
	}

	return client
}

// WithBaseURL sets the base URL for all requests
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithDefaultHeaders sets default headers for all requests
func WithDefaultHeaders(headers map[string]string) ClientOption {
	return func(c *Client) {
		for k, v := range headers {
			c.defaultHeaders[k] = v
		}
	}
}

// WithBasicAuth configures basic authentication
func WithBasicAuth(username, password string) ClientOption {
	return func(c *Client) {
		c.authMethod = AuthBasic
		c.authConfig["username"] = username
		c.authConfig["password"] = password
	}
}

// WithBearerToken configures bearer token authentication
func WithBearerToken(token string) ClientOption {
	return func(c *Client) {
		c.authMethod = AuthBearer
		c.authConfig["token"] = token
	}
}

// WithAPIKey configures API key authentication
func WithAPIKey(key, location string) ClientOption {
	return func(c *Client) {
		c.authMethod = AuthApiKey
		c.authConfig["key"] = key
		c.authConfig["location"] = location // "header" or "query"
	}
}

// WithInsecureSkipVerify allows skipping TLS certificate verification
func WithInsecureSkipVerify(skip bool) ClientOption {
	return func(c *Client) {
		if skip {
			transport := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			c.httpClient.Transport = transport
		}
	}
}

// Request represents an HTTP request configuration
type Request struct {
	Method  string
	Path    string
	Headers map[string]string
	Query   map[string]string
	Body    interface{}
}

// Do sends an HTTP request and returns the response
func (c *Client) Do(ctx context.Context, req Request) (*http.Response, error) {
	// Construct full URL
	fullURL, err := c.buildURL(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %v", err)
	}

	// Prepare request body
	var body io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		body = bytes.NewBuffer(jsonBody)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set default headers
	for k, v := range c.defaultHeaders {
		httpReq.Header.Set(k, v)
	}

	// Set request-specific headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Apply authentication
	c.applyAuthentication(httpReq)

	// Send request
	return c.httpClient.Do(httpReq)
}

// buildURL constructs the full URL with base URL and query parameters
func (c *Client) buildURL(req Request) (string, error) {
	// Combine base URL with request path
	fullURL := c.baseURL + req.Path

	// Parse the URL
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return "", err
	}

	// Add query parameters
	query := parsedURL.Query()
	for k, v := range req.Query {
		query.Add(k, v)
	}

	// Handle API key in query if applicable
	if c.authMethod == AuthApiKey && c.authConfig["location"] == "query" {
		query.Add("api_key", c.authConfig["key"])
	}

	// Set the modified query
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// applyAuthentication adds authentication to the request based on configured method
func (c *Client) applyAuthentication(req *http.Request) {
	switch c.authMethod {
	case AuthBasic:
		req.SetBasicAuth(c.authConfig["username"], c.authConfig["password"])
	case AuthBearer:
		req.Header.Set("Authorization", "Bearer "+c.authConfig["token"])
	case AuthApiKey:
		if c.authConfig["location"] == "header" {
			req.Header.Set("X-API-Key", c.authConfig["key"])
		}
	}
}

// ReadJSONResponse reads and unmarshals JSON response
func ReadJSONResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return nil
}

// Helper function to create Basic Auth header manually if needed
func CreateBasicAuthHeader(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
