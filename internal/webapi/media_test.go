package webapi

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"testing"
)

func TestMediaURLRejectsPrivateAddresses(t *testing.T) {
	for _, value := range []string{"http://127.0.0.1/a", "http://10.0.0.1/a", "http://169.254.169.254/latest", "http://[::1]/a", "file:///tmp/a", "https://user:pass@example.com/a"} {
		if err := ValidateMediaURL(t.Context(), nil, value); err == nil {
			t.Fatalf("expected unsafe URL rejection for %s", value)
		}
	}
}

func TestPublicIPClassification(t *testing.T) {
	if !isPublicIP(net.ParseIP("1.1.1.1")) {
		t.Fatal("public IPv4 was rejected")
	}
	if isPublicIP(net.ParseIP("192.168.1.1")) || isPublicIP(net.ParseIP("198.18.0.1")) || isPublicIP(net.ParseIP("100.64.0.1")) || isPublicIP(net.ParseIP("2001:db8::1")) {
		t.Fatal("special-use address was accepted")
	}
}

func TestMediaHTTPClientDisablesEnvironmentProxy(t *testing.T) {
	client := newMediaHTTPClient(net.DefaultResolver)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("unexpected transport type %T", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("secure media client must not use environment proxies")
	}
}

func TestLimitedMediaReaderRejectsOverflow(t *testing.T) {
	closer := io.NopCloser(bytes.NewReader([]byte("abcdef")))
	reader := &limitReadCloser{reader: io.LimitReader(closer, 6), closer: closer, remaining: 5}
	data, err := io.ReadAll(reader)
	if err == nil || string(data) != "abcde" {
		t.Fatalf("expected bounded data plus overflow error, got %q %v", data, err)
	}
}
