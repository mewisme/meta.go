package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrInvalidAppState = errors.New("invalid app state")

type AppStateCookie struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	HostOnly bool   `json:"hostOnly,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
}

type AppState []AppStateCookie

func ParseAppStateJSON(data []byte) (AppState, error) {
	var raw []struct {
		Key      string `json:"key"`
		Name     string `json:"name"`
		Value    string `json:"value"`
		Domain   string `json:"domain"`
		Path     string `json:"path"`
		HostOnly bool   `json:"hostOnly"`
		Secure   bool   `json:"secure"`
		HTTPOnly bool   `json:"httpOnly"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidAppState, err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: no cookies", ErrInvalidAppState)
	}
	state := make(AppState, 0, len(raw))
	for _, item := range raw {
		key, name := strings.TrimSpace(item.Key), strings.TrimSpace(item.Name)
		if key != "" && name != "" && key != name {
			return nil, fmt.Errorf("%w: conflicting key and name", ErrInvalidAppState)
		}
		if key == "" {
			key = name
		}
		cookie := AppStateCookie{Key: key, Value: item.Value, Domain: item.Domain, Path: item.Path, HostOnly: item.HostOnly, Secure: item.Secure, HTTPOnly: item.HTTPOnly}
		if err := validateAppStateCookie(cookie); err != nil {
			return nil, err
		}
		state = append(state, cookie)
	}
	return state, nil
}

func (s AppState) Cookies() (Cookies, error) {
	if len(s) == 0 {
		return nil, fmt.Errorf("%w: no cookies", ErrInvalidAppState)
	}
	cookies := make(Cookies, len(s))
	for _, item := range s {
		if err := validateAppStateCookie(item); err != nil {
			return nil, err
		}
		cookies[strings.TrimSpace(item.Key)] = item.Value
	}
	if err := cookies.ValidateRegular(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidAppState, err)
	}
	return cookies, nil
}

func (s AppState) Clone() AppState { return append(AppState(nil), s...) }

func (c Cookies) AppState() AppState {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	state := make(AppState, 0, len(keys))
	for _, key := range keys {
		state = append(state, AppStateCookie{Key: key, Value: c[key]})
	}
	return state
}

func validateAppStateCookie(cookie AppStateCookie) error {
	if strings.TrimSpace(cookie.Key) == "" {
		return fmt.Errorf("%w: cookie key is required", ErrInvalidAppState)
	}
	if cookie.Value == "" {
		return fmt.Errorf("%w: cookie %q value is required", ErrInvalidAppState, strings.TrimSpace(cookie.Key))
	}
	return nil
}
