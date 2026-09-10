package meta

import (
	"context"
	"strings"
	"time"

	"go.mewis.me/meta-extra/pkg/messagix/socket"

	"go.mewis.me/meta.go/model"
)

func (b *messagixBackend) ShareContact(ctx context.Context, threadID, contactID model.ID, text string) error {
	thread, err := parsePositiveID(threadID, "thread")
	if err != nil {
		return err
	}
	contact, err := parsePositiveID(contactID, "contact")
	if err != nil {
		return err
	}
	if err := b.client.WaitUntilCanSendMessages(ctx, 10*time.Second); err != nil {
		return err
	}
	var textPtr *string
	if strings.TrimSpace(text) != "" {
		textPtr = &text
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.ShareContactTask{ContactID: contact, SyncGroup: 1, Text: textPtr, ThreadID: thread})
	return err
}

func (b *messagixBackend) SetRestricted(ctx context.Context, userID model.ID, restricted bool) error {
	user, err := parsePositiveID(userID, "user")
	if err != nil {
		return err
	}
	action := socket.MessengerRestrictActionUnrestrict
	if restricted {
		action = socket.MessengerRestrictActionRestrict
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.SetMessengerRestrictTask{RestricteeID: user, Action: action})
	return err
}

func (b *messagixBackend) SetMessageBlocked(ctx context.Context, userID model.ID, blocked bool) error {
	user, err := parsePositiveID(userID, "user")
	if err != nil {
		return err
	}
	status := socket.MessengerBlockStatusUnblocked
	if blocked {
		status = socket.MessengerBlockStatusMessageBlocked
	}
	_, err = b.client.ExecuteTasks(ctx, &socket.SetMessengerBlockStatusTask{BlockeeID: user, BlockedByViewerStatus: status})
	return err
}
