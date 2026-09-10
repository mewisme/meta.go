package server

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	fberrors "go.mewis.me/meta.go/errors"
	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/model"
)

func sendRequestFromProto(req *metav1.SendRequest) model.SendRequest {
	result := model.SendRequest{ThreadID: model.ID(req.GetThreadId()), Text: req.GetText(), StickerID: model.ID(req.GetStickerId()), URL: req.GetUrl(), Encryption: encryptionPolicyFromProto(req.GetEncryption())}
	if reply := req.GetReplyTo(); reply != nil {
		result.ReplyTo = &model.ReplyReference{MessageID: model.ID(reply.GetMessageId()), SenderID: model.ID(reply.GetSenderId())}
	}
	result.Mentions = make([]model.Mention, 0, len(req.GetMentions()))
	for _, mention := range req.GetMentions() {
		result.Mentions = append(result.Mentions, model.Mention{UserID: model.ID(mention.GetUserId()), Offset: int(mention.GetOffset()), Length: int(mention.GetLength())})
	}
	return result
}

func encryptionPolicyFromProto(value metav1.EncryptionPolicy) model.EncryptionPolicy {
	switch value {
	case metav1.EncryptionPolicy_ENCRYPTION_POLICY_REQUIRED:
		return model.EncryptionRequired
	case metav1.EncryptionPolicy_ENCRYPTION_POLICY_DISABLED:
		return model.EncryptionDisabled
	default:
		return model.EncryptionAuto
	}
}

func userToProto(user model.User) *metav1.User {
	return &metav1.User{Id: user.ID.String(), Name: user.Name, FirstName: user.FirstName, Username: user.Username, ProfileUrl: user.ProfileURL, AvatarUrl: user.AvatarURL, Gender: user.Gender, IsMessengerUser: user.IsMessengerUser, IsVerified: user.IsVerified, CanViewerMessage: user.CanViewerMessage, MessengerRestricted: user.MessengerRestricted, MessengerBlockStatus: messengerBlockStatusToProto(user.MessengerBlockStatus)}
}

func messengerBlockStatusToProto(value model.MessengerBlockStatus) metav1.MessengerBlockStatus {
	switch value {
	case model.MessengerBlockUnblocked:
		return metav1.MessengerBlockStatus_MESSENGER_BLOCK_STATUS_UNBLOCKED
	case model.MessengerBlockMessageBlocked:
		return metav1.MessengerBlockStatus_MESSENGER_BLOCK_STATUS_MESSAGE_BLOCKED
	case model.MessengerBlockFullyBlocked:
		return metav1.MessengerBlockStatus_MESSENGER_BLOCK_STATUS_FULLY_BLOCKED
	case model.MessengerBlockUnknown:
		return metav1.MessengerBlockStatus_MESSENGER_BLOCK_STATUS_UNKNOWN
	default:
		return metav1.MessengerBlockStatus_MESSENGER_BLOCK_STATUS_UNSPECIFIED
	}
}

func threadToProto(thread model.Thread) *metav1.Thread {
	result := &metav1.Thread{Id: thread.ID.String(), Name: thread.Name, Type: thread.Type, Emoji: thread.Emoji, MessageCount: thread.MessageCount, ApprovalMode: thread.ApprovalMode, Joinable: thread.Joinable, JoinableUrl: thread.JoinableURL, LastActivity: timestampOrNil(thread.LastActivity), Nicknames: make(map[string]string, len(thread.Nicknames))}
	result.Participants = make([]*metav1.User, 0, len(thread.Participants))
	for _, user := range thread.Participants {
		result.Participants = append(result.Participants, userToProto(user))
	}
	result.Admins = make([]string, len(thread.Admins))
	for i, id := range thread.Admins {
		result.Admins[i] = id.String()
	}
	for id, nickname := range thread.Nicknames {
		result.Nicknames[id.String()] = nickname
	}
	return result
}

func pinnedMessageToProto(message model.PinnedMessage) *metav1.PinnedMessage {
	return &metav1.PinnedMessage{ThreadId: message.ThreadID.String(), MessageId: message.MessageID.String(), PinnedAt: timestampOrNil(message.PinnedAt), AuthorityLevel: message.AuthorityLevel}
}

func pollDetailsToProto(poll *model.PollDetails) *metav1.PollDetails {
	if poll == nil {
		return nil
	}
	result := &metav1.PollDetails{Id: poll.ID.String(), ThreadId: poll.ThreadID.String(), Title: poll.Title, LastUpdateMessageId: poll.LastUpdateMessageID.String(), LastUpdateMessageTimestamp: timestampOrNil(poll.LastUpdateMessageTimestamp), LastUpdateMessageEventType: poll.LastUpdateMessageEventType}
	result.Options = make([]*metav1.PollOption, 0, len(poll.Options))
	for _, option := range poll.Options {
		result.Options = append(result.Options, &metav1.PollOption{Id: option.ID.String(), Text: option.Text, SortKeyVotingTimestamp: timestampOrNil(option.SortKeyVotingTimestamp), SortKeyCreationTimestamp: timestampOrNil(option.SortKeyCreationTimestamp)})
	}
	result.Votes = make([]*metav1.PollVote, 0, len(poll.Votes))
	for _, vote := range poll.Votes {
		result.Votes = append(result.Votes, &metav1.PollVote{OptionId: vote.OptionID.String(), ContactId: vote.ContactID.String(), Timestamp: timestampOrNil(vote.Timestamp), VoteCount: vote.VoteCount, ThreadId: vote.ThreadID.String(), MessageId: vote.MessageID.String()})
	}
	return result
}

func messageSearchResultToProto(result model.MessageSearchResult) *metav1.MessageSearchResult {
	out := &metav1.MessageSearchResult{MessageId: result.MessageID.String(), ThreadId: result.ThreadID.String(), ThreadType: result.ThreadType, GlobalIndex: result.GlobalIndex, SenderName: result.SenderName, SenderAvatarUrl: result.SenderAvatarURL, Timestamp: timestampOrNil(result.Timestamp), Text: result.Text}
	out.Highlights = make([]*metav1.MessageSearchHighlight, 0, len(result.Highlights))
	for _, highlight := range result.Highlights {
		out.Highlights = append(out.Highlights, &metav1.MessageSearchHighlight{Offset: int32(highlight.Offset), Length: int32(highlight.Length)})
	}
	return out
}

func messageRequestToProto(request model.MessageRequest) *metav1.MessageRequest {
	return &metav1.MessageRequest{SenderId: request.SenderID.String(), Snippet: request.Snippet, Timestamp: timestampOrNil(request.Timestamp)}
}

func themeToProto(theme *model.Theme) *metav1.Theme {
	if theme == nil {
		return nil
	}
	return &metav1.Theme{Id: theme.ID.String(), Name: theme.Name, Description: theme.Description, AppColorMode: theme.AppColorMode, ComposerBackgroundColor: theme.ComposerBackgroundColor, BackgroundGradientColors: append([]string(nil), theme.BackgroundGradientColors...), TitleBarButtonTintColor: theme.TitleBarButtonTintColor, InboundMessageGradientColors: append([]string(nil), theme.InboundMessageGradientColors...), TitleBarTextColor: theme.TitleBarTextColor, ComposerTintColor: theme.ComposerTintColor, TitleBarAttributionColor: theme.TitleBarAttributionColor, ComposerInputBackgroundColor: theme.ComposerInputBackgroundColor, HotLikeColor: theme.HotLikeColor, BackgroundImage: theme.BackgroundImage, MessageTextColor: theme.MessageTextColor, InboundMessageTextColor: theme.InboundMessageTextColor, PrimaryButtonBackgroundColor: theme.PrimaryButtonBackgroundColor, TitleBarBackgroundColor: theme.TitleBarBackgroundColor, TertiaryTextColor: theme.TertiaryTextColor, ReactionPillBackgroundColor: theme.ReactionPillBackgroundColor, SecondaryTextColor: theme.SecondaryTextColor, FallbackColor: theme.FallbackColor, GradientColors: append([]string(nil), theme.GradientColors...), NormalThemeId: theme.NormalThemeID.String(), IconAsset: theme.IconAsset}
}

func noteToProto(note *model.Note) *metav1.Note {
	if note == nil {
		return nil
	}
	return &metav1.Note{Id: note.ID.String(), Description: note.Description}
}

func idsFromStrings(values []string) []model.ID {
	result := make([]model.ID, len(values))
	for i, value := range values {
		result[i] = model.ID(value)
	}
	return result
}

func durationFromProto(value *durationpb.Duration, indefinitely bool) (time.Duration, error) {
	if indefinitely {
		if value != nil && value.AsDuration() != 0 {
			return 0, fmt.Errorf("%w: duration and indefinitely cannot both be set", fberrors.ErrInvalidInput)
		}
		return -1, nil
	}
	if value == nil {
		return 0, nil
	}
	if err := value.CheckValid(); err != nil {
		return 0, fmt.Errorf("%w: invalid duration", fberrors.ErrInvalidInput)
	}
	duration := value.AsDuration()
	if duration < 0 {
		return 0, fmt.Errorf("%w: duration cannot be negative; use indefinitely", fberrors.ErrInvalidInput)
	}
	return duration, nil
}

func timeFromProto(value *timestamppb.Timestamp, field string) (time.Time, error) {
	if value == nil {
		return time.Time{}, fmt.Errorf("%w: %s is required", fberrors.ErrInvalidInput, field)
	}
	if err := value.CheckValid(); err != nil {
		return time.Time{}, fmt.Errorf("%w: invalid %s", fberrors.ErrInvalidInput, field)
	}
	return value.AsTime(), nil
}
