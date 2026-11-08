package apicompare

import (
	"compress/gzip"
	"io"
	"net/http"
	"time"
)

type Response struct {
	Status      int
	ContentType string
	Body        []byte
	Duration    time.Duration
	Err         error
}

type Client struct {
	http     *http.Client
	adminKey string
}

func NewClient(adminKey string) *Client {
	return &Client{http: &http.Client{Timeout: 5 * time.Minute}, adminKey: adminKey}
}

func (client *Client) Get(url string, admin bool) Response {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Response{Err: err}
	}
	request.Header.Set("Accept-Encoding", "gzip")
	if admin {
		request.Header.Set("X-API-Key", client.adminKey)
	}
	started := time.Now()
	response, err := client.http.Do(request)
	if err != nil {
		return Response{Err: err}
	}
	defer response.Body.Close()
	var reader io.Reader = response.Body
	if response.Header.Get("Content-Encoding") == "gzip" {
		unzipped, err := gzip.NewReader(response.Body)
		if err != nil {
			return Response{Err: err}
		}
		defer unzipped.Close()
		reader = unzipped
	}
	body, err := io.ReadAll(reader)
	return Response{
		Status: response.StatusCode, ContentType: baseContentType(response.Header.Get("Content-Type")),
		Body: body, Duration: time.Since(started), Err: err,
	}
}
