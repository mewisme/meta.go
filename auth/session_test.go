package auth

import (
	"errors"
	"testing"

	"go.mau.fi/mautrix-meta/pkg/messagix"
	fberrors "go.mewis.me/fbgo/errors"
)

func TestParseHomepage(t *testing.T) {
	body := []byte(`<script>DTSGInitialData",[],{"token":"DTSG"}</script><input name="jazoest" value="22000"><script>{"sessionId":"sid","actorID":"123","client_revision":456,}</script>`)
	session, err := ParseHomepage(body, Cookies{"c_user": "123", "xs": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if session.FBID != "123" || session.DTSG != "DTSG" || session.SessionID != "sid" || session.ClientRevision != 456 {
		t.Fatalf("unexpected session: %#v", session)
	}
}

func TestParseHomepageProtocolChange(t *testing.T) {
	_, err := ParseHomepage([]byte(`<html></html>`), Cookies{"c_user": "123", "xs": "x"})
	if !errors.Is(err, fberrors.ErrProtocolChanged) {
		t.Fatalf("expected protocol change, got %v", err)
	}
}

func TestNormalizeMessagixAuthError(t *testing.T) {
	if err := normalizeMessagixAuthError(messagix.ErrCheckpointRequired); !errors.Is(err, fberrors.ErrCheckpointRequired) {
		t.Fatalf("expected checkpoint normalization, got %v", err)
	}
	if err := normalizeMessagixAuthError(messagix.ErrTokenInvalidated); !errors.Is(err, fberrors.ErrSessionExpired) {
		t.Fatalf("expected session expiry normalization, got %v", err)
	}
}
