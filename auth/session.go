package auth

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.mewis.me/meta-extra/pkg/messagix"
	metaCookies "go.mewis.me/meta-extra/pkg/messagix/cookies"
	metaHTTP "go.mewis.me/meta-extra/pkg/messagix/httpclient"
	metaTypes "go.mewis.me/meta-extra/pkg/messagix/types"

	fberrors "go.mewis.me/meta.go/errors"
	"go.mewis.me/meta.go/internal/webapi"
	"go.mewis.me/meta.go/model"
)

type Session struct {
	Cookies        Cookies
	FBID           model.ID
	Name           string
	Username       string
	DTSG           string
	Jazoest        string
	LSD            string
	SessionID      string
	ClientRevision int64
	BootstrappedAt time.Time
}

type SessionValidator struct{}

func (v SessionValidator) Validate(ctx context.Context, cookies Cookies) (Session, error) {
	if err := cookies.ValidateRegular(); err != nil {
		return Session{}, err
	}
	jar := &metaCookies.Cookies{Platform: metaTypes.Facebook}
	values := make(map[metaCookies.MetaCookieName]string, len(cookies))
	for key, value := range cookies {
		values[metaCookies.MetaCookieName(key)] = value
	}
	jar.UpdateValues(values)
	client := messagix.NewClient(jar, zerolog.Nop(), &messagix.Config{})
	user, _, err := client.LoadMessagesPage(ctx)
	if err != nil {
		if errors.Is(err, metaHTTP.ErrTokenInvalidated) || errors.Is(err, metaHTTP.ErrCheckpointRequired) {
			return Session{}, normalizeMessagixAuthError(err)
		}
		return Session{}, err
	}
	if user == nil || user.GetFBID() == 0 {
		return Session{}, &fberrors.ProtocolError{Operation: "messagix session validation", Cause: fberrors.ErrProtocolChanged}
	}
	return Session{Cookies: cookies.Clone(), FBID: model.ID(strconv.FormatInt(user.GetFBID(), 10)), Name: user.GetName(), Username: user.GetUsername(), BootstrappedAt: time.Now()}, nil
}

func normalizeMessagixAuthError(err error) error {
	switch {
	case errors.Is(err, metaHTTP.ErrCheckpointRequired):
		return fmt.Errorf("%w: %v", fberrors.ErrCheckpointRequired, err)
	case errors.Is(err, metaHTTP.ErrTokenInvalidated):
		return fmt.Errorf("%w: %v", fberrors.ErrSessionExpired, err)
	default:
		return err
	}
}

type SessionBootstrapper struct {
	HTTP *http.Client
	URL  string
}

var sessionPatterns = map[string][]*regexp.Regexp{
	"dtsg": {
		regexp.MustCompile(`DTSGInitialData[^}]*"token":"([^"]+)"`),
		regexp.MustCompile(`"async_get_token":"([^"]+)"`),
	},
	"jazoest": {
		regexp.MustCompile(`jazoest=([0-9]+)`),
		regexp.MustCompile(`"jazoest":"?([0-9]+)`),
		regexp.MustCompile(`name=["']jazoest["'][^>]*value=["']([0-9]+)["']`),
	},
	"lsd": {
		regexp.MustCompile(`"LSD"[^}]*"token":"([^"]+)"`),
		regexp.MustCompile(`"lsd":"([^"]+)"`),
	},
	"session_id": {
		regexp.MustCompile(`"sessionId":"([^"]+)"`),
	},
	"actor_id": {
		regexp.MustCompile(`"actorID":"([0-9]+)"`),
	},
	"client_revision": {
		regexp.MustCompile(`"client_revision":([0-9]+)`),
	},
}

func (b SessionBootstrapper) Bootstrap(ctx context.Context, cookies Cookies) (Session, error) {
	if err := cookies.ValidateRegular(); err != nil {
		return Session{}, err
	}
	client := b.HTTP
	if client == nil {
		client = webapi.NewClient(nil).HTTP
	}
	url := b.URL
	if url == "" {
		url = "https://www.facebook.com/"
	}
	req, err := webapi.NewRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Session{}, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/140.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", cookies.String())
	resp, err := client.Do(req)
	if err != nil {
		return Session{}, fmt.Errorf("bootstrap session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return Session{}, fberrors.ErrSessionExpired
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Session{}, &fberrors.ProtocolError{Operation: "session bootstrap", Endpoint: url, StatusCode: resp.StatusCode, Cause: fberrors.ErrProtocolChanged}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return Session{}, err
	}
	return ParseHomepage(body, cookies)
}

func ParseHomepage(body []byte, cookies Cookies) (Session, error) {
	text := html.UnescapeString(string(body))
	dtsg := firstMatch(text, sessionPatterns["dtsg"])
	jazoest := firstMatch(text, sessionPatterns["jazoest"])
	lsd := firstMatch(text, sessionPatterns["lsd"])
	sessionID := firstMatch(text, sessionPatterns["session_id"])
	fbid := firstMatch(text, sessionPatterns["actor_id"])
	if fbid == "" {
		fbid = cookies["c_user"]
	}
	revisionText := firstMatch(text, sessionPatterns["client_revision"])
	revision, _ := strconv.ParseInt(revisionText, 10, 64)
	missing := make([]string, 0, 5)
	for key, value := range map[string]string{"fb_dtsg": dtsg, "jazoest": jazoest, "sessionID": sessionID, "FacebookID": fbid, "clientRevision": revisionText} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return Session{}, &fberrors.ProtocolError{Operation: "parse homepage bootstrap", Code: strings.Join(missing, ","), Cause: fberrors.ErrProtocolChanged}
	}
	return Session{Cookies: cookies.Clone(), FBID: model.ID(fbid), DTSG: dtsg, Jazoest: jazoest, LSD: lsd, SessionID: sessionID, ClientRevision: revision, BootstrappedAt: time.Now()}, nil
}

func firstMatch(text string, patterns []*regexp.Regexp) string {
	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) > 1 {
			return match[1]
		}
	}
	return ""
}
