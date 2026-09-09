package meta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.mewis.me/fbgo/internal/protocol"
	"go.mewis.me/fbgo/internal/webapi"
	"go.mewis.me/fbgo/model"
	"go.mewis.me/meta-extra/pkg/messagix/socket"
	metaTypes "go.mewis.me/meta-extra/pkg/messagix/types"
)

func (b *messagixBackend) ListThreads(ctx context.Context, limit int) (model.ThreadList, error) {
	if limit <= 0 {
		limit = 50
	}
	queries := map[string]any{"o0": map[string]any{"doc_id": protocol.ThreadListDocID, "query_params": map[string]any{"limit": limit, "before": nil, "tags": []string{"INBOX"}, "includeDeliveryReceipts": false, "includeSeqID": true}}}
	items, err := b.graphQLBatch(ctx, queries)
	if err != nil {
		return model.ThreadList{}, err
	}
	for _, item := range items {
		var root map[string]any
		if err := json.Unmarshal(item, &root); err != nil {
			return model.ThreadList{}, err
		}
		if err := facebookErrorEnvelope(root); err != nil {
			return model.ThreadList{}, err
		}
		if wrapped, ok := root["o0"].(map[string]any); ok {
			root = wrapped
		}
		if err := graphQLError(root); err != nil {
			return model.ThreadList{}, err
		}
		threads := mapAt(root, "data", "viewer", "message_threads")
		if len(threads) == 0 {
			continue
		}
		seq, err := strconv.ParseInt(strings.TrimSpace(stringValue(threads["sync_sequence_id"])), 10, 64)
		if err != nil || seq < 0 {
			return model.ThreadList{}, errors.New("thread list response contains invalid sync sequence ID")
		}
		nodes, _ := threads["nodes"].([]any)
		result := model.ThreadList{Threads: make([]model.Thread, 0, len(nodes)), SyncSequenceID: seq}
		for _, raw := range nodes {
			threadMap, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			thread := normalizeThread(threadMap)
			if !thread.ID.Empty() {
				result.Threads = append(result.Threads, thread)
			}
		}
		return result, nil
	}
	return model.ThreadList{}, errors.New("thread list response did not contain thread data")
}

func (b *messagixBackend) GetThread(ctx context.Context, threadID model.ID) (*model.Thread, error) {
	list, err := b.ListThreads(ctx, 100)
	if err != nil {
		return nil, err
	}
	for index := range list.Threads {
		if list.Threads[index].ID == threadID {
			return &list.Threads[index], nil
		}
	}
	return nil, fmt.Errorf("thread %s not found in inbox", threadID)
}

func normalizeThread(value map[string]any) model.Thread {
	key := mapAt(value, "thread_key")
	customization := mapAt(value, "customization_info")
	joinable := mapAt(value, "joinable_mode")
	thread := model.Thread{ID: model.ID(firstString(key, "thread_fbid", "other_user_id")), Name: stringValue(value["name"]), Type: stringValue(value["thread_type"]), Emoji: stringValue(customization["emoji"]), MessageCount: int64Value(value["messages_count"]), ApprovalMode: boolValue(value["approval_mode"]), JoinableURL: stringValue(joinable["link"]), LastActivity: parseFacebookTimestamp(firstString(value, "updated_time_precise", "last_message_timestamp"))}
	thread.Joinable = stringValue(joinable["mode"]) != "" && stringValue(joinable["mode"]) != "0"
	if customizations, ok := customization["participant_customizations"].([]any); ok {
		thread.Nicknames = make(map[model.ID]string)
		for _, raw := range customizations {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			id := model.ID(firstString(item, "participant_id", "participantId"))
			if id.Empty() {
				continue
			}
			thread.Nicknames[id] = stringValue(item["nickname"])
		}
		if len(thread.Nicknames) == 0 {
			thread.Nicknames = nil
		}
	}
	if admins, ok := value["thread_admins"].([]any); ok {
		thread.Admins = make([]model.ID, 0, len(admins))
		for _, raw := range admins {
			if admin, ok := raw.(map[string]any); ok {
				if id := stringValue(admin["id"]); id != "" {
					thread.Admins = append(thread.Admins, model.ID(id))
				}
			}
		}
	}
	if edges, ok := mapAt(value, "all_participants")["edges"].([]any); ok {
		thread.Participants = make([]model.User, 0, len(edges))
		for _, raw := range edges {
			edge, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			actor := mapAt(edge, "node", "messaging_actor")
			id := stringValue(actor["id"])
			if id == "" {
				continue
			}
			image := mapAt(actor, "big_image_src")
			thread.Participants = append(thread.Participants, model.User{ID: model.ID(id), Name: stringValue(actor["name"]), Username: stringValue(actor["username"]), ProfileURL: stringValue(actor["url"]), AvatarURL: stringValue(image["uri"]), Gender: stringValue(actor["gender"])})
		}
	}
	return thread
}

func (b *messagixBackend) SetThreadAdmin(ctx context.Context, threadID, userID model.ID, admin bool) error {
	threadKey, err := strconv.ParseInt(threadID.String(), 10, 64)
	if err != nil || threadKey <= 0 {
		return errors.New("invalid thread ID for admin update")
	}
	contactID, err := strconv.ParseInt(userID.String(), 10, 64)
	if err != nil || contactID <= 0 {
		return errors.New("invalid user ID for admin update")
	}
	isAdmin := 0
	if admin {
		isAdmin = 1
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.UpdateAdminTask{ThreadKey: threadKey, ContactID: contactID, IsAdmin: isAdmin})
	return err
}

func (b *messagixBackend) SetThreadEmoji(ctx context.Context, threadID model.ID, emoji string) error {
	form, _, err := b.baseForm(ctx)
	if err != nil {
		return err
	}
	form.Set("emoji_choice", emoji)
	form.Set("thread_or_other_fbid", threadID.String())
	return b.postThreadMutation(ctx, protocol.ThreadEmojiURL, form)
}

func (b *messagixBackend) SetThreadNickname(ctx context.Context, threadID, userID model.ID, nickname string) error {
	form, _, err := b.baseForm(ctx)
	if err != nil {
		return err
	}
	form.Set("nickname", nickname)
	form.Set("participant_id", userID.String())
	form.Set("thread_or_other_fbid", threadID.String())
	return b.postThreadMutation(ctx, protocol.ThreadNicknameURL, form)
}

func (b *messagixBackend) SetThreadName(ctx context.Context, threadID model.ID, name string) error {
	form, state, err := b.baseForm(ctx)
	if err != nil {
		return err
	}
	threadingID, err := webapi.NewThreadingID(time.Now())
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	form.Set("client", "mercury")
	form.Set("author", "fbid:"+state.FBID)
	form.Set("timestamp", strconv.FormatInt(now, 10))
	form.Set("thread_fbid", threadID.String())
	form.Set("thread_name", strings.TrimSpace(name))
	form.Set("thread_id", threadID.String())
	form.Set("source", "source:chat:web")
	form.Set("source_tags[0]", "source:chat")
	form.Set("client_thread_id", "root:"+threadingID)
	form.Set("offline_threading_id", threadingID)
	form.Set("message_id", threadingID)
	form.Set("threading_id", fmt.Sprintf("<%d:%s@mail.projektitan.com>", now, threadingID))
	form.Set("ephemeral_ttl_mode", "0")
	form.Set("manual_retry_cnt", "0")
	form.Set("ui_push_phase", "V3")
	form.Set("log_message_type", "log:thread-name")
	return b.postThreadMutation(ctx, protocol.ThreadNameURL, form)
}

func (b *messagixBackend) postThreadMutation(ctx context.Context, endpoint string, form url.Values) error {
	headers := b.graphQLHeaders("")
	headers.Set("referer", protocol.FacebookBaseURL+"/messages/")
	_, data, err := b.client.GetHTTP().MakeRequest(ctx, endpoint, http.MethodPost, headers, []byte(form.Encode()), metaTypes.FORM)
	if err != nil {
		return err
	}
	var result map[string]any
	if err := webapi.DecodeJSONObject(data, true, &result); err != nil {
		return err
	}
	if err := facebookErrorEnvelope(result); err != nil {
		return err
	}
	return nil
}

func int64Value(value any) int64 {
	text := stringValue(value)
	parsed, _ := strconv.ParseInt(text, 10, 64)
	return parsed
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, _ := strconv.ParseBool(typed)
		return parsed
	case float64:
		return typed != 0
	default:
		return false
	}
}
