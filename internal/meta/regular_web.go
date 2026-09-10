package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	metaTypes "go.mewis.me/meta-extra/pkg/messagix/types"
	"go.mewis.me/meta-extra/pkg/messagix/useragent"
	"go.mewis.me/meta.go/internal/graphql"
	"go.mewis.me/meta.go/internal/protocol"
	"go.mewis.me/meta.go/internal/webapi"
	"go.mewis.me/meta.go/model"
)

type browserFormState struct {
	FBID           string
	DTSG           string
	Jazoest        string
	ClientRevision string
	LSD            string
}

var browserStatePatterns = map[string][]*regexp.Regexp{
	"dtsg": {
		regexp.MustCompile(`DTSGInitialData[^}]*"token":"([^"]+)"`),
		regexp.MustCompile(`DTSGInitData[^}]*"token":"([^"]+)"`),
		regexp.MustCompile(`"async_get_token":"([^"]+)"`),
	},
	"jazoest": {
		regexp.MustCompile(`(?:[?&]|&amp;)jazoest=([0-9]+)`),
		regexp.MustCompile(`"jazoest":"?([0-9]+)`),
	},
	"client_revision": {
		regexp.MustCompile(`"client_revision":([0-9]+)`),
	},
	"lsd": {
		regexp.MustCompile(`"LSD"[^}]*"token":"([^"]+)"`),
		regexp.MustCompile(`"lsd":"([^"]+)"`),
	},
}

func (b *messagixBackend) browserFormState(ctx context.Context) (browserFormState, error) {
	b.browserStateMu.Lock()
	defer b.browserStateMu.Unlock()
	if b.browserState != nil {
		return *b.browserState, nil
	}
	account, err := b.client.GetCurrentAccount()
	if err != nil {
		return browserFormState{}, err
	}
	headers := browserDocumentHeaders(b.client.GetCookies().String())
	if width, _ := b.client.GetCookies().GetViewports(); width != "" {
		headers.Set("viewport-width", width)
	}
	_, body, err := b.client.GetHTTP().MakeRequest(ctx, b.client.GetEndpoint("messages"), http.MethodGet, headers, nil, metaTypes.NONE)
	if err != nil {
		return browserFormState{}, fmt.Errorf("load Messenger browser form state: %w", err)
	}
	text := html.UnescapeString(string(body))
	state := browserFormState{
		FBID:           strconv.FormatInt(account.GetFBID(), 10),
		DTSG:           firstBrowserStateMatch(text, browserStatePatterns["dtsg"]),
		Jazoest:        firstBrowserStateMatch(text, browserStatePatterns["jazoest"]),
		ClientRevision: firstBrowserStateMatch(text, browserStatePatterns["client_revision"]),
		LSD:            firstBrowserStateMatch(text, browserStatePatterns["lsd"]),
	}
	if state.FBID == "0" || state.DTSG == "" || state.ClientRevision == "0" {
		return browserFormState{}, errors.New("messagix browser form state is incomplete")
	}
	b.browserState = &state
	return state, nil
}

func browserDocumentHeaders(cookie string) http.Header {
	headers := make(http.Header)
	headers.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	headers.Set("accept-language", "en-US,en;q=0.9")
	headers.Set("dpr", useragent.DPR)
	headers.Set("user-agent", useragent.UserAgent)
	headers.Set("sec-ch-ua", useragent.SecCHUserAgent)
	headers.Set("sec-ch-ua-platform", useragent.SecCHPlatform)
	headers.Set("sec-ch-prefers-color-scheme", useragent.SecCHPrefersColorScheme)
	headers.Set("sec-ch-ua-full-version-list", useragent.SecCHFullVersionList)
	headers.Set("sec-ch-ua-mobile", useragent.SecCHMobile)
	headers.Set("sec-ch-ua-model", useragent.SecCHModel)
	headers.Set("sec-ch-ua-platform-version", useragent.SecCHPlatformVersion)
	headers.Set("sec-fetch-dest", "document")
	headers.Set("sec-fetch-mode", "navigate")
	headers.Set("sec-fetch-site", "none")
	headers.Set("sec-fetch-user", "?1")
	headers.Set("upgrade-insecure-requests", "1")
	headers.Set("x-asbd-id", "129477")
	if cookie != "" {
		headers.Set("cookie", cookie)
	}
	return headers
}

func firstBrowserStateMatch(text string, patterns []*regexp.Regexp) string {
	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

func (b *messagixBackend) baseForm(ctx context.Context) (url.Values, browserFormState, error) {
	state, err := b.browserFormState(ctx)
	if err != nil {
		return nil, browserFormState{}, err
	}
	formMap := webapi.BaseForm(webapi.FormSession{FBID: state.FBID, DTSG: state.DTSG, Jazoest: state.Jazoest, ClientRevision: state.ClientRevision}, &b.requestCounter)
	form := make(url.Values, len(formMap)+2)
	for key, value := range formMap {
		form.Set(key, value)
	}
	if state.LSD != "" {
		form.Set("lsd", state.LSD)
	}
	return form, state, nil
}

func (b *messagixBackend) graphQL(ctx context.Context, friendlyName, docID string, variables any) (map[string]any, error) {
	form, state, err := b.baseForm(ctx)
	if err != nil {
		return nil, err
	}
	encodedVariables, err := json.Marshal(variables)
	if err != nil {
		return nil, err
	}
	form.Set("fb_api_caller_class", "RelayModern")
	form.Set("fb_api_req_friendly_name", friendlyName)
	form.Set("variables", string(encodedVariables))
	form.Set("server_timestamps", "true")
	form.Set("doc_id", docID)
	headers := b.graphQLHeaders(friendlyName)
	if state.LSD != "" {
		headers.Set("x-fb-lsd", state.LSD)
	}
	_, data, err := b.client.GetHTTP().MakeRequest(ctx, b.client.GetEndpoint("graphql"), http.MethodPost, headers, []byte(form.Encode()), metaTypes.FORM)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := webapi.DecodeJSONObject(data, true, &result); err != nil {
		return nil, err
	}
	if err := facebookErrorEnvelope(result); err != nil {
		return nil, err
	}
	if err := graphQLError(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (b *messagixBackend) graphQLBatch(ctx context.Context, queries map[string]any) ([]json.RawMessage, error) {
	form, state, err := b.baseForm(ctx)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(queries)
	if err != nil {
		return nil, err
	}
	form.Set("queries", string(encoded))
	headers := b.graphQLHeaders("")
	if state.LSD != "" {
		headers.Set("x-fb-lsd", state.LSD)
	}
	_, data, err := b.client.GetHTTP().MakeRequest(ctx, protocol.GraphQLBatchURL, http.MethodPost, headers, []byte(form.Encode()), metaTypes.FORM)
	if err != nil {
		return nil, err
	}
	return graphql.DecodeBatch(data)
}

func (b *messagixBackend) graphQLHeaders(friendlyName string) http.Header {
	headers := make(http.Header)
	headers.Set("accept", "*/*")
	headers.Set("accept-language", "en-US,en;q=0.9")
	headers.Set("origin", protocol.FacebookBaseURL)
	headers.Set("referer", protocol.FacebookBaseURL+"/messages/")
	headers.Set("user-agent", useragent.UserAgent)
	headers.Set("sec-ch-ua", useragent.SecCHUserAgent)
	headers.Set("sec-ch-ua-mobile", useragent.SecCHMobile)
	headers.Set("sec-ch-ua-platform", useragent.SecCHPlatform)
	headers.Set("sec-fetch-dest", "empty")
	headers.Set("sec-fetch-mode", "cors")
	headers.Set("sec-fetch-site", "same-origin")
	if cookie := b.client.GetCookies().String(); cookie != "" {
		headers.Set("cookie", cookie)
	}
	if friendlyName != "" {
		headers.Set("x-fb-friendly-name", friendlyName)
	}
	return headers
}

func facebookErrorEnvelope(result map[string]any) error {
	if result["error"] == nil && result["errorDescription"] == nil && result["errorSummary"] == nil {
		return nil
	}
	description := firstString(result, "errorDescription", "errorSummary")
	if description == "" {
		description = "Facebook request failed"
	}
	return fmt.Errorf("%s (code %s)", description, stringValue(result["error"]))
}

func graphQLError(result map[string]any) error {
	errorsValue, ok := result["errors"].([]any)
	if !ok || len(errorsValue) == 0 {
		return nil
	}
	if first, ok := errorsValue[0].(map[string]any); ok {
		if message, ok := first["message"].(string); ok && message != "" {
			return errors.New(message)
		}
	}
	return errors.New("Facebook GraphQL request failed")
}

func (b *messagixBackend) ListMessageRequests(ctx context.Context) ([]MessageRequest, error) {
	queries := map[string]any{"o0": map[string]any{"doc_id": protocol.MessageRequestsDocID, "query_params": map[string]any{"limit": 100, "before": nil, "tags": []string{"PENDING"}, "includeDeliveryReceipts": false, "includeSeqID": true}}}
	items, err := b.graphQLBatch(ctx, queries)
	if err != nil {
		return nil, err
	}
	seenKeys := make([]string, 0)
	for _, item := range items {
		var root map[string]any
		if err := json.Unmarshal(item, &root); err != nil {
			return nil, err
		}
		if err := facebookErrorEnvelope(root); err != nil {
			return nil, err
		}
		result := root
		for key := range root {
			seenKeys = append(seenKeys, key)
		}
		if wrapped, ok := root["o0"].(map[string]any); ok {
			result = wrapped
		}
		if _, hasData := result["data"]; !hasData {
			continue
		}
		if err := graphQLError(result); err != nil {
			return nil, err
		}
		data := mapAt(result, "data", "viewer", "message_threads")
		nodes, _ := data["nodes"].([]any)
		requests := make([]MessageRequest, 0, len(nodes))
		for _, rawThread := range nodes {
			thread, ok := rawThread.(map[string]any)
			if !ok {
				continue
			}
			lastMessage := mapAt(thread, "last_message")
			messages, _ := lastMessage["nodes"].([]any)
			if len(messages) == 0 {
				continue
			}
			message, ok := messages[0].(map[string]any)
			if !ok {
				continue
			}
			actor := mapAt(message, "message_sender", "messaging_actor")
			requests = append(requests, MessageRequest{SenderID: model.ID(stringValue(actor["id"])), Snippet: stringValue(message["snippet"]), Timestamp: parseFacebookTimestamp(stringValue(message["timestamp_precise"]))})
		}
		return requests, nil
	}
	sort.Strings(seenKeys)
	return nil, fmt.Errorf("message requests response did not contain thread data (keys: %s)", strings.Join(seenKeys, ","))
}

func (b *messagixBackend) ListThemes(ctx context.Context) ([]Theme, error) {
	result, err := b.graphQL(ctx, protocol.ThemeListFriendlyName, protocol.ThemeListDocID, map[string]any{"version": "default"})
	if err != nil {
		return nil, err
	}
	data := mapAt(result, "data")
	rawThemes, _ := data["messenger_thread_themes"].([]any)
	themes := make([]Theme, 0, len(rawThemes))
	for _, raw := range rawThemes {
		themeMap, ok := raw.(map[string]any)
		if !ok || stringValue(themeMap["id"]) == "" {
			continue
		}
		themes = append(themes, normalizeTheme(themeMap))
	}
	return themes, nil
}

func normalizeTheme(value map[string]any) Theme {
	background := mapAt(value, "background_asset", "image")
	icon := mapAt(value, "icon_asset", "image")
	return Theme{
		ID: model.ID(stringValue(value["id"])), Name: stringValue(value["accessibility_label"]), Description: stringValue(value["description"]), AppColorMode: stringValue(value["app_color_mode"]), ComposerBackgroundColor: stringValue(value["composer_background_color"]), BackgroundGradientColors: stringSlice(value["background_gradient_colors"]), TitleBarButtonTintColor: stringValue(value["title_bar_button_tint_color"]), InboundMessageGradientColors: stringSlice(value["inbound_message_gradient_colors"]), TitleBarTextColor: stringValue(value["title_bar_text_color"]), ComposerTintColor: stringValue(value["composer_tint_color"]), TitleBarAttributionColor: stringValue(value["title_bar_attribution_color"]), ComposerInputBackgroundColor: stringValue(value["composer_input_background_color"]), HotLikeColor: stringValue(value["hot_like_color"]), BackgroundImage: stringValue(background["uri"]), MessageTextColor: stringValue(value["message_text_color"]), InboundMessageTextColor: stringValue(value["inbound_message_text_color"]), PrimaryButtonBackgroundColor: stringValue(value["primary_button_background_color"]), TitleBarBackgroundColor: stringValue(value["title_bar_background_color"]), TertiaryTextColor: stringValue(value["tertiary_text_color"]), ReactionPillBackgroundColor: stringValue(value["reaction_pill_background_color"]), SecondaryTextColor: stringValue(value["secondary_text_color"]), FallbackColor: stringValue(value["fallback_color"]), GradientColors: stringSlice(value["gradient_colors"]), NormalThemeID: model.ID(stringValue(value["normal_theme_id"])), IconAsset: stringValue(icon["uri"]),
	}
}

type themeTask struct {
	ThreadKey    int64 `json:"thread_key"`
	ThemeFBID    int64 `json:"theme_fbid"`
	SyncGroup    int64 `json:"sync_group"`
	Source       any   `json:"source,omitempty"`
	Payload      any   `json:"payload,omitempty"`
	label        string
	queue        string
	includeNulls bool
}

func (t *themeTask) GetLabel() string { return t.label }
func (t *themeTask) Create() (any, string) {
	if !t.includeNulls {
		return t, t.queue
	}
	return struct {
		ThreadKey int64 `json:"thread_key"`
		ThemeFBID int64 `json:"theme_fbid"`
		SyncGroup int64 `json:"sync_group"`
		Source    any   `json:"source"`
		Payload   any   `json:"payload"`
	}{ThreadKey: t.ThreadKey, ThemeFBID: t.ThemeFBID, SyncGroup: t.SyncGroup}, t.queue
}

func (b *messagixBackend) SetTheme(ctx context.Context, threadID, themeID model.ID) error {
	thread, err := strconv.ParseInt(threadID.String(), 10, 64)
	if err != nil || thread == 0 {
		return fmt.Errorf("invalid thread ID %q", threadID)
	}
	theme, err := strconv.ParseInt(themeID.String(), 10, 64)
	if err != nil || theme == 0 {
		return fmt.Errorf("invalid theme ID %q", themeID)
	}
	tasks := []struct {
		label, queue string
		includeNulls bool
	}{{"1013", "ai_generated_theme", false}, {"1037", "msgr_custom_thread_theme", false}, {"1028", "thread_theme_writer", false}, {"43", "thread_theme", true}}
	for _, task := range tasks {
		if _, err := b.client.ExecuteTasks(ctx, &themeTask{ThreadKey: thread, ThemeFBID: theme, SyncGroup: 1, label: task.label, queue: task.queue, includeNulls: task.includeNulls}); err != nil {
			return err
		}
	}
	return nil
}

func (b *messagixBackend) CurrentNote(ctx context.Context) (*Note, error) {
	result, err := b.graphQL(ctx, protocol.NoteCheckFriendlyName, protocol.NoteCheckDocID, map[string]any{"scale": 2})
	if err != nil {
		return nil, err
	}
	noteValue := mapAt(result, "data", "viewer", "actor")["msgr_user_rich_status"]
	if noteValue == nil {
		return nil, nil
	}
	noteMap, ok := noteValue.(map[string]any)
	if !ok {
		return nil, errors.New("invalid note response")
	}
	return normalizeNote(noteMap), nil
}

func (b *messagixBackend) CreateNote(ctx context.Context, text, privacy string) (*Note, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("note text is required")
	}
	privacy = normalizePrivacy(privacy)
	sessionID, err := webapi.NewSessionID()
	if err != nil {
		return nil, err
	}
	account, err := b.client.GetCurrentAccount()
	if err != nil {
		return nil, err
	}
	variables := map[string]any{"input": map[string]any{"client_mutation_id": strconv.FormatInt(time.Now().UnixNano(), 10), "actor_id": strconv.FormatInt(account.GetFBID(), 10), "description": text, "duration": 86400, "note_type": "TEXT_NOTE", "privacy": privacy, "session_id": strconv.FormatUint(sessionID, 10)}}
	result, err := b.graphQL(ctx, protocol.NoteCreateFriendlyName, protocol.NoteCreateDocID, variables)
	if err != nil {
		return nil, err
	}
	status := mapAt(result, "data", "xfb_rich_status_create")["status"]
	noteMap, ok := status.(map[string]any)
	if !ok {
		return nil, errors.New("note create response did not contain status")
	}
	return normalizeNote(noteMap), nil
}

func (b *messagixBackend) DeleteNote(ctx context.Context, noteID model.ID) error {
	if noteID.Empty() {
		return errors.New("note ID is required")
	}
	account, err := b.client.GetCurrentAccount()
	if err != nil {
		return err
	}
	variables := map[string]any{"input": map[string]any{"client_mutation_id": strconv.FormatInt(time.Now().UnixNano(), 10), "actor_id": strconv.FormatInt(account.GetFBID(), 10), "rich_status_id": noteID.String()}}
	result, err := b.graphQL(ctx, protocol.NoteDeleteFriendlyName, protocol.NoteDeleteDocID, variables)
	if err != nil {
		return err
	}
	if mapAt(result, "data")["xfb_rich_status_delete"] == nil {
		return errors.New("note delete response did not contain deletion status")
	}
	return nil
}

func normalizePrivacy(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" || normalized == "PUBLIC" || normalized == "EVERYONE" {
		return "FRIENDS"
	}
	return normalized
}

func normalizeNote(value map[string]any) *Note {
	return &Note{ID: model.ID(firstString(value, "id", "rich_status_id")), Description: firstString(value, "description", "text")}
}

func parseFacebookTimestamp(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if milliseconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.UnixMilli(milliseconds)
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	return time.Time{}
}

func mapAt(root map[string]any, path ...string) map[string]any {
	current := root
	for _, key := range path {
		next, ok := current[key].(map[string]any)
		if !ok {
			return map[string]any{}
		}
		current = next
	}
	return current
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return ""
	}
}

func firstString(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if result := stringValue(value[key]); result != "" {
			return result
		}
	}
	return ""
}

func stringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text := stringValue(item); text != "" {
			result = append(result, text)
		}
	}
	return result
}
