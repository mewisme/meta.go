package webapi

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
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
		result[strings.TrimSpace(key)] = strings.TrimSpace(val)
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
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{HTTP: client, MaxBodyBytes: 16 << 20}
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
