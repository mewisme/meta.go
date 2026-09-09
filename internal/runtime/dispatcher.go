package runtime

import (
	"sync"
	"sync/atomic"

	"go.mewis.me/fbgo/model"
)

type Event struct {
	Kind string
	Data any
}

type Dispatcher struct {
	queue    chan model.Event
	closed   chan struct{}
	once     sync.Once
	dropped  atomic.Uint64
	mu       sync.RWMutex
	nextID   uint64
	handlers map[uint64]func(model.Event)
}

func NewDispatcher(size int) *Dispatcher {
	if size < 1 {
		size = 100
	}
	return &Dispatcher{queue: make(chan model.Event, size), closed: make(chan struct{}), handlers: map[uint64]func(model.Event){}}
}

func (d *Dispatcher) Publish(event model.Event) bool {
	select {
	case <-d.closed:
		return false
	default:
	}
	published := false
	select {
	case d.queue <- event:
		published = true
	default:
	}
	if !published {
		select {
		case <-d.queue:
			d.dropped.Add(1)
		default:
		}
		select {
		case d.queue <- event:
			published = true
		case <-d.closed:
			return false
		default:
			d.dropped.Add(1)
		}
	}
	d.mu.RLock()
	handlers := make([]func(model.Event), 0, len(d.handlers))
	for _, handler := range d.handlers {
		handlers = append(handlers, handler)
	}
	d.mu.RUnlock()
	for _, handler := range handlers {
		handler(event)
	}
	return published
}

func (d *Dispatcher) Events() <-chan model.Event { return d.queue }
func (d *Dispatcher) Dropped() uint64            { return d.dropped.Load() }

func (d *Dispatcher) On(handler func(model.Event)) func() {
	if handler == nil {
		return func() {}
	}
	d.mu.Lock()
	d.nextID++
	id := d.nextID
	d.handlers[id] = handler
	d.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			d.mu.Lock()
			delete(d.handlers, id)
			d.mu.Unlock()
		})
	}
}

func (d *Dispatcher) Close() {
	d.once.Do(func() { close(d.closed) })
}

func (d *Dispatcher) Health() model.HealthSnapshot {
	return model.HealthSnapshot{Regular: model.ConnectionDisconnected, E2EE: model.ConnectionDisconnected, DroppedEventCount: d.Dropped()}
}
