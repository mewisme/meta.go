package meta

import (
	"context"
	"errors"
	"testing"

	fberrors "go.mewis.me/meta.go/errors"
)

type authFakeBackend struct {
	*fakeBackend
	state     AuthState
	refreshes int
	fail      bool
}

func (b *authFakeBackend) AuthState(context.Context) (AuthState, error) {
	return cloneAuthState(b.state), nil
}

func (b *authFakeBackend) RefreshAuth(_ context.Context, cookies map[string]string) (AuthState, error) {
	b.refreshes++
	if b.fail {
		return AuthState{}, errors.New("refresh failed")
	}
	b.state.Cookies = cloneStringMap(cookies)
	return cloneAuthState(b.state), nil
}

func TestEngineRefreshAuthPreservesConnectionAndListeners(t *testing.T) {
	backend := &authFakeBackend{fakeBackend: &fakeBackend{}, state: AuthState{Cookies: map[string]string{"c_user": "1", "xs": "old"}, FBID: "1", DTSG: "old"}}
	engine := newEngine(context.Background(), backend, 4)
	defer engine.Close()
	if _, err := engine.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	called := 0
	engine.On(func(Event) { called++ })
	engine.emit(Event{Kind: EventMessage})
	state, err := engine.RefreshAuth(context.Background(), map[string]string{"c_user": "1", "xs": "new"})
	if err != nil {
		t.Fatal(err)
	}
	engine.emit(Event{Kind: EventMessage})
	backend.mu.Lock()
	disconnected := backend.disconnected
	backend.mu.Unlock()
	if state.Cookies["xs"] != "new" || backend.refreshes != 1 || disconnected || !engine.Connected() || called != 2 {
		t.Fatalf("refresh broke continuity: state=%#v refreshes=%d disconnected=%t connected=%t calls=%d", state, backend.refreshes, disconnected, engine.Connected(), called)
	}
}

func TestEngineFailedRefreshLeavesConnectionAndListenersAlive(t *testing.T) {
	backend := &authFakeBackend{fakeBackend: &fakeBackend{}, state: AuthState{Cookies: map[string]string{"c_user": "1", "xs": "old"}, FBID: "1"}, fail: true}
	engine := newEngine(context.Background(), backend, 4)
	defer engine.Close()
	if _, err := engine.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	called := 0
	engine.On(func(Event) { called++ })
	if _, err := engine.RefreshAuth(context.Background(), map[string]string{"c_user": "1", "xs": "new"}); err == nil {
		t.Fatal("expected refresh failure")
	}
	engine.emit(Event{Kind: EventMessage})
	state, err := engine.AuthState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	backend.mu.Lock()
	disconnected := backend.disconnected
	backend.mu.Unlock()
	if state.Cookies["xs"] != "old" || disconnected || !engine.Connected() || called != 1 {
		t.Fatalf("failed refresh broke state: state=%#v disconnected=%t connected=%t calls=%d", state, disconnected, engine.Connected(), called)
	}
}

func TestEngineAuthRefreshUnsupportedAndNotConnected(t *testing.T) {
	engine := newEngine(context.Background(), &fakeBackend{}, 1)
	defer engine.Close()
	if _, err := engine.RefreshAuth(context.Background(), map[string]string{"c_user": "1", "xs": "x"}); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected not connected, got %v", err)
	}
	if _, err := engine.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.AuthState(context.Background()); !errors.Is(err, fberrors.ErrUnsupported) {
		t.Fatalf("expected unsupported, got %v", err)
	}
}

func cloneAuthState(state AuthState) AuthState {
	state.Cookies = cloneStringMap(state.Cookies)
	return state
}

func cloneStringMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
