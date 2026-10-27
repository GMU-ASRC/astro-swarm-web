package worker

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const resultUploadTimeout = 600 * time.Second

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

type requestBody struct {
	data        []byte
	gzipped     bool
	contentType string
	workerID    string
}

func NewClient(baseURL string, apiKey string) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 32
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Transport: transport},
	}
}

func encodeGzipJSON(payload any) (requestBody, error) {
	var buffer bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buffer, gzip.BestSpeed)
	if err != nil {
		return requestBody{}, err
	}
	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		return requestBody{}, err
	}
	if err := writer.Close(); err != nil {
		return requestBody{}, err
	}
	return requestBody{data: buffer.Bytes(), gzipped: true}, nil
}

func (c *Client) post(path string, payload any, out any, timeout time.Duration) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.send(path, requestBody{data: data}, out, timeout)
}

func (c *Client) send(path string, body requestBody, out any, timeout time.Duration) error {
	request, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body.data))
	if err != nil {
		return err
	}
	contentType := body.contentType
	if contentType == "" {
		contentType = "application/json"
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("X-API-Key", c.apiKey)
	if body.workerID != "" {
		request.Header.Set("X-Worker-ID", body.workerID)
	}
	if body.gzipped {
		request.Header.Set("Content-Encoding", "gzip")
	}

	client := *c.http
	client.Timeout = timeout
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

func (c *Client) Progress(jobID string, request ProgressRequest) (bool, error) {
	response := ProgressResponse{}
	err := c.post(fmt.Sprintf("/api/worker/jobs/%s/progress", jobID), request, &response, 30*time.Second)
	if err != nil {
		return false, err
	}
	return response.Cancel, nil
}

func (c *Client) PostResult(jobID string, body requestBody) (bool, error) {
	response := ResultResponse{Accepted: true}
	err := c.send(fmt.Sprintf("/api/worker/jobs/%s/result", jobID), body, &response, resultUploadTimeout)
	if err != nil {
		return false, err
	}
	return response.Accepted, nil
}

func (c *Client) UploadReplayChunk(jobID string, uploadID string, key string, offset int, workerID string, chunk []byte) (ReplayChunkResponse, error) {
	response := ReplayChunkResponse{}
	path := fmt.Sprintf("/api/worker/jobs/%s/replays/%s/%s?offset=%d", jobID, uploadID, key, offset)
	body := requestBody{data: chunk, contentType: "application/octet-stream", workerID: workerID}
	err := c.send(path, body, &response, 120*time.Second)
	return response, err
}

func (c *Client) Fail(jobID string, request FailRequest) error {
	return c.post(fmt.Sprintf("/api/worker/jobs/%s/fail", jobID), request, nil, 30*time.Second)
}

func truncate(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}
