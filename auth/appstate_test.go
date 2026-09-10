package auth

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseAppStateJSON(t *testing.T) {
	state, err := ParseAppStateJSON([]byte(`[{"key":"c_user","value":"1","domain":".facebook.com","unknown":true},{"name":"xs","value":"x","httpOnly":true}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(state) != 2 || state[0].Key != "c_user" || state[0].Domain != ".facebook.com" || state[1].Key != "xs" || !state[1].HTTPOnly {
		t.Fatalf("unexpected state: %#v", state)
	}
	cookies, err := state.Cookies()
	if err != nil {
		t.Fatal(err)
	}
	if cookies["c_user"] != "1" || cookies["xs"] != "x" {
		t.Fatalf("unexpected cookies: %#v", cookies)
	}
}

func TestAppStateRejectsMalformedInput(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(`[]`),
		[]byte(`[{"value":"x"}]`),
		[]byte(`[{"key":"xs","value":""}]`),
		[]byte(`[{"key":"xs","name":"c_user","value":"x"}]`),
		[]byte(`[{"key":"xs","value":"x"}]`),
	} {
		state, err := ParseAppStateJSON(data)
		if err == nil {
			_, err = state.Cookies()
		}
		if !errors.Is(err, ErrInvalidAppState) {
			t.Fatalf("expected invalid app state for %s, got %v", data, err)
		}
	}
}

func TestAppStateDuplicateCookieUsesLastValue(t *testing.T) {
	state := AppState{{Key: "c_user", Value: "1"}, {Key: "xs", Value: "old"}, {Key: "xs", Value: "new"}}
	cookies, err := state.Cookies()
	if err != nil {
		t.Fatal(err)
	}
	if cookies["xs"] != "new" {
		t.Fatalf("xs = %q", cookies["xs"])
	}
}

func TestCookiesAppStateRoundTrip(t *testing.T) {
	cookies := Cookies{"xs": "x", "c_user": "1", "fr": "f"}
	state := cookies.AppState()
	if got := []string{state[0].Key, state[1].Key, state[2].Key}; !reflect.DeepEqual(got, []string{"c_user", "fr", "xs"}) {
		t.Fatalf("unexpected order: %#v", got)
	}
	roundTrip, err := state.Cookies()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roundTrip, cookies) {
		t.Fatalf("round trip mismatch: %#v", roundTrip)
	}
}
