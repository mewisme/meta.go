package meta

import (
	"context"
	"errors"
	"testing"
)

func TestMessagixBackendTransportContextUsesEngineLifetime(t *testing.T) {
	lifetime, cancelLifetime := context.WithCancel(context.Background())
	defer cancelLifetime()
	request, cancelRequest := context.WithCancel(context.Background())
	b := &messagixBackend{lifetimeCtx: lifetime}
	cancelRequest()

	ctx := b.transportContext()
	if !errors.Is(request.Err(), context.Canceled) {
		t.Fatal("request context was not canceled")
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("transport context followed request lifetime: %v", err)
	}

	cancelLifetime()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("transport context did not follow engine lifetime: %v", ctx.Err())
	}
}

func TestMessagixBackendTransportContextFallsBackToBackground(t *testing.T) {
	ctx := new(messagixBackend).transportContext()
	if ctx == nil || ctx.Err() != nil {
		t.Fatalf("unexpected fallback context: %v", ctx)
	}
}
