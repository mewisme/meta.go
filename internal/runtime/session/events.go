package session

import (
	"sync"
	"sync/atomic"
	"time"

	"go.mewis.me/meta.go/model"
)

const DefaultSubscriberBuffer = 100

type Event struct {
	SessionID string
	Sequence  uint64
	EmittedAt time.Time
	Payload   model.Event
}

type subscriber struct {
	events  chan Event
	dropped atomic.Uint64
}

type Subscription struct {
	Events <-chan Event
	cancel func()
	once   sync.Once
	drops  *atomic.Uint64
}

func (s *Subscription) Close() {
	if s == nil {
		return
	}
	s.once.Do(s.cancel)
}

func (s *Subscription) Dropped() uint64 {
	if s == nil || s.drops == nil {
		return 0
	}
	return s.drops.Load()
}

func (s *Session) Subscribe(buffer int) (*Subscription, error) {
	if s == nil || s.closed.Load() {
		return nil, ErrClosed
	}
	if buffer < 1 {
		buffer = DefaultSubscriberBuffer
	}
	sub := &subscriber{events: make(chan Event, buffer)}
	s.eventMu.Lock()
	if s.closed.Load() {
		s.eventMu.Unlock()
		return nil, ErrClosed
	}
	s.nextSubscriberID++
	id := s.nextSubscriberID
	s.subscribers[id] = sub
	s.eventMu.Unlock()
	return &Subscription{Events: sub.events, drops: &sub.dropped, cancel: func() { s.removeSubscriber(id) }}, nil
}

func (s *Session) SubscriberDropped() uint64 {
	if s == nil {
		return 0
	}
	return s.subscriberDropped.Load()
}

func (s *Session) SubscriberCount() int {
	if s == nil {
		return 0
	}
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	return len(s.subscribers)
}

func (s *Session) publish(payload model.Event) {
	if s == nil || s.closed.Load() {
		return
	}
	s.eventMu.Lock()
	if s.closed.Load() {
		s.eventMu.Unlock()
		return
	}
	s.sequence++
	event := Event{SessionID: s.id, Sequence: s.sequence, EmittedAt: time.Now().UTC(), Payload: payload}
	for _, sub := range s.subscribers {
		select {
		case sub.events <- event:
		default:
			select {
			case <-sub.events:
				sub.dropped.Add(1)
				s.subscriberDropped.Add(1)
			default:
			}
			select {
			case sub.events <- event:
			default:
				sub.dropped.Add(1)
				s.subscriberDropped.Add(1)
			}
		}
	}
	s.eventMu.Unlock()
}

func (s *Session) removeSubscriber(id uint64) {
	s.eventMu.Lock()
	sub := s.subscribers[id]
	delete(s.subscribers, id)
	if sub != nil {
		close(sub.events)
	}
	s.eventMu.Unlock()
}

func (s *Session) closeSubscribers() {
	s.eventMu.Lock()
	for id, sub := range s.subscribers {
		delete(s.subscribers, id)
		close(sub.events)
	}
	s.eventMu.Unlock()
}
