package httpclient

import (
	"context"
	"encoding/json"
	"httpclient/models"
	"httpclient/utils"
	"net/url"
	"os"
	"time"

	"log/slog"
)

func Example() {
	logger := slog.New(utils.NewPrettyJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	baseURL, _ := url.Parse("https://jsonplaceholder.typicode.com")
	client := NewCommonHTTPClient(ClientConfig{
		BaseURL:           baseURL,
		DefaultHeaders:    map[string]string{"Authorization": "Bearer token"},
		DisableLogBody:    false,
		DisableLogHeaders: false,
		DisableLogQuery:   false,
		MaxRetries:        3,
		RetryBackoff:      1 * time.Second,
		Logger:            logger,
	})

	opts := RequestOptions{
		Method: "GET",
		Path:   "/posts",
		QueryParams: map[string]string{
			"limit": "5",
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(context.Background(), opts)
	if err != nil {
		logger.Error("Request failed:", slog.Any("error", err))
		return
	}
	defer resp.Body.Close()

	var data []models.Post
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		logger.Error("Decode error:", slog.Any("error", err))
		return
	}

	logger.Info("Response data:", slog.Any("first_post", data[0]))
}
