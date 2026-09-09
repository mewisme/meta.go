package model

import (
	"io"
	"time"
)

type ID string

func (id ID) String() string { return string(id) }
func (id ID) Empty() bool    { return id == "" }

type TransportKind string

const (
	TransportUnknown   TransportKind = "unknown"
	TransportMessenger TransportKind = "messenger"
	TransportE2EE      TransportKind = "e2ee"
)

type EncryptionKind string

const (
	EncryptionNone EncryptionKind = "none"
	EncryptionE2EE EncryptionKind = "e2ee"
)

type EncryptionPolicy string

const (
	EncryptionAuto     EncryptionPolicy = "auto"
	EncryptionRequired EncryptionPolicy = "required"
	EncryptionDisabled EncryptionPolicy = "disabled"
)

type ConnectionState string

const (
	ConnectionDisconnected ConnectionState = "disconnected"
	ConnectionConnecting   ConnectionState = "connecting"
	ConnectionConnected    ConnectionState = "connected"
	ConnectionFailed       ConnectionState = "failed"
)

type HealthSnapshot struct {
	Regular            ConnectionState `json:"regular"`
	E2EE               ConnectionState `json:"e2ee"`
	ReconnectCount     uint64          `json:"reconnectCount"`
	DroppedEventCount  uint64          `json:"droppedEventCount"`
	LastSuccessfulSend time.Time       `json:"lastSuccessfulSend,omitempty"`
	LastReceive        time.Time       `json:"lastReceive,omitempty"`
}

type User struct {
	ID       ID     `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
}

type Thread struct {
	ID           ID     `json:"id"`
	Name         string `json:"name,omitempty"`
	Type         string `json:"type,omitempty"`
	Participants []User `json:"participants,omitempty"`
	Admins       []ID   `json:"admins,omitempty"`
	Emoji        string `json:"emoji,omitempty"`
}

type Attachment struct {
	ID          ID     `json:"id,omitempty"`
	Type        string `json:"type"`
	URL         string `json:"url,omitempty"`
	PreviewURL  string `json:"previewUrl,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Size        int64  `json:"size,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Duration    int    `json:"duration,omitempty"`
}

type ReplyReference struct {
	MessageID ID `json:"messageId"`
	SenderID  ID `json:"senderId,omitempty"`
}

type Mention struct {
	UserID ID  `json:"userId"`
	Offset int `json:"offset"`
	Length int `json:"length"`
}

type Message struct {
	ID          ID              `json:"id"`
	ThreadID    ID              `json:"threadId"`
	SenderID    ID              `json:"senderId"`
	Text        string          `json:"text,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
	ReplyTo     *ReplyReference `json:"replyTo,omitempty"`
	Mentions    []Mention       `json:"mentions,omitempty"`
	Attachments []Attachment    `json:"attachments,omitempty"`
	Encryption  EncryptionKind  `json:"encryption"`
	Transport   TransportKind   `json:"transport"`
}

type AttachmentInput struct {
	Name        string
	ContentType string
	Reader      io.Reader
	Size        int64
}

type SendRequest struct {
	ThreadID    ID
	Text        string
	ReplyTo     *ReplyReference
	Mentions    []Mention
	Attachments []AttachmentInput
	Encryption  EncryptionPolicy
}

type SendResult struct {
	MessageID ID        `json:"messageId"`
	Timestamp time.Time `json:"timestamp"`
}

type EventKind string

const (
	EventReady         EventKind = "ready"
	EventReconnected   EventKind = "reconnected"
	EventDisconnected  EventKind = "disconnected"
	EventError         EventKind = "error"
	EventMessage       EventKind = "message"
	EventMessageEdit   EventKind = "messageEdit"
	EventMessageUnsend EventKind = "messageUnsend"
	EventReaction      EventKind = "reaction"
	EventTyping        EventKind = "typing"
	EventReadReceipt   EventKind = "readReceipt"
)

type Event struct {
	Kind EventKind `json:"kind"`
	Data any       `json:"data,omitempty"`
}
