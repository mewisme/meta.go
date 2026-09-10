package meta

import (
	"log/slog"
	"net/http"
	"time"

	"go.mewis.me/meta.go/auth"
	"go.mewis.me/meta.go/internal/logging"
	"go.mewis.me/meta.go/storage"
)

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient supplies the HTTP transport used by meta. Client-level cookie
// and redirect policies are intentionally managed by meta and are not reused.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			clone := *httpClient
			client.httpClient = &clone
		}
	}
}

// WithLogger replaces the structured logger used by meta.
func WithLogger(logger *slog.Logger) Option {
	return func(client *Client) {
		if logger != nil {
			client.logger = logging.RedactLogger(logger)
		}
	}
}

// WithTimeout changes the default network timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(client *Client) { client.timeout = timeout }
}

func WithCookies(cookies auth.Cookies) Option {
	return func(client *Client) { client.cookies = cookies.Clone() }
}

func WithProfile(profile storage.Profile) Option {
	return func(client *Client) { client.profile = profile }
}

func WithSecretStore(store storage.SecretStore) Option {
	return func(client *Client) { client.secrets = store }
}

func WithE2EE(enabled bool) Option {
	return func(client *Client) { client.e2ee = enabled }
}

func WithEventBuffer(size int) Option {
	return func(client *Client) { client.eventBuffer = size }
}
