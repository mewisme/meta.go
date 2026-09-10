package server

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	metav1 "go.mewis.me/meta.go/gen/go/meta/v1"
	"go.mewis.me/meta.go/model"
)

func accountToProto(user model.User) *metav1.Account {
	return &metav1.Account{Id: user.ID.String(), Name: user.Name, Username: user.Username}
}

func healthToProto(health model.HealthSnapshot, subscriberDropped uint64) *metav1.HealthSnapshot {
	return &metav1.HealthSnapshot{
		Regular:                     connectionStateToProto(health.Regular),
		E2Ee:                        connectionStateToProto(health.E2EE),
		ReconnectCount:              health.ReconnectCount,
		EngineDroppedEventCount:     health.DroppedEventCount,
		SubscriberDroppedEventCount: subscriberDropped,
		LastSuccessfulSend:          timestampOrNil(health.LastSuccessfulSend),
		LastReceive:                 timestampOrNil(health.LastReceive),
		LastErrorCategory:           health.LastErrorCategory,
	}
}

func connectionStateToProto(state model.ConnectionState) metav1.ConnectionState {
	switch state {
	case model.ConnectionDisconnected:
		return metav1.ConnectionState_CONNECTION_STATE_DISCONNECTED
	case model.ConnectionConnecting:
		return metav1.ConnectionState_CONNECTION_STATE_CONNECTING
	case model.ConnectionConnected:
		return metav1.ConnectionState_CONNECTION_STATE_CONNECTED
	case model.ConnectionFailed:
		return metav1.ConnectionState_CONNECTION_STATE_FAILED
	default:
		return metav1.ConnectionState_CONNECTION_STATE_UNSPECIFIED
	}
}

func timestampOrNil(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}
