package server

import (
	"fmt"

	"go.mewis.me/meta.go/auth"
	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
)

func authSourceFromProto(value *metav1.SessionAuth) (auth.Source, error) {
	if value == nil {
		return auth.Source{}, fmt.Errorf("%w: authentication is required", fberrors.ErrInvalidInput)
	}
	var source auth.Source
	switch value := value.Source.(type) {
	case *metav1.SessionAuth_Cookies:
		if value.Cookies == nil {
			return auth.Source{}, fmt.Errorf("%w: cookies are required", fberrors.ErrInvalidInput)
		}
		source.Cookies = make(auth.Cookies, len(value.Cookies.Values))
		for key, cookie := range value.Cookies.Values {
			source.Cookies[key] = cookie
		}
	case *metav1.SessionAuth_AppState:
		if value.AppState == nil || len(value.AppState.Cookies) == 0 {
			return auth.Source{}, fmt.Errorf("%w: app state is required", fberrors.ErrInvalidInput)
		}
		source.AppState = make(auth.AppState, 0, len(value.AppState.Cookies))
		for _, cookie := range value.AppState.Cookies {
			if cookie == nil {
				return auth.Source{}, fmt.Errorf("%w: app state cookie is required", fberrors.ErrInvalidInput)
			}
			source.AppState = append(source.AppState, auth.AppStateCookie{Key: cookie.Key, Value: cookie.Value, Domain: cookie.Domain, Path: cookie.Path, HostOnly: cookie.HostOnly, Secure: cookie.Secure, HTTPOnly: cookie.HttpOnly})
		}
	case *metav1.SessionAuth_Credentials:
		if value.Credentials == nil {
			return auth.Source{}, fmt.Errorf("%w: credentials are required", fberrors.ErrInvalidInput)
		}
		credentials := auth.Credentials{Identifier: value.Credentials.Identifier, Password: value.Credentials.Password}
		switch factor := value.Credentials.SecondFactor.(type) {
		case *metav1.Credentials_Totp:
			credentials.TOTP = factor.Totp
		case *metav1.Credentials_Otp:
			credentials.OTP = factor.Otp
		}
		source.Credentials = &credentials
	default:
		return auth.Source{}, fmt.Errorf("%w: authentication source is required", fberrors.ErrInvalidInput)
	}
	if err := source.Validate(); err != nil {
		return auth.Source{}, err
	}
	return source, nil
}

func authSnapshotToProto(snapshot auth.AuthSnapshot) *metav1.AuthSnapshot {
	cookies := make(map[string]string, len(snapshot.Cookies))
	for key, value := range snapshot.Cookies {
		cookies[key] = value
	}
	state := make([]*metav1.AppStateCookie, 0, len(snapshot.AppState))
	for _, cookie := range snapshot.AppState {
		state = append(state, &metav1.AppStateCookie{Key: cookie.Key, Value: cookie.Value, Domain: cookie.Domain, Path: cookie.Path, HostOnly: cookie.HostOnly, Secure: cookie.Secure, HttpOnly: cookie.HTTPOnly})
	}
	session := snapshot.Session
	return &metav1.AuthSnapshot{
		Cookies:  &metav1.CookieMap{Values: cookies},
		AppState: &metav1.AppState{Cookies: state},
		Session: &metav1.FacebookSession{AccountId: session.FBID.String(), Name: session.Name, Username: session.Username, Dtsg: session.DTSG, Jazoest: session.Jazoest, Lsd: session.LSD, SessionId: session.SessionID,
			ClientRevision: session.ClientRevision, RefreshedAt: timestampOrNil(session.BootstrappedAt)},
	}
}
