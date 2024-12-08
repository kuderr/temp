package httpclient2

import (
	"context"
	"fmt"
	"httpclient/models"
	"net/http"
	"time"
)

func Example() {
	// Create a client with multiple configurations
	client := New(
		WithBaseURL("https://jsonplaceholder.typicode.com"),
		WithTimeout(45*time.Second),
		WithBearerToken("your-access-token"),
		WithDefaultHeaders(map[string]string{
			"Content-Type": "application/json",
		}),
	)

	// Prepare a GET request
	req := Request{
		Method: http.MethodGet,
		Path:   "/posts",
		Query: map[string]string{
			"limit": "5",
		},
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	// Send the request
	resp, err := client.Do(context.Background(), req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}

	// Parse JSON response
	var users []models.Post
	if err := ReadJSONResponse(resp, &users); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		return
	}

	// Alternative client with API Key authentication
	New(
		WithBaseURL("https://another-api.com"),
		WithAPIKey("your-api-key", "header"),
	)
}
