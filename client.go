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
	"go.mewis.me/fbgo/internal/logging"
	"go.mewis.me/fbgo/internal/meta"
	"go.mewis.me/fbgo/messenger"
	"go.mewis.me/fbgo/model"
	"go.mewis.me/fbgo/storage"
)

// Client is the root fbgo client. Feature services are attached as the
// implementation phases add their transport capabilities.
type Client struct {
	Messenger *messenger.Service

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
	engine    *meta.Engine
	account   model.User
	closed    bool
}

// New creates a client with production-safe defaults and applies opts in order.
func New(opts ...Option) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{httpClient: &http.Client{}, logger: logging.RedactLogger(slog.Default()), timeout: 30 * time.Second, eventBuffer: 100, ctx: ctx, cancel: cancel}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	client.httpClient.Timeout = client.timeout
	return client
}

func NewClient(opts ...Option) (*Client, error) { return New(opts...), nil }

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
		return err
	}
	engine, err := meta.New(c.ctx, meta.Config{Cookies: map[string]string(cookies), Platform: "facebook", Logger: zerolog.Nop(), EventBuffer: c.eventBuffer, DeviceStore: deviceStore})
	if err != nil {
		return err
	}
	account, err := engine.Connect(ctx)
	if err != nil {
		engine.Close()
		return err
	}
	if c.e2ee {
		if err := engine.ConnectE2EE(ctx, account.ID); err != nil {
			engine.Close()
			return err
		}
	}
	c.mu.Lock()
	c.engine = engine
	c.account = model.User{ID: account.ID, Name: account.Name, Username: account.Username}
	c.Messenger = messenger.NewService(engine)
	c.mu.Unlock()
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
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	engine := c.engine
	c.engine = nil
	c.Messenger = nil
	c.mu.Unlock()
	c.cancel()
	if engine != nil {
		engine.Close()
	}
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

func (c *Client) Account() User {
	if c == nil {
		return User{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.account
}
