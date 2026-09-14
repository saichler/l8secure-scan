package mocks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client handles API communication with the secscan-web server. Mirrors
// ../l8alarms/go/tests/mocks/client.go's real, generic pattern -- a
// standalone external process with no vnic of its own, so it can only
// reach whatever HTTP routes the web server exposes (same limitation
// noted there: any service with no HTTP route, none in this project,
// would need the in-process test harness instead).
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: baseURL, client: httpClient}
}

// BaseURL returns the server address this client was constructed with --
// used by go/tests to build a second Client against the same in-process
// test web server (e.g. to authenticate as a different user).
func (c *Client) BaseURL() string {
	return c.baseURL
}

func L8QueryText(queryText string) string {
	q := map[string]interface{}{"text": queryText}
	data, _ := json.Marshal(q)
	return string(data)
}

func (c *Client) Authenticate(user, password string) error {
	authData := map[string]string{"user": user, "pass": password}
	body, _ := json.Marshal(authData)

	resp, err := c.client.Post(c.baseURL+"/auth", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var authResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	token, ok := authResp["token"].(string)
	if !ok {
		return fmt.Errorf("token not found in auth response")
	}
	c.token = token
	return nil
}

func (c *Client) Post(endpoint string, data interface{}) (string, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return string(respBody), fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}

func (c *Client) Get(endpoint string, queryJSON string) (string, error) {
	fullURL := c.baseURL + endpoint + "?body=" + url.QueryEscape(queryJSON)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return string(respBody), fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}
