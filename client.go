package fbgo

import (
	"log/slog"
	"net/http"
	"time"
)

// Client is the root fbgo client. Feature services are attached as the
// implementation phases add their transport capabilities.
type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
	timeout    time.Duration
}

// New creates a client with production-safe defaults and applies opts in order.
func New(opts ...Option) *Client {
	client := &Client{httpClient: &http.Client{}, logger: slog.Default(), timeout: 30 * time.Second}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	client.httpClient.Timeout = client.timeout
	return client
}
