# Library quick start

```go
package main

import (
	"context"
	"log"

	"go.mewis.me/meta.go"
	"go.mewis.me/meta.go/auth"
)

func main() {
	client, err := meta.NewClient(
		meta.WithCookies(auth.Cookies{
			"c_user": "...",
			"xs":     "...",
		}),
		meta.WithE2EE(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}

	result, err := client.Messenger.Send(ctx, meta.SendRequest{
		ThreadID: "1234567890",
		Text:     "hello from meta",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("sent %s", result.MessageID)
}
```

## Events

Use either the event channel:

```go
for event := range client.Events() {
	if event.Kind == meta.EventMessage && event.Message != nil {
		log.Printf("message: %s", event.Message.Text)
	}
}
```

or filtered handlers:

```go
unsubscribe := client.On(meta.EventMessage, func(event meta.Event) {
	// Handler is invoked outside internal locks.
})
defer unsubscribe()
```

## Feature services

After `Connect` succeeds:

- `client.Messenger` exposes regular and E2EE messaging operations;
- `client.Threads` exposes thread queries and mutations;
- `client.Facebook` exposes Facebook profile, social, post, notification and Marketplace operations.

All blocking/network operations accept `context.Context`.
