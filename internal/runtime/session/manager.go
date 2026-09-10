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
	"go.mewis.me/meta.go/model"
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

type Messenger interface {
	Send(context.Context, model.SendRequest) (model.SendResult, error)
	Forward(context.Context, model.ID, model.ID) (model.SendResult, error)
	ShareContact(context.Context, model.ID, model.ID, string) error
	React(context.Context, model.ID, model.ID, string) error
	Edit(context.Context, model.ID, string) error
	Unsend(context.Context, model.ID) error
	Typing(context.Context, model.ID, bool, bool, int64) error
	Read(context.Context, model.ID, time.Time) error
	MessageRequests(context.Context) ([]model.MessageRequest, error)
	Themes(context.Context) ([]model.Theme, error)
	FindTheme(context.Context, string) (*model.Theme, error)
	SetTheme(context.Context, model.ID, model.ID) error
	CurrentNote(context.Context) (*model.Note, error)
	CreateNote(context.Context, string, string) (*model.Note, error)
	DeleteNote(context.Context, model.ID) error
	RecreateNote(context.Context, model.ID, string, string) (*model.Note, error)
	SetRestricted(context.Context, model.ID, bool) error
	SetMessageBlocked(context.Context, model.ID, bool) error
}

type Threads interface {
	List(context.Context, int) (model.ThreadList, error)
	Get(context.Context, model.ID) (*model.Thread, error)
	CreatePoll(context.Context, model.ID, string, []string) error
	VotePoll(context.Context, model.ID, model.ID, []model.ID) error
	PinnedMessages(context.Context, model.ID) ([]model.PinnedMessage, error)
	PollDetails(context.Context, model.ID) (*model.PollDetails, error)
	SearchMessages(context.Context, model.MessageSearchRequest) (*model.MessageSearchPage, error)
	Mute(context.Context, model.ID, time.Duration) error
	MuteCalls(context.Context, model.ID, time.Duration) error
	SetApprovalMode(context.Context, model.ID, bool) error
	SetArchived(context.Context, model.ID, bool) error
	PinMessage(context.Context, model.ID, model.ID) error
	UnpinMessage(context.Context, model.ID, model.ID) error
	Delete(context.Context, model.ID) error
	CreateDM(context.Context, model.ID) (model.ID, error)
	SearchUsers(context.Context, string) ([]model.User, error)
	GetContact(context.Context, model.ID) (*model.User, error)
	SetAdmin(context.Context, model.ID, model.ID, bool) error
	SetName(context.Context, model.ID, string) error
	SetEmoji(context.Context, model.ID, string) error
	SetNickname(context.Context, model.ID, model.ID, string) error
}

type E2EE interface {
	SendE2EE(context.Context, model.E2EESendRequest) (model.SendResult, error)
	ReactE2EE(context.Context, model.E2EEReactionRequest) error
	EditE2EE(context.Context, string, model.ID, string) error
	UnsendE2EE(context.Context, string, model.ID) error
	TypingE2EE(context.Context, string, bool) error
	ReadE2EE(context.Context, model.E2EEReadRequest) error
}

type Facebook interface {
	User(context.Context, model.ID) (*model.FacebookUser, error)
	Search(context.Context, string, int) ([]model.SearchResult, error)
	Notifications(context.Context, int) ([]model.Notification, error)
	SetBio(context.Context, string, bool) error
	CreateAdditionalProfile(context.Context, string, string) error
	Unfriend(context.Context, model.ID) error
	SetBlocked(context.Context, model.ID, bool) error
	CreatePost(context.Context, string) (*model.Post, error)
	ArchivePost(context.Context, model.ID, model.PostOwnership) error
	DeletePost(context.Context, model.ID, model.PostOwnership) error
	CreateMarketplaceListing(context.Context, model.MarketplaceListingInput) (*model.MarketplaceListing, error)
	MarketplaceListing(context.Context, model.ID) (*model.MarketplaceListing, error)
	SetProfessionalMode(context.Context, bool) error
}

type Client interface {
	Connect(context.Context) error
	Close() error
	Health() model.HealthSnapshot
	Account() model.User
	MessengerService() Messenger
	ThreadService() Threads
	E2EEService() E2EE
	FacebookService() Facebook
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

func (c *managedClient) Connect(ctx context.Context) error { return c.client.Connect(ctx) }
func (c *managedClient) Close() error                      { return c.client.Close() }
func (c *managedClient) Health() model.HealthSnapshot      { return c.client.Health() }
func (c *managedClient) Account() model.User               { return c.client.Account() }
func (c *managedClient) MessengerService() Messenger       { return c.client.Messenger }
func (c *managedClient) ThreadService() Threads            { return c.client.Threads }
func (c *managedClient) E2EEService() E2EE                 { return c.client.Messenger }
func (c *managedClient) FacebookService() Facebook         { return c.client.Facebook }
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
