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
type ReplyReference = model.ReplyReference
type Mention = model.Mention
type Message = model.Message
type AttachmentInput = model.AttachmentInput
type SendRequest = model.SendRequest
type SendResult = model.SendResult
type EventKind = model.EventKind
type Event = model.Event

const (
	TransportUnknown   = model.TransportUnknown
	TransportMessenger = model.TransportMessenger
	TransportE2EE      = model.TransportE2EE
	EncryptionNone     = model.EncryptionNone
	EncryptionE2EE     = model.EncryptionE2EE
	EncryptionAuto     = model.EncryptionAuto
	EncryptionRequired = model.EncryptionRequired
	EncryptionDisabled = model.EncryptionDisabled
	EventReady         = model.EventReady
	EventReconnected   = model.EventReconnected
	EventDisconnected  = model.EventDisconnected
	EventError         = model.EventError
	EventMessage       = model.EventMessage
	EventMessageEdit   = model.EventMessageEdit
	EventMessageUnsend = model.EventMessageUnsend
	EventReaction      = model.EventReaction
	EventTyping        = model.EventTyping
	EventReadReceipt   = model.EventReadReceipt
)

type Handler func(Event)
type UnsubscribeFunc func()
