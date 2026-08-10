package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(baseURL string, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 300 * time.Second},
	}
}

func (c *Client) post(path string, payload any, out any, timeout time.Duration) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", c.apiKey)

	client := c.http
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s returned %d: %s", path, response.StatusCode, truncate(string(raw), 200))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func (c *Client) Register(request RegisterRequest) (RegisterResponse, error) {
	response := RegisterResponse{Enabled: true}
	err := c.post("/api/worker/register", request, &response, 30*time.Second)
	return response, err
}

func (c *Client) Heartbeat(request HeartbeatRequest) (HeartbeatResponse, error) {
	response := HeartbeatResponse{}
	err := c.post("/api/worker/heartbeat", request, &response, 30*time.Second)
	return response, err
}

func (c *Client) Claim(request ClaimRequest) (ClaimResponse, error) {
	response := ClaimResponse{}
	err := c.post("/api/worker/claim", request, &response, 30*time.Second)
	return response, err
}

func (c *Client) Progress(shardID string, request ProgressRequest) (bool, error) {
	response := ProgressResponse{}
	err := c.post(fmt.Sprintf("/api/worker/shards/%s/progress", shardID), request, &response, 30*time.Second)
	if err != nil {
		return false, err
	}
	return response.Cancel, nil
}

func (c *Client) PostResult(shardID string, request ResultRequest) error {
	return c.post(fmt.Sprintf("/api/worker/shards/%s/result", shardID), request, nil, 300*time.Second)
}

func (c *Client) Fail(shardID string, request FailRequest) error {
	return c.post(fmt.Sprintf("/api/worker/shards/%s/fail", shardID), request, nil, 30*time.Second)
}

func truncate(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}
