package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mewis.me/meta.go/internal/runtime/server"
)

type bootstrapInfo struct {
	Endpoint string `json:"endpoint"`
	Token    string `json:"token"`
	Protocol int    `json:"protocol"`
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	flags := flag.NewFlagSet("meta-runtime", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	listenAddress := flags.String("listen", server.DefaultListenAddress, "loopback TCP listen address")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	listener, err := server.Listen(*listenAddress)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer listener.Close()
	runtimeServer, err := server.New(server.Config{Token: os.Getenv("META_RUNTIME_TOKEN"), Capabilities: []string{"e2ee.media", "e2ee.text", "facebook.marketplace", "facebook.notifications", "facebook.posts", "facebook.profile", "facebook.search", "facebook.social", "messenger.controls", "messenger.media", "messenger.notes", "messenger.polls", "messenger.search", "messenger.send", "messenger.threads", "runtime.info", "session.events", "session.health", "session.lifecycle"}})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := json.NewEncoder(os.Stdout).Encode(bootstrapInfo{Endpoint: listener.Addr().String(), Token: runtimeServer.Token(), Protocol: server.ProtocolMajor}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- runtimeServer.Serve(listener) }()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serveErr:
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := runtimeServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
}
