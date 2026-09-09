package fbgo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.mewis.me/fbgo/auth"
	fberrors "go.mewis.me/fbgo/errors"
	facebookservice "go.mewis.me/fbgo/facebook"
	"go.mewis.me/fbgo/internal/logging"
	"go.mewis.me/fbgo/internal/meta"
	"go.mewis.me/fbgo/messenger"
	"go.mewis.me/fbgo/model"
	"go.mewis.me/fbgo/storage"
	threadservice "go.mewis.me/fbgo/thread"
)

// Client is the root fbgo client. Feature services are attached as the
// implementation phases add their transport capabilities.
type Client struct {
	Messenger *messenger.Service
	Threads   *threadservice.Service
	Facebook  *facebookservice.Service

	httpClient  *http.Client
	logger      *slog.Logger
	timeout     time.Duration
	cookies     auth.Cookies
	profile     storage.Profile
	secrets     storage.SecretStore
	e2ee        bool
	eventBuffer int

	ctx    context.Context
	cancel context.CancelFunc

	mu        sync.RWMutex
	connectMu sync.Mutex
	handlerMu sync.RWMutex
	engine    *meta.Engine
	account   model.User
	closed    bool
	nextID    uint64
	handlers  map[uint64]clientHandler
}

type clientHandler struct {
	kind    EventKind
	handler Handler
}

// NewClient creates a client with production-safe defaults and applies opts in order.
func NewClient(opts ...Option) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{httpClient: &http.Client{}, logger: logging.RedactLogger(slog.Default()), timeout: 30 * time.Second, eventBuffer: 100, ctx: ctx, cancel: cancel, handlers: map[uint64]clientHandler{}}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	if client.timeout <= 0 {
		cancel()
		return nil, fmt.Errorf("%w: timeout must be positive", fberrors.ErrInvalidInput)
	}
	if client.eventBuffer < 1 {
		cancel()
		return nil, fmt.Errorf("%w: event buffer must be positive", fberrors.ErrInvalidInput)
	}
	client.httpClient.Timeout = client.timeout
	return client, nil
}

func (c *Client) Connect(ctx context.Context) error {
	if c == nil {
		return errors.New("nil fbgo client")
	}
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return errors.New("fbgo client is closed")
	}
	if c.engine != nil && c.engine.Connected() {
		c.mu.RUnlock()
		return nil
	}
	cookies := c.cookies.Clone()
	profile := c.profile
	secrets := c.secrets
	c.mu.RUnlock()
	if len(cookies) == 0 && profile.Name != "" && secrets != nil {
		loaded, err := (auth.ProfileManager{Secrets: secrets}).LoadCookies(ctx, profile.Name)
		if err != nil {
			return fmt.Errorf("%w: load profile cookies: %v", fberrors.ErrUnauthorized, err)
		}
		cookies = loaded
	}
	if len(cookies) == 0 {
		return fmt.Errorf("%w: no cookies configured", fberrors.ErrUnauthorized)
	}
	if err := cookies.ValidateRegular(); err != nil {
		return fmt.Errorf("%w: %v", fberrors.ErrUnauthorized, err)
	}
	deviceStore, err := c.deviceStore(ctx, profile, secrets)
	if err != nil {
		c.logger.WarnContext(ctx, "connect failed", "category", string(fberrors.Classify(err)))
		return err
	}
	c.logger.DebugContext(ctx, "connecting", "e2ee", c.e2ee)
	engine, err := meta.New(c.ctx, meta.Config{Cookies: map[string]string(cookies), Platform: "facebook", Logger: zerolog.Nop(), EventBuffer: c.eventBuffer, DeviceStore: deviceStore, HTTPClient: c.httpClient, Timeout: c.timeout})
	if err != nil {
		c.logger.WarnContext(ctx, "connect failed", "category", string(fberrors.Classify(err)))
		return err
	}
	engine.On(c.dispatchEvent)
	account, err := engine.Connect(ctx)
	if err != nil {
		engine.Close()
		c.logger.WarnContext(ctx, "connect failed", "category", string(fberrors.Classify(err)))
		return err
	}
	if c.e2ee {
		if err := engine.ConnectE2EE(ctx, account.ID); err != nil {
			engine.Close()
			c.logger.WarnContext(ctx, "E2EE connect failed", "category", string(fberrors.Classify(err)))
			return err
		}
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		engine.Close()
		return errors.New("fbgo client is closed")
	}
	c.engine = engine
	c.account = model.User{ID: account.ID, Name: account.Name, Username: account.Username}
	c.Messenger = messenger.NewService(engine)
	c.Threads = threadservice.NewService(engine)
	c.Facebook = facebookservice.NewService(engine)
	c.mu.Unlock()
	c.logger.DebugContext(ctx, "connected", "e2ee", c.e2ee)
	return nil
}

func (c *Client) deviceStore(ctx context.Context, profile storage.Profile, secrets storage.SecretStore) (*meta.DeviceStore, error) {
	if !c.e2ee || profile.Name == "" || secrets == nil {
		return nil, nil
	}
	const stateKey = "e2ee_state"
	data, err := secrets.Get(ctx, profile.Name, stateKey)
	var deviceStore *meta.DeviceStore
	switch {
	case err == nil:
		deviceStore, err = meta.LoadDeviceStore(data)
		if err != nil {
			return nil, fmt.Errorf("load E2EE device state: %w", err)
		}
	case errors.Is(err, storage.ErrNotFound):
		deviceStore, err = meta.NewMemoryDeviceStore()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("load E2EE device state: %w", err)
	}
	deviceStore.WithPersistence(func(ctx context.Context, data []byte) error { return secrets.Put(ctx, profile.Name, stateKey, data) })
	return deviceStore, nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.cancel()
	engine := c.engine
	c.engine = nil
	c.cookies = nil
	c.profile = storage.Profile{}
	c.secrets = nil
	c.account = model.User{}
	c.Messenger = nil
	c.Threads = nil
	c.Facebook = nil
	c.mu.Unlock()
	c.handlerMu.Lock()
	clear(c.handlers)
	c.handlerMu.Unlock()
	if engine != nil {
		engine.Close()
	}
	// Wait for a concurrent Connect to observe cancellation and finish before
	// returning. Cancellation happens before taking this lock to avoid making
	// Close wait on a network operation it could have interrupted.
	c.connectMu.Lock()
	c.connectMu.Unlock()
	c.logger.Debug("client closed")
	return nil
}

func (c *Client) Health() HealthSnapshot {
	if c == nil {
		return HealthSnapshot{Regular: model.ConnectionDisconnected, E2EE: model.ConnectionDisconnected}
	}
	c.mu.RLock()
	engine := c.engine
	c.mu.RUnlock()
	if engine == nil {
		return HealthSnapshot{Regular: model.ConnectionDisconnected, E2EE: model.ConnectionDisconnected}
	}
	return engine.Health()
}

func (c *Client) Events() <-chan Event {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	engine := c.engine
	c.mu.RUnlock()
	if engine == nil {
		return nil
	}
	return engine.Events()
}

func (c *Client) On(kind EventKind, handler Handler) UnsubscribeFunc {
	if c == nil || handler == nil {
		return func() {}
	}
	c.mu.RLock()
	closed := c.closed
	c.mu.RUnlock()
	if closed {
		return func() {}
	}
	c.handlerMu.Lock()
	c.nextID++
	id := c.nextID
	c.handlers[id] = clientHandler{kind: kind, handler: handler}
	c.handlerMu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			c.handlerMu.Lock()
			delete(c.handlers, id)
			c.handlerMu.Unlock()
		})
	}
}

func (c *Client) dispatchEvent(event Event) {
	c.handlerMu.RLock()
	handlers := make([]Handler, 0, len(c.handlers))
	for _, item := range c.handlers {
		if item.kind == "" || item.kind == event.Kind {
			handlers = append(handlers, item.handler)
		}
	}
	c.handlerMu.RUnlock()
	for _, handler := range handlers {
		handler(event)
	}
}

func (c *Client) Account() User {
	if c == nil {
		return User{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.account
}
