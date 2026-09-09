package webapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultMediaLimit int64 = 100 << 20

var ErrUnsafeMediaURL = errors.New("unsafe media URL")

type MediaDownload struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}

func ValidateMediaURL(ctx context.Context, resolver *net.Resolver, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: invalid URL", ErrUnsafeMediaURL)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return fmt.Errorf("%w: unsupported scheme", ErrUnsafeMediaURL)
	}
	if parsed.User != nil {
		return fmt.Errorf("%w: embedded credentials are not allowed", ErrUnsafeMediaURL)
	}
	host := parsed.Hostname()
	if host == "" || strings.EqualFold(host, "localhost") {
		return fmt.Errorf("%w: invalid host", ErrUnsafeMediaURL)
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return fmt.Errorf("%w: private address", ErrUnsafeMediaURL)
		}
		return nil
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addresses, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve media host: %w", err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("%w: host has no addresses", ErrUnsafeMediaURL)
	}
	for _, address := range addresses {
		if !isPublicIP(address.IP) {
			return fmt.Errorf("%w: host resolves to private address", ErrUnsafeMediaURL)
		}
	}
	return nil
}

func DownloadMedia(ctx context.Context, rawURL string, maxBytes int64) (*MediaDownload, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMediaLimit
	}
	resolver := net.DefaultResolver
	if err := ValidateMediaURL(ctx, resolver, rawURL); err != nil {
		return nil, err
	}
	client := newMediaHTTPClient(resolver)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		return ValidateMediaURL(req.Context(), resolver, req.URL.String())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("media download HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxBytes {
		resp.Body.Close()
		return nil, fmt.Errorf("media exceeds %d-byte limit", maxBytes)
	}
	return &MediaDownload{Body: &limitReadCloser{reader: io.LimitReader(resp.Body, maxBytes+1), closer: resp.Body, remaining: maxBytes}, ContentType: resp.Header.Get("Content-Type"), ContentLength: resp.ContentLength}, nil
}

func newMediaHTTPClient(resolver *net.Resolver) *http.Client {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Secure media fetches intentionally ignore HTTP(S)_PROXY. A proxy would
	// resolve the hostname outside this process and bypass the pinned-IP SSRF
	// check performed by DialContext.
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		addresses, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, candidate := range addresses {
			if !isPublicIP(candidate.IP) {
				return nil, fmt.Errorf("%w: resolved private address", ErrUnsafeMediaURL)
			}
		}
		var lastErr error
		for _, candidate := range addresses {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = errors.New("no resolved addresses")
		}
		return nil, lastErr
	}
	return &http.Client{Transport: transport}
}

type limitReadCloser struct {
	reader    io.Reader
	closer    io.Closer
	remaining int64
	exceeded  bool
}

func (r *limitReadCloser) Read(buffer []byte) (int, error) {
	if r.exceeded {
		return 0, fmt.Errorf("media exceeds size limit")
	}
	if int64(len(buffer)) > r.remaining+1 {
		buffer = buffer[:r.remaining+1]
	}
	n, err := r.reader.Read(buffer)
	if int64(n) > r.remaining {
		r.exceeded = true
		if r.remaining <= 0 {
			return 0, fmt.Errorf("media exceeds size limit")
		}
		allowed := int(r.remaining)
		r.remaining = 0
		return allowed, fmt.Errorf("media exceeds size limit")
	}
	r.remaining -= int64(n)
	return n, err
}

func (r *limitReadCloser) Close() error { return r.closer.Close() }

func isPublicIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		first, second := ipv4[0], ipv4[1]
		if first == 0 || first == 127 || first >= 224 || first == 169 && second == 254 || first == 100 && second >= 64 && second <= 127 || first == 192 && second == 0 && ipv4[2] == 0 || first == 192 && second == 0 && ipv4[2] == 2 || first == 198 && (second == 18 || second == 19) || first == 198 && second == 51 && ipv4[2] == 100 || first == 203 && second == 0 && ipv4[2] == 113 {
			return false
		}
		return true
	}
	// IPv6 documentation, unique-local and mapped special-use ranges.
	value := ip.String()
	if strings.HasPrefix(value, "2001:db8:") || strings.HasPrefix(value, "fc") || strings.HasPrefix(value, "fd") {
		return false
	}
	return true
}
