package fbgo

import "go.mewis.me/fbgo/model"

type TransportKind = model.TransportKind
type EncryptionKind = model.EncryptionKind
type EncryptionPolicy = model.EncryptionPolicy
type ConnectionState = model.ConnectionState
type HealthSnapshot = model.HealthSnapshot
type User = model.User
type Thread = model.Thread
type Attachment = model.Attachment
type E2EEMediaKind = model.E2EEMediaKind
type E2EEMediaReference = model.E2EEMediaReference
type E2EEMediaInput = model.E2EEMediaInput
type E2EEMediaDownload = model.E2EEMediaDownload
type ReplyReference = model.ReplyReference
type Mention = model.Mention
type Message = model.Message
type AttachmentInput = model.AttachmentInput
type UploadInput = model.UploadInput
type UploadResult = model.UploadResult
type SendRequest = model.SendRequest
type SendResult = model.SendResult
type E2EESendRequest = model.E2EESendRequest
type E2EEReactionRequest = model.E2EEReactionRequest
type E2EEReadRequest = model.E2EEReadRequest
type ReactionEvent = model.ReactionEvent
type TypingEvent = model.TypingEvent
type ReadReceiptEvent = model.ReadReceiptEvent
type DeliveryReceiptEvent = model.DeliveryReceiptEvent
type MessageEditEvent = model.MessageEditEvent
type MessageUnsendEvent = model.MessageUnsendEvent
type ThreadUpdateEvent = model.ThreadUpdateEvent
type E2EEReceiptEvent = model.E2EEReceiptEvent
type MessageRequest = model.MessageRequest
type Theme = model.Theme
type Note = model.Note
type EventKind = model.EventKind
type Event = model.Event

const (
	TransportUnknown       = model.TransportUnknown
	TransportMessenger     = model.TransportMessenger
	TransportE2EE          = model.TransportE2EE
	EncryptionNone         = model.EncryptionNone
	EncryptionE2EE         = model.EncryptionE2EE
	EncryptionAuto         = model.EncryptionAuto
	EncryptionRequired     = model.EncryptionRequired
	EncryptionDisabled     = model.EncryptionDisabled
	E2EEMediaImage         = model.E2EEMediaImage
	E2EEMediaVideo         = model.E2EEMediaVideo
	E2EEMediaAudio         = model.E2EEMediaAudio
	E2EEMediaDocument      = model.E2EEMediaDocument
	E2EEMediaSticker       = model.E2EEMediaSticker
	ConnectionDisconnected = model.ConnectionDisconnected
	ConnectionConnecting   = model.ConnectionConnecting
	ConnectionConnected    = model.ConnectionConnected
	ConnectionFailed       = model.ConnectionFailed
	EventReady             = model.EventReady
	EventReconnected       = model.EventReconnected
	EventDisconnected      = model.EventDisconnected
	EventError             = model.EventError
	EventMessage           = model.EventMessage
	EventMessageEdit       = model.EventMessageEdit
	EventMessageUnsend     = model.EventMessageUnsend
	EventReaction          = model.EventReaction
	EventTyping            = model.EventTyping
	EventReadReceipt       = model.EventReadReceipt
	EventDeliveryReceipt   = model.EventDeliveryReceipt
	EventThreadUpdate      = model.EventThreadUpdate
	EventE2EEReady         = model.EventE2EEReady
	EventE2EEReceipt       = model.EventE2EEReceipt
)

type Handler func(Event)
type UnsubscribeFunc func()
