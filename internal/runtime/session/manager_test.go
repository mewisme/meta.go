package session

import (
	"context"
	"errors"
	"testing"

	"go.mewis.me/meta.go/facebook"
	"go.mewis.me/meta.go/messenger"
	"go.mewis.me/meta.go/model"
	"go.mewis.me/meta.go/thread"
)

type fakeClient struct {
	account  model.User
	health   model.HealthSnapshot
	connects int
	closes   int
}

func (f *fakeClient) Connect(context.Context) error {
	f.connects++
	return nil
}

func (f *fakeClient) Close() error {
	f.closes++
	return nil
}

func (f *fakeClient) Health() model.HealthSnapshot         { return f.health }
func (f *fakeClient) Account() model.User                  { return f.account }
func (f *fakeClient) MessengerService() *messenger.Service { return nil }
func (f *fakeClient) ThreadService() *thread.Service       { return nil }
func (f *fakeClient) FacebookService() *facebook.Service   { return nil }
func (f *fakeClient) Subscribe(func(model.Event)) func()   { return func() {} }

func TestManagerLifecycle(t *testing.T) {
	fake := &fakeClient{account: model.User{ID: "42", Name: "Mew"}, health: model.HealthSnapshot{Regular: model.ConnectionConnected}}
	var captured Config
	manager := NewManager(func(config Config) (Client, error) {
		captured = config
		return fake, nil
	})
	created, err := manager.Create(Config{Cookies: map[string]string{"c_user": "42", "xs": "secret"}, E2EE: true, EventBuffer: 32})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID() == "" || manager.Len() != 1 || !captured.E2EE || captured.EventBuffer != 32 {
		t.Fatalf("unexpected created session: id=%q len=%d config=%#v", created.ID(), manager.Len(), captured)
	}
	got, err := manager.Get(created.ID())
	if err != nil || got != created {
		t.Fatalf("get session: got=%p want=%p err=%v", got, created, err)
	}
	account, err := got.Connect(t.Context())
	if err != nil || account.ID != "42" || fake.connects != 1 {
		t.Fatalf("connect: account=%#v connects=%d err=%v", account, fake.connects, err)
	}
	if health := got.Health(); health.Regular != model.ConnectionConnected {
		t.Fatalf("unexpected health: %#v", health)
	}
	if err := manager.Close(created.ID()); err != nil {
		t.Fatal(err)
	}
	if fake.closes != 1 || manager.Len() != 0 {
		t.Fatalf("unexpected close state: closes=%d len=%d", fake.closes, manager.Len())
	}
	if _, err := manager.Get(created.ID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if err := manager.Close(created.ID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected second close to be not found, got %v", err)
	}
}

func TestManagerCloseAll(t *testing.T) {
	clients := []*fakeClient{{}, {}}
	index := 0
	manager := NewManager(func(Config) (Client, error) {
		client := clients[index]
		index++
		return client, nil
	})
	for range clients {
		if _, err := manager.Create(Config{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := manager.CloseAll(); err != nil {
		t.Fatal(err)
	}
	if manager.Len() != 0 || clients[0].closes != 1 || clients[1].closes != 1 {
		t.Fatalf("unexpected close-all state: len=%d closes=%d,%d", manager.Len(), clients[0].closes, clients[1].closes)
	}
}
