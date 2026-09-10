package session

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	metago "go.mewis.me/meta.go"
	"go.mewis.me/meta.go/auth"
	"go.mewis.me/meta.go/facebook"
	"go.mewis.me/meta.go/messenger"
	"go.mewis.me/meta.go/model"
	"go.mewis.me/meta.go/thread"
)

var (
	ErrNotFound = errors.New("runtime session not found")
	ErrClosed   = errors.New("runtime session closed")
)

type Config struct {
	Cookies     auth.Cookies
	E2EE        bool
	EventBuffer int
	Timeout     time.Duration
}

type Client interface {
	Connect(context.Context) error
	Close() error
	Health() model.HealthSnapshot
	Account() model.User
	MessengerService() *messenger.Service
	ThreadService() *thread.Service
	FacebookService() *facebook.Service
	Subscribe(func(model.Event)) func()
}

type Factory func(Config) (Client, error)

type managedClient struct{ client *metago.Client }

func newManagedClient(config Config) (Client, error) {
	opts := []metago.Option{metago.WithCookies(config.Cookies), metago.WithE2EE(config.E2EE)}
	if config.EventBuffer > 0 {
		opts = append(opts, metago.WithEventBuffer(config.EventBuffer))
	}
	if config.Timeout > 0 {
		opts = append(opts, metago.WithTimeout(config.Timeout))
	}
	client, err := metago.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &managedClient{client: client}, nil
}

func (c *managedClient) Connect(ctx context.Context) error    { return c.client.Connect(ctx) }
func (c *managedClient) Close() error                         { return c.client.Close() }
func (c *managedClient) Health() model.HealthSnapshot         { return c.client.Health() }
func (c *managedClient) Account() model.User                  { return c.client.Account() }
func (c *managedClient) MessengerService() *messenger.Service { return c.client.Messenger }
func (c *managedClient) ThreadService() *thread.Service       { return c.client.Threads }
func (c *managedClient) FacebookService() *facebook.Service   { return c.client.Facebook }
func (c *managedClient) Subscribe(handler func(model.Event)) func() {
	return c.client.On("", handler)
}

type Session struct {
	id                string
	client            Client
	closed            atomic.Bool
	eventMu           sync.Mutex
	subscribers       map[uint64]*subscriber
	nextSubscriberID  uint64
	sequence          uint64
	subscriberDropped atomic.Uint64
	unsubscribe       func()
}

func (s *Session) ID() string { return s.id }

func (s *Session) Connect(ctx context.Context) (model.User, error) {
	if s == nil || s.closed.Load() {
		return model.User{}, ErrClosed
	}
	if err := s.client.Connect(ctx); err != nil {
		return model.User{}, err
	}
	return s.client.Account(), nil
}

func (s *Session) Health() model.HealthSnapshot {
	if s == nil || s.closed.Load() {
		return model.HealthSnapshot{Regular: model.ConnectionDisconnected, E2EE: model.ConnectionDisconnected}
	}
	return s.client.Health()
}

func (s *Session) Client() Client { return s.client }

func (s *Session) Close() error {
	if s == nil || !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	if s.unsubscribe != nil {
		s.unsubscribe()
	}
	s.closeSubscribers()
	return s.client.Close()
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	factory  Factory
}

func NewManager(factory Factory) *Manager {
	if factory == nil {
		factory = newManagedClient
	}
	return &Manager{sessions: make(map[string]*Session), factory: factory}
}

func (m *Manager) Create(config Config) (*Session, error) {
	client, err := m.factory(config)
	if err != nil {
		return nil, err
	}
	s := &Session{id: uuid.NewString(), client: client, subscribers: make(map[uint64]*subscriber)}
	s.unsubscribe = client.Subscribe(s.publish)
	if s.unsubscribe == nil {
		s.unsubscribe = func() {}
	}
	m.mu.Lock()
	m.sessions[s.id] = s
	m.mu.Unlock()
	return s, nil
}

func (m *Manager) Get(id string) (*Session, error) {
	m.mu.RLock()
	s := m.sessions[id]
	m.mu.RUnlock()
	if s == nil {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *Manager) Close(id string) error {
	m.mu.Lock()
	s := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if s == nil {
		return ErrNotFound
	}
	return s.Close()
}

func (m *Manager) CloseAll() error {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))
	for id, s := range m.sessions {
		sessions = append(sessions, s)
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	var result error
	for _, s := range sessions {
		result = errors.Join(result, s.Close())
	}
	return result
}

func (m *Manager) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
