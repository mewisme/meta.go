package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.mewis.me/meta.go/storage"
)

const (
	secretCookies = "cookies"
	secretSession = "session"
	secretE2EE    = "e2ee_state"
)

var managedSecretKeys = [...]string{secretCookies, secretSession, secretE2EE}

type ProfileManager struct {
	Profiles storage.ProfileStore
	Secrets  storage.SecretStore
}

func (m ProfileManager) Create(ctx context.Context, name, displayName string) (storage.Profile, error) {
	if m.Profiles == nil || m.Secrets == nil {
		return storage.Profile{}, errors.New("profile manager stores are required")
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsRune(name, '\x00') {
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

func (m ProfileManager) Rename(ctx context.Context, oldName, newName string) (storage.Profile, error) {
	oldName, newName = strings.TrimSpace(oldName), strings.TrimSpace(newName)
	if oldName == "" || newName == "" || strings.ContainsRune(oldName, '\x00') || strings.ContainsRune(newName, '\x00') {
		return storage.Profile{}, errors.New("profile names are required")
	}
	if oldName == newName {
		return m.Profiles.Get(ctx, oldName)
	}
	if _, err := m.Profiles.Get(ctx, newName); err == nil {
		return storage.Profile{}, errors.New("destination profile already exists")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return storage.Profile{}, err
	}
	profile, err := m.Profiles.Get(ctx, oldName)
	if err != nil {
		return storage.Profile{}, err
	}
	values := make(map[string][]byte, len(managedSecretKeys))
	for _, key := range managedSecretKeys {
		if _, err := m.Secrets.Get(ctx, newName, key); err == nil {
			return storage.Profile{}, fmt.Errorf("destination profile secret %q already exists", key)
		} else if !errors.Is(err, storage.ErrNotFound) {
			return storage.Profile{}, err
		}
		value, err := m.Secrets.Get(ctx, oldName, key)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return storage.Profile{}, err
		}
		values[key] = value
	}
	original := profile
	profile.Name = newName
	if err := m.Profiles.Put(ctx, profile); err != nil {
		return storage.Profile{}, err
	}
	created := make([]string, 0, len(values))
	rollbackNew := func(cause error) error {
		rollbackCtx := context.WithoutCancel(ctx)
		var rollback []error
		for _, key := range created {
			if err := m.Secrets.Delete(rollbackCtx, newName, key); err != nil && !errors.Is(err, storage.ErrNotFound) {
				rollback = append(rollback, err)
			}
		}
		if err := m.Profiles.Delete(rollbackCtx, newName); err != nil && !errors.Is(err, storage.ErrNotFound) {
			rollback = append(rollback, err)
		}
		return errors.Join(cause, errors.Join(rollback...))
	}
	for _, key := range managedSecretKeys {
		value, ok := values[key]
		if !ok {
			continue
		}
		if err := m.Secrets.Put(ctx, newName, key, value); err != nil {
			return storage.Profile{}, rollbackNew(err)
		}
		created = append(created, key)
	}
	if err := m.Profiles.Delete(ctx, oldName); err != nil {
		return storage.Profile{}, rollbackNew(err)
	}
	deleted := make([]string, 0, len(values))
	for _, key := range managedSecretKeys {
		if _, ok := values[key]; !ok {
			continue
		}
		if err := m.Secrets.Delete(ctx, oldName, key); err != nil {
			rollbackCtx := context.WithoutCancel(ctx)
			var rollback []error
			if restoreErr := m.Profiles.Put(rollbackCtx, original); restoreErr != nil {
				rollback = append(rollback, restoreErr)
			}
			for _, restoreKey := range deleted {
				if restoreErr := m.Secrets.Put(rollbackCtx, oldName, restoreKey, values[restoreKey]); restoreErr != nil {
					rollback = append(rollback, restoreErr)
				}
			}
			for _, newKey := range created {
				if removeErr := m.Secrets.Delete(rollbackCtx, newName, newKey); removeErr != nil && !errors.Is(removeErr, storage.ErrNotFound) {
					rollback = append(rollback, removeErr)
				}
			}
			if removeErr := m.Profiles.Delete(rollbackCtx, newName); removeErr != nil && !errors.Is(removeErr, storage.ErrNotFound) {
				rollback = append(rollback, removeErr)
			}
			return storage.Profile{}, errors.Join(err, errors.Join(rollback...))
		}
		deleted = append(deleted, key)
	}
	return profile, nil
}

func (m ProfileManager) Remove(ctx context.Context, name string) error {
	if err := m.Logout(ctx, name, LogoutPolicy{RemoveCookies: true, RemoveSession: true, RemoveE2EE: true}); err != nil {
		return err
	}
	return m.Profiles.Delete(ctx, name)
}
