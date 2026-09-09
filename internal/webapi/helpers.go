package webapi

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const AntiHijackPrefix = "for (;;);"

type RequestCounter struct{ value atomic.Uint64 }

func (c *RequestCounter) NextBase36() string { return strconv.FormatUint(c.value.Add(1), 36) }

func ParseCookieString(value string) map[string]string {
	result := map[string]string{}
	for _, part := range strings.Split(value, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(val)
	}
	return result
}

func DecodeJSONObject(data []byte, stripPrefix bool, out any) error {
	data = bytes.TrimSpace(data)
	if stripPrefix && bytes.HasPrefix(data, []byte(AntiHijackPrefix)) {
		data = bytes.TrimSpace(data[len(AntiHijackPrefix):])
	}
	if len(data) == 0 || data[0] != '{' {
		return errors.New("facebook response is not a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

type FormSession struct {
	FBID           string
	DTSG           string
	Jazoest        string
	ClientRevision string
}

func BaseForm(session FormSession, counter *RequestCounter) map[string]string {
	return map[string]string{
		"fb_dtsg": session.DTSG,
		"jazoest": session.Jazoest,
		"__a":     "1",
		"__user":  session.FBID,
		"__req":   counter.NextBase36(),
		"__rev":   session.ClientRevision,
		"av":      session.FBID,
	}
}

func NewSessionID() (uint64, error) {
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(1<<53))
	if err != nil {
		return 0, err
	}
	return value.Uint64() + 1, nil
}

func NewThreadingID(now time.Time) (string, error) {
	random, err := cryptorand.Int(cryptorand.Reader, new(big.Int).Lsh(big.NewInt(1), 22))
	if err != nil {
		return "", err
	}
	return strconv.FormatUint((uint64(now.UnixMilli())<<22)|random.Uint64(), 10), nil
}

type Client struct {
	HTTP         *http.Client
	MaxBodyBytes int64
}

func NewClient(client *http.Client) *Client {
	if client == nil {
		client = &http.Client{}
	}
	clone := *client
	if clone.Timeout == 0 {
		clone.Timeout = 30 * time.Second
	}
	if clone.Jar == nil {
		clone.Jar, _ = cookiejar.New(nil)
	}
	if clone.CheckRedirect == nil {
		clone.CheckRedirect = SafeRedirectPolicy
	}
	return &Client{HTTP: &clone, MaxBodyBytes: 16 << 20}
}

func SafeRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("too many redirects")
	}
	if len(via) > 0 && via[len(via)-1].URL.Scheme == "https" && req.URL.Scheme == "http" {
		return errors.New("refusing HTTPS to HTTP redirect")
	}
	return nil
}

func NewRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	if ctx == nil {
		return nil, errors.New("nil context")
	}
	return http.NewRequestWithContext(ctx, method, rawURL, body)
}

func NewFormRequest(ctx context.Context, method, rawURL string, form url.Values) (*http.Request, error) {
	req, err := NewRequest(ctx, method, rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func ApplyBrowserHeaders(req *http.Request, origin, referer string) {
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
}

func IsRetryable(err error, statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooEarly, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	var networkError net.Error
	return err != nil && errors.As(err, &networkError) && (networkError.Timeout() || networkError.Temporary())
}

func (c *Client) Do(req *http.Request) ([]byte, error) {
	response, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	limit := c.MaxBodyBytes
	if limit <= 0 {
		limit = 16 << 20
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("response body exceeds limit")
	}
	return data, nil
}
