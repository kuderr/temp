package restydemo

import (
	"context"
	"encoding/json"
	"httpclient/models"
	"httpclient/utils"
	"io"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

func Example() {
	// Configuration flags to control logging
	disableLogBody := false
	disableLogHeaders := false
	disableLogQuery := false

	// Initialize slog logger
	logger := slog.New(utils.NewPrettyJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	baseURL, _ := url.Parse("https://jsonplaceholder.typicode.com")

	// Create a Resty client
	client := resty.New().
		SetBaseURL(baseURL.String()).
		SetTimeout(30*time.Second).
		SetHeader("Authorization", "Bearer token").
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		// Before sending the request, log request details
		OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
			logRequest(logger, r, disableLogBody, disableLogHeaders, disableLogQuery)
			return nil
		}).
		// After receiving the response, log response details
		OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
			logResponse(logger, resp, disableLogBody, disableLogHeaders)
			return nil
		})

	// Prepare request options
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.R().
		SetContext(ctx).
		SetQueryParam("limit", "5").
		Get("/posts")

	if err != nil {
		logger.Error("Request failed", slog.Any("error", err))
		return
	}

	// Decode JSON response
	var data []models.Post
	if err := json.Unmarshal(resp.Body(), &data); err != nil {
		logger.Error("Decode error", slog.Any("error", err))
		return
	}

	logger.Info("Response data:", slog.Any("first_post", data[0]))

	var data2 []models.Post
	resp, err = client.R().
		SetContext(ctx).
		SetQueryParam("limit", "5").
		SetResult(&data2).
		Get("/posts")

	if err != nil {
		logger.Error("Request failed", slog.Any("error", err))
		return
	}

	logger.Info("Response data:", slog.Any("first_post", data2[0]))
}

// logRequest logs request details before sending it.
func logRequest(logger *slog.Logger, r *resty.Request, disableLogBody, disableLogHeaders, disableLogQuery bool) {
	var headers map[string][]string
	if !disableLogHeaders {
		headers = make(map[string][]string)
		for k, v := range r.Header {
			headers[k] = v
		}
	}

	queryStr := ""
	if !disableLogQuery && r.RawRequest != nil {
		queryValues := r.RawRequest.URL.Query()
		queryMap := map[string][]string(queryValues)
		qBytes, _ := json.Marshal(queryMap)
		queryStr = string(qBytes)
	}

	bodyStr := ""
	if !disableLogBody && r.Body != nil {
		switch b := r.Body.(type) {
		case string:
			bodyStr = b
		case []byte:
			bodyStr = string(b)
		case io.Reader:
			// If needed, re-read the reader here, but that can affect the request.
			// For simplicity, we won't handle this case deeply here.
		}
	}

	logger.Info("Outgoing request",
		slog.String("method", r.Method),
		slog.String("url", r.URL),
		slog.String("query", queryStr),
		slog.Any("headers", headers),
		slog.String("body", bodyStr),
	)
}

// logResponse logs response details after receiving it.
func logResponse(logger *slog.Logger, resp *resty.Response, disableLogBody, disableLogHeaders bool) {
	var headers map[string][]string
	if !disableLogHeaders {
		headers = make(map[string][]string)
		for k, v := range resp.Header() {
			headers[k] = v
		}
	}

	var bodyStr string
	if !disableLogBody && resp.Body() != nil {
		bodyStr = string(resp.Body())
	}

	logger.Info("Incoming response",
		slog.Int("status_code", resp.StatusCode()),
		slog.Any("headers", headers),
		slog.String("body", bodyStr),
	)
}
