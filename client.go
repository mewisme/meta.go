package meta

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"go.mewis.me/meta.go/auth"
	fberrors "go.mewis.me/meta.go/errors"
	facebookservice "go.mewis.me/meta.go/facebook"
	"go.mewis.me/meta.go/internal/logging"
	"go.mewis.me/meta.go/internal/meta"
	"go.mewis.me/meta.go/messenger"
	"go.mewis.me/meta.go/model"
	"go.mewis.me/meta.go/storage"
	threadservice "go.mewis.me/meta.go/thread"
)

// Client is the root meta client. Feature services are attached as the
// implementation phases add their transport capabilities.
type Client struct {
	Messenger *messenger.Service
	Threads   *threadservice.Service
	Facebook  *facebookservice.Service

	httpClient  *http.Client
	logger      *slog.Logger
	timeout     time.Duration
	cookies     auth.Cookies
	appState    auth.AppState
	credentials auth.Credentials
	authSource  clientAuthSource
	optionErr   error
	profile     storage.Profile
	secrets     storage.SecretStore
	e2ee        bool
	eventBuffer int

	ctx    context.Context
	cancel context.CancelFunc

	mu              sync.RWMutex
	connectMu       sync.Mutex
	handlerMu       sync.RWMutex
	engine          *meta.Engine
	account         model.User
	closed          bool
	nextID          uint64
	handlers        map[uint64]clientHandler
	credentialLogin func(context.Context, auth.Credentials) (auth.Cookies, error)
}

type clientAuthSource uint8

const (
	clientAuthNone clientAuthSource = iota
	clientAuthCookies
	clientAuthAppState
	clientAuthCredentials
)

type clientHandler struct {
	kind    EventKind
	handler Handler
}

// NewClient creates a client with production-safe defaults and applies opts in order.
func NewClient(opts ...Option) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{httpClient: &http.Client{}, logger: logging.RedactLogger(slog.Default()), timeout: 30 * time.Second, eventBuffer: 100, ctx: ctx, cancel: cancel, handlers: map[uint64]clientHandler{}, credentialLogin: func(ctx context.Context, credentials auth.Credentials) (auth.Cookies, error) {
		return auth.NewCredentialLogin().Login(ctx, credentials)
	}}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	if client.optionErr != nil {
		cancel()
		return nil, client.optionErr
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
		return errors.New("nil meta client")
	}
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return errors.New("meta client is closed")
	}
	if c.engine != nil && c.engine.Connected() {
		c.mu.RUnlock()
		return nil
	}
	profile := c.profile
	secrets := c.secrets
	c.mu.RUnlock()
	cookies, err := c.resolveAuth(ctx, profile, secrets)
	if err != nil {
		return err
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
		return errors.New("meta client is closed")
	}
	c.engine = engine
	c.account = model.User{ID: account.ID, Name: account.Name, Username: account.Username}
	if c.authSource != clientAuthNone {
		c.cookies = cookies.Clone()
		c.appState = nil
		c.credentials = auth.Credentials{}
		c.authSource = clientAuthCookies
	}
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
	c.appState = nil
	c.credentials = auth.Credentials{}
	c.authSource = clientAuthNone
	c.optionErr = nil
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
	c.logger.Debug("client closed")
	c.connectMu.Unlock()
	return nil
}

func (c *Client) resolveAuth(ctx context.Context, profile storage.Profile, secrets storage.SecretStore) (auth.Cookies, error) {
	c.mu.RLock()
	source := c.authSource
	cookies := c.cookies.Clone()
	state := c.appState.Clone()
	credentials := c.credentials
	login := c.credentialLogin
	c.mu.RUnlock()

	switch source {
	case clientAuthCookies:
	case clientAuthAppState:
		resolved, err := state.Cookies()
		if err != nil {
			return nil, fmt.Errorf("%w: %v", fberrors.ErrUnauthorized, err)
		}
		cookies = resolved
	case clientAuthCredentials:
		if login == nil {
			return nil, errors.New("credential login is unavailable")
		}
		resolved, err := login(ctx, credentials)
		if err != nil {
			return nil, err
		}
		if err := resolved.ValidateRegular(); err != nil {
			return nil, fmt.Errorf("%w: %v", fberrors.ErrUnauthorized, err)
		}
		cookies = resolved
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return nil, errors.New("meta client is closed")
		}
		if c.authSource == clientAuthCredentials {
			c.cookies = cookies.Clone()
			c.credentials = auth.Credentials{}
			c.authSource = clientAuthCookies
		}
		c.mu.Unlock()
	case clientAuthNone:
		if profile.Name != "" && secrets != nil {
			loaded, err := (auth.ProfileManager{Secrets: secrets}).LoadCookies(ctx, profile.Name)
			if err != nil {
				return nil, fmt.Errorf("%w: load profile cookies: %v", fberrors.ErrUnauthorized, err)
			}
			cookies = loaded
		}
	}
	if len(cookies) == 0 {
		return nil, fmt.Errorf("%w: no authentication configured", fberrors.ErrUnauthorized)
	}
	if err := cookies.ValidateRegular(); err != nil {
		return nil, fmt.Errorf("%w: %v", fberrors.ErrUnauthorized, err)
	}
	return cookies, nil
}

func (c *Client) RefreshAuth(ctx context.Context, source *auth.Source) (auth.AuthSnapshot, error) {
	if c == nil {
		return auth.AuthSnapshot{}, errors.New("nil meta client")
	}
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return auth.AuthSnapshot{}, errors.New("meta client is closed")
	}
	engine, account, profile, secrets := c.engine, c.account, c.profile, c.secrets
	c.mu.RUnlock()
	if engine == nil || !engine.Connected() {
		return auth.AuthSnapshot{}, fberrors.ErrNotConnected
	}
	previous, err := engine.AuthState(ctx)
	if err != nil {
		return auth.AuthSnapshot{}, err
	}
	cookies := auth.Cookies(previous.Cookies).Clone()
	if source != nil {
		cookies, err = c.resolveSource(ctx, *source)
		if err != nil {
			return auth.AuthSnapshot{}, err
		}
		if err := requireSameAccount(account.ID, cookies["c_user"]); err != nil {
			return auth.AuthSnapshot{}, err
		}
		validated, err := (auth.SessionValidator{}).Validate(ctx, cookies)
		if err != nil {
			return auth.AuthSnapshot{}, err
		}
		if validated.FBID != account.ID {
			return auth.AuthSnapshot{}, fmt.Errorf("%w: authenticated account differs from connected account", fberrors.ErrAccountMismatch)
		}
	}
	state, err := engine.RefreshAuth(ctx, map[string]string(cookies))
	if err != nil {
		return auth.AuthSnapshot{}, err
	}
	if state.FBID != account.ID {
		_, rollbackErr := engine.RefreshAuth(context.WithoutCancel(ctx), previous.Cookies)
		return auth.AuthSnapshot{}, errors.Join(fmt.Errorf("%w: refreshed account differs from connected account", fberrors.ErrAccountMismatch), rollbackErr)
	}
	refreshed := auth.Cookies(state.Cookies).Clone()
	snapshot := authSnapshot(state, account)
	if profile.Name != "" && secrets != nil {
		if err := (auth.ProfileManager{Secrets: secrets}).SaveAuthSnapshot(ctx, profile.Name, snapshot); err != nil {
			_, rollbackErr := engine.RefreshAuth(context.WithoutCancel(ctx), previous.Cookies)
			return auth.AuthSnapshot{}, errors.Join(fmt.Errorf("persist refreshed auth state: %w", err), rollbackErr)
		}
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return auth.AuthSnapshot{}, errors.New("meta client is closed")
	}
	c.cookies = refreshed.Clone()
	c.appState = nil
	c.credentials = auth.Credentials{}
	c.authSource = clientAuthCookies
	c.mu.Unlock()
	return snapshot, nil
}

func (c *Client) AuthSnapshot(ctx context.Context) (auth.AuthSnapshot, error) {
	if c == nil {
		return auth.AuthSnapshot{}, errors.New("nil meta client")
	}
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return auth.AuthSnapshot{}, errors.New("meta client is closed")
	}
	engine, account := c.engine, c.account
	c.mu.RUnlock()
	if engine == nil || !engine.Connected() {
		return auth.AuthSnapshot{}, fberrors.ErrNotConnected
	}
	state, err := engine.AuthState(ctx)
	if err != nil {
		return auth.AuthSnapshot{}, err
	}
	if state.FBID != account.ID {
		return auth.AuthSnapshot{}, fmt.Errorf("%w: auth state differs from connected account", fberrors.ErrAccountMismatch)
	}
	cookies := auth.Cookies(state.Cookies).Clone()
	c.mu.Lock()
	if !c.closed {
		c.cookies = cookies.Clone()
		c.appState = nil
		c.credentials = auth.Credentials{}
		c.authSource = clientAuthCookies
	}
	c.mu.Unlock()
	return authSnapshot(state, account), nil
}

func (c *Client) resolveSource(ctx context.Context, source auth.Source) (auth.Cookies, error) {
	if err := source.Validate(); err != nil {
		return nil, err
	}
	switch {
	case source.Cookies != nil:
		return source.Cookies.Clone(), nil
	case source.AppState != nil:
		return source.AppState.Cookies()
	default:
		c.mu.RLock()
		login := c.credentialLogin
		c.mu.RUnlock()
		if login == nil {
			return nil, errors.New("credential login is unavailable")
		}
		cookies, err := login(ctx, *source.Credentials)
		if err != nil {
			return nil, err
		}
		if err := cookies.ValidateRegular(); err != nil {
			return nil, fmt.Errorf("%w: %v", fberrors.ErrUnauthorized, err)
		}
		return cookies, nil
	}
}

func requireSameAccount(accountID model.ID, cookieUserID string) error {
	if accountID.Empty() || cookieUserID == "" || accountID.String() != cookieUserID {
		return fmt.Errorf("%w: authentication source belongs to another account", fberrors.ErrAccountMismatch)
	}
	return nil
}

func authSnapshot(state meta.AuthState, account model.User) auth.AuthSnapshot {
	cookies := auth.Cookies(state.Cookies).Clone()
	return auth.AuthSnapshot{Cookies: cookies, AppState: cookies.AppState(), Session: auth.Session{Cookies: cookies.Clone(), FBID: state.FBID, Name: account.Name, Username: account.Username, DTSG: state.DTSG, Jazoest: state.Jazoest, LSD: state.LSD, SessionID: state.SessionID, ClientRevision: state.ClientRevision, BootstrappedAt: time.Now()}}
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
