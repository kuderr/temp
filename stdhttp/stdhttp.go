package stdhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"httpclient/models"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

func Example() {
	// Basic GET Request
	basicGetRequest()

	// GET Request with Query Parameters
	getRequestWithQueryParams()

	// POST Request with JSON Body
	postJSONRequest()

	// Request with Custom Headers
	requestWithCustomHeaders()

	// Request with Basic Authentication
	basicAuthRequest()

	// Request with Bearer Token
	bearerTokenRequest()
}

// Basic GET request
func basicGetRequest() {
	resp, err := http.Get("https://jsonplaceholder.typicode.com/posts")
	if err != nil {
		log.Printf("GET request error: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Reading response error: %v", err)
		return
	}

	fmt.Printf("Basic GET Response: %s\n", string(body))
}

// GET request with query parameters
func getRequestWithQueryParams() {
	// Create a URL with query parameters
	baseURL, err := url.Parse("https://jsonplaceholder.typicode.com/posts")
	if err != nil {
		log.Printf("URL parsing error: %v", err)
		return
	}

	// Add query parameters
	params := url.Values{}
	params.Add("page", "1")
	params.Add("limit", "10")
	baseURL.RawQuery = params.Encode()

	resp, err := http.Get(baseURL.String())
	if err != nil {
		log.Printf("GET request with params error: %v", err)
		return
	}
	defer resp.Body.Close()

	var users []models.Post
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		log.Printf("JSON decoding error: %v", err)
		return
	}

	fmt.Printf("Users from query: %+v\n", users)
}

// POST request with JSON body
func postJSONRequest() {
	// Create a new user
	newUser := models.Post{
		UserID: 1,
		ID:     101,
		Title:  "johndoe",
		Body:   "john@example.com",
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(newUser)
	if err != nil {
		log.Printf("JSON marshaling error: %v", err)
		return
	}

	// Create POST request
	resp, err := http.Post(
		"https://jsonplaceholder.typicode.com/posts",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Printf("POST request error: %v", err)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Reading response error: %v", err)
		return
	}

	fmt.Printf("POST Response: %s\n", string(body))
}

// Request with custom headers
func requestWithCustomHeaders() {
	// Create a custom HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create a new request
	req, err := http.NewRequest("GET", "https://jsonplaceholder.typicode.com/posts", nil)
	if err != nil {
		log.Printf("Creating request error: %v", err)
		return
	}

	// Set custom headers
	req.Header.Set("X-Custom-Header", "CustomValue")
	req.Header.Set("Accept", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Sending request error: %v", err)
		return
	}
	defer resp.Body.Close()

	// Print response headers
	fmt.Println("Response Headers:")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}
}

// Basic Authentication request
func basicAuthRequest() {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://jsonplaceholder.typicode.com/posts", nil)
	if err != nil {
		log.Printf("Creating request error: %v", err)
		return
	}

	// Set Basic Authentication
	req.SetBasicAuth("username", "password")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Sending request error: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Reading response error: %v", err)
		return
	}

	fmt.Printf("Basic Auth Response: %s\n", string(body))
}

// Bearer Token Authentication request
func bearerTokenRequest() {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://jsonplaceholder.typicode.com/posts", nil)
	if err != nil {
		log.Printf("Creating request error: %v", err)
		return
	}

	// Set Bearer Token
	req.Header.Set("Authorization", "Bearer your_access_token_here")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Sending request error: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Reading response error: %v", err)
		return
	}

	fmt.Printf("Bearer Token Response: %s\n", string(body))
}
