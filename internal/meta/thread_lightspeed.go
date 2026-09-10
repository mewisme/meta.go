package meta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"go.mewis.me/meta-extra/pkg/messagix"
	metaHTTP "go.mewis.me/meta-extra/pkg/messagix/httpclient"
	"go.mewis.me/meta-extra/pkg/messagix/socket"
	"go.mewis.me/meta-extra/pkg/messagix/table"

	"go.mewis.me/meta.go/model"
)

func (b *messagixBackend) CreatePoll(ctx context.Context, threadID model.ID, question string, options []string) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.CreatePollTask{ThreadKey: thread, QuestionText: question, Options: options, SyncGroup: 1})
	return err
}

func (b *messagixBackend) VotePoll(ctx context.Context, threadID, pollID model.ID, optionIDs []model.ID) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	poll, err := parsePositiveID(pollID, "poll")
	if err != nil {
		return err
	}
	selected := make([]int64, len(optionIDs))
	for i, optionID := range optionIDs {
		selected[i], err = parsePositiveID(optionID, "poll option")
		if err != nil {
			return err
		}
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.UpdatePollTask{ThreadKey: thread, PollID: poll, SelectedOptions: selected, SyncGroup: 1})
	return err
}

func (b *messagixBackend) ListPinnedMessages(_ context.Context, threadID model.ID) ([]model.PinnedMessage, error) {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return nil, err
	}
	items, err := b.client.ListPinnedMessages(thread)
	if err != nil {
		return nil, err
	}
	result := make([]model.PinnedMessage, 0, len(items))
	for _, item := range items {
		result = append(result, model.PinnedMessage{ThreadID: id64(item.ThreadKey), MessageID: model.ID(item.MessageID), PinnedAt: unixMilli(item.PinnedTimestampMS), AuthorityLevel: item.AuthorityLevel})
	}
	return result, nil
}

func (b *messagixBackend) FetchPollDetails(ctx context.Context, pollID model.ID) (*model.PollDetails, error) {
	poll, err := parsePositiveID(pollID, "poll")
	if err != nil {
		return nil, err
	}
	details, err := b.client.FetchPollDetails(ctx, poll)
	if err != nil {
		return nil, err
	}
	return normalizePollDetails(details), nil
}

func normalizePollDetails(details *messagix.PollDetails) *model.PollDetails {
	if details == nil {
		return nil
	}
	result := &model.PollDetails{ID: id64(details.ID), ThreadID: id64(details.ThreadKey), Title: details.Title, LastUpdateMessageID: model.ID(details.LastUpdateMessageID), LastUpdateMessageTimestamp: unixMilli(details.LastUpdateMessageTimestampMS), LastUpdateMessageEventType: details.LastUpdateMessageEventType, Options: make([]model.PollOption, 0, len(details.Options)), Votes: make([]model.PollVote, 0, len(details.Votes))}
	for _, option := range details.Options {
		result.Options = append(result.Options, model.PollOption{ID: id64(option.ID), Text: option.Text, SortKeyVotingTimestamp: unixMilli(option.SortKeyVotingTimestamp), SortKeyCreationTimestamp: unixMilli(option.SortKeyCreationTimestamp)})
	}
	for _, vote := range details.Votes {
		result.Votes = append(result.Votes, model.PollVote{OptionID: id64(vote.OptionID), ContactID: id64(vote.ContactID), Timestamp: unixMilli(vote.TimestampMS), VoteCount: vote.VoteCount, ThreadID: id64(vote.ThreadKey), MessageID: model.ID(vote.MessageID)})
	}
	return result
}

func (b *messagixBackend) SearchThreadMessages(ctx context.Context, req model.MessageSearchRequest) (*model.MessageSearchPage, error) {
	thread, err := parsePositiveID(req.ThreadID, "thread")
	if err != nil {
		return nil, err
	}
	page, err := b.client.SearchMessages(ctx, messagix.MessageSearchRequest{ThreadKey: thread, Query: req.Query, Cursor: req.Cursor})
	if err != nil {
		return nil, err
	}
	return normalizeMessageSearchPage(page), nil
}

func normalizeMessageSearchPage(page *messagix.MessageSearchPage) *model.MessageSearchPage {
	if page == nil {
		return nil
	}
	result := &model.MessageSearchPage{Results: make([]model.MessageSearchResult, 0, len(page.Results)), ResultCount: page.ResultCount, HasNextPage: page.HasNextPage, NextCursor: page.NextCursor}
	for _, item := range page.Results {
		highlights := make([]model.MessageSearchHighlight, 0, len(item.Highlights))
		for _, highlight := range item.Highlights {
			highlights = append(highlights, model.MessageSearchHighlight{Offset: highlight.Offset, Length: highlight.Length})
		}
		result.Results = append(result.Results, model.MessageSearchResult{MessageID: model.ID(item.MessageID), ThreadID: id64(item.ThreadKey), ThreadType: messengerThreadType(item.ThreadType), GlobalIndex: item.GlobalIndex, SenderName: item.SenderName, SenderAvatarURL: item.SenderAvatarURL, Timestamp: unixMilli(item.TimestampMS), Text: item.Text, Highlights: highlights})
	}
	return result
}

func messengerThreadType(value table.ThreadType) string {
	switch value {
	case table.ONE_TO_ONE:
		return "oneToOne"
	case table.GROUP_THREAD:
		return "group"
	case table.ROOM:
		return "room"
	case table.MARKETPLACE:
		return "marketplace"
	case table.AI_BOT:
		return "aiBot"
	default:
		return "unknown"
	}
}

func unixMilli(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(value)
}

func (b *messagixBackend) MuteThread(ctx context.Context, threadID model.ID, duration time.Duration) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	millis := duration.Milliseconds()
	if duration < 0 {
		millis = -1000
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.MuteThreadTask{ThreadKey: thread, MuteExpireTimeMS: millis, SyncGroup: 1})
	return err
}

func (b *messagixBackend) MuteThreadCalls(ctx context.Context, threadID model.ID, duration time.Duration) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	_, err = b.client.SetThreadCallsMute(ctx, thread, muteExpiry(duration))
	return err
}

func (b *messagixBackend) SetThreadApprovalMode(ctx context.Context, threadID model.ID, enabled bool) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	_, err = b.client.SetThreadApprovalMode(ctx, thread, enabled)
	return err
}

func (b *messagixBackend) SetThreadArchived(ctx context.Context, threadID model.ID, archived bool) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	_, err = b.client.SetThreadArchived(ctx, thread, archived)
	return err
}

func (b *messagixBackend) SetMessagePinned(ctx context.Context, threadID, messageID model.ID, pinned bool) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	if messageID.Empty() {
		return errors.New("message ID is required")
	}
	_, err = b.client.SetMessagePinned(ctx, thread, messageID.String(), pinned)
	return err
}

func muteExpiry(duration time.Duration) int64 {
	if duration < 0 {
		return -1
	}
	if duration == 0 {
		return 0
	}
	return time.Now().Add(duration).UnixMilli()
}

func (b *messagixBackend) SetThreadPhoto(ctx context.Context, threadID model.ID, input model.AttachmentInput) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	if input.Reader == nil {
		return errors.New("nil thread photo reader")
	}
	if input.Size > maxUploadBytes {
		return fmt.Errorf("thread photo exceeds %d-byte limit", maxUploadBytes)
	}
	data, err := io.ReadAll(io.LimitReader(input.Reader, maxUploadBytes+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > maxUploadBytes {
		return fmt.Errorf("thread photo exceeds %d-byte limit", maxUploadBytes)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "group_photo.jpg"
	}
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "image/jpeg"
	}
	response, err := b.client.GetHTTP().SendMercuryUploadRequest(ctx, thread, &metaHTTP.MercuryUploadMedia{Filename: name, MimeType: contentType, MediaData: data})
	if err != nil {
		return err
	}
	var imageID int64
	if response != nil && response.Payload.RealMetadata != nil {
		imageID = response.Payload.RealMetadata.GetFbId()
	}
	if imageID == 0 {
		return errors.New("thread photo upload returned empty image ID")
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.SetThreadImageTask{ThreadKey: thread, ImageID: imageID, SyncGroup: 1})
	return err
}

func (b *messagixBackend) DeleteThread(ctx context.Context, threadID model.ID) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.DeleteThreadTask{ThreadKey: thread, SyncGroup: 1})
	return err
}

func (b *messagixBackend) CreateDM(ctx context.Context, userID model.ID) (model.ID, error) {
	user, err := parsePositiveID(userID, "user")
	if err != nil {
		return "", err
	}
	tbl, err := b.client.ExecuteTasks(ctx, &socket.CreateThreadTask{ThreadFBID: user, ForceUpsert: 1, SyncGroup: 1})
	if err != nil {
		return "", err
	}
	if tbl != nil && len(tbl.LSDeleteThenInsertThread) > 0 && tbl.LSDeleteThenInsertThread[0] != nil && tbl.LSDeleteThenInsertThread[0].ThreadKey != 0 {
		return id64(tbl.LSDeleteThenInsertThread[0].ThreadKey), nil
	}
	return userID, nil
}

func (b *messagixBackend) SearchMessengerUsers(ctx context.Context, query string) ([]model.User, error) {
	tbl, err := b.client.ExecuteTasks(ctx, &socket.SearchUserTask{Query: query, SupportedTypes: []table.SearchType{table.SearchTypeContact, table.SearchTypeNonContact}, SurfaceType: 15})
	if err != nil {
		return nil, err
	}
	if tbl == nil {
		return []model.User{}, nil
	}
	users := make([]model.User, 0, len(tbl.LSInsertSearchResult))
	for _, result := range tbl.LSInsertSearchResult {
		if result == nil || strings.TrimSpace(result.ResultId) == "" {
			continue
		}
		users = append(users, model.User{ID: model.ID(result.ResultId), Name: result.DisplayName, AvatarURL: result.ProfilePicUrl, IsVerified: result.IsVerified, CanViewerMessage: result.CanViewerMessage})
	}
	return users, nil
}

func (b *messagixBackend) GetMessengerContact(ctx context.Context, userID model.ID) (*model.User, error) {
	user, err := parsePositiveID(userID, "user")
	if err != nil {
		return nil, err
	}
	tbl, err := b.client.ExecuteTasks(ctx, &socket.GetContactsFullTask{ContactID: user})
	if err != nil {
		return nil, err
	}
	if tbl != nil {
		for _, contact := range tbl.LSDeleteThenInsertContact {
			if contact != nil && contact.Id == user {
				return &model.User{ID: id64(contact.Id), Name: contact.Name, FirstName: contact.FirstName, Username: contact.Username, AvatarURL: contact.GetAvatarURL(), Gender: messengerGender(contact.Gender), IsMessengerUser: contact.IsMessengerUser, CanViewerMessage: contact.CanViewerMessage}, nil
			}
		}
	}
	return nil, fmt.Errorf("messenger contact %s not found", userID)
}

func messengerGender(gender table.Gender) string {
	switch gender {
	case table.FEMALE_SINGULAR, table.FEMALE_SINGULAR_GUESS, table.FEMALE_PLURAL:
		return "female"
	case table.MALE_SINGULAR, table.MALE_SINGULAR_GUESS, table.MALE_PLURAL:
		return "male"
	case table.NEUTER_SINGULAR, table.NEUTER_PLURAL:
		return "neutral"
	default:
		return ""
	}
}

func parsePositiveID(value model.ID, name string) (int64, error) {
	id, err := strconv.ParseInt(value.String(), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s ID %q", name, value)
	}
	return id, nil
}
