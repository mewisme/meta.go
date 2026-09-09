package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.mewis.me/fbgo/storage"
)

const (
	secretCookies = "cookies"
	secretSession = "session"
	secretE2EE    = "e2ee_state"
)

type ProfileManager struct {
	Profiles storage.ProfileStore
	Secrets  storage.SecretStore
}

func (m ProfileManager) Create(ctx context.Context, name, displayName string) (storage.Profile, error) {
	if m.Profiles == nil || m.Secrets == nil {
		return storage.Profile{}, errors.New("profile manager stores are required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return storage.Profile{}, errors.New("profile name is required")
	}
	profile := storage.Profile{Name: name, DisplayName: displayName}
	if err := m.Profiles.Put(ctx, profile); err != nil {
		return storage.Profile{}, err
	}
	return profile, nil
}

func (m ProfileManager) ImportCookies(ctx context.Context, profile string, cookies Cookies) error {
	if err := cookies.ValidateRegular(); err != nil {
		return err
	}
	if _, err := m.Profiles.Get(ctx, profile); err != nil {
		return err
	}
	return m.Secrets.Put(ctx, profile, secretCookies, []byte(cookies.String()))
}

func (m ProfileManager) LoadCookies(ctx context.Context, profile string) (Cookies, error) {
	data, err := m.Secrets.Get(ctx, profile, secretCookies)
	if err != nil {
		return nil, err
	}
	return ParseCookieString(string(data))
}

func (m ProfileManager) SaveSession(ctx context.Context, profile string, session Session) error {
	type persistedSession struct {
		FBID           string `json:"fbid"`
		DTSG           string `json:"dtsg"`
		Jazoest        string `json:"jazoest"`
		SessionID      string `json:"session_id"`
		ClientRevision int64  `json:"client_revision"`
	}
	data, err := json.Marshal(persistedSession{FBID: session.FBID.String(), DTSG: session.DTSG, Jazoest: session.Jazoest, SessionID: session.SessionID, ClientRevision: session.ClientRevision})
	if err != nil {
		return err
	}
	return m.Secrets.Put(ctx, profile, secretSession, data)
}

func (m ProfileManager) Refresh(ctx context.Context, profile string, validator SessionValidator) (Session, error) {
	cookies, err := m.LoadCookies(ctx, profile)
	if err != nil {
		return Session{}, err
	}
	session, err := validator.Validate(ctx, cookies)
	if err != nil {
		return Session{}, err
	}
	if err := m.SaveSession(ctx, profile, session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (m ProfileManager) RefreshLegacyTokens(ctx context.Context, profile string, bootstrapper SessionBootstrapper) (Session, error) {
	cookies, err := m.LoadCookies(ctx, profile)
	if err != nil {
		return Session{}, err
	}
	session, err := bootstrapper.Bootstrap(ctx, cookies)
	if err != nil {
		return Session{}, err
	}
	if err := m.SaveSession(ctx, profile, session); err != nil {
		return Session{}, err
	}
	return session, nil
}

type LogoutPolicy struct {
	RemoveCookies bool
	RemoveSession bool
	RemoveE2EE    bool
}

func (m ProfileManager) Logout(ctx context.Context, profile string, policy LogoutPolicy) error {
	var errs []error
	deleteSecret := func(key string) {
		if err := m.Secrets.Delete(ctx, profile, key); err != nil && !errors.Is(err, storage.ErrNotFound) {
			errs = append(errs, fmt.Errorf("delete %s: %w", key, err))
		}
	}
	if policy.RemoveCookies {
		deleteSecret(secretCookies)
	}
	if policy.RemoveSession {
		deleteSecret(secretSession)
	}
	if policy.RemoveE2EE {
		deleteSecret(secretE2EE)
	}
	return errors.Join(errs...)
}

func (m ProfileManager) ImportLegacyCookieString(ctx context.Context, profile, raw string) error {
	if _, err := m.Secrets.Get(ctx, profile, secretCookies); err == nil {
		return errors.New("refusing to overwrite existing cookie secret")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	cookies, err := ParseCookieString(raw)
	if err != nil {
		return err
	}
	return m.ImportCookies(ctx, profile, cookies)
}

func (m ProfileManager) ImportLegacyJSON(ctx context.Context, profile string, data []byte) error {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	for _, key := range []string{"cookies", "cookieFacebook", "setCookies"} {
		if raw, ok := payload[key].(string); ok && strings.TrimSpace(raw) != "" {
			return m.ImportLegacyCookieString(ctx, profile, raw)
		}
	}
	return errors.New("legacy JSON does not contain a supported cookie field")
}

func (m ProfileManager) ImportLegacyE2EEState(ctx context.Context, profile string, data []byte) error {
	if len(data) == 0 {
		return errors.New("legacy E2EE state is empty")
	}
	if _, err := m.Profiles.Get(ctx, profile); err != nil {
		return err
	}
	if _, err := m.Secrets.Get(ctx, profile, secretE2EE); err == nil {
		return errors.New("refusing to overwrite existing E2EE state")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	return m.Secrets.Put(ctx, profile, secretE2EE, append([]byte(nil), data...))
}
