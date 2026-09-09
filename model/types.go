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
	ID         ID     `json:"id"`
	Name       string `json:"name"`
	Username   string `json:"username,omitempty"`
	ProfileURL string `json:"profileUrl,omitempty"`
	AvatarURL  string `json:"avatarUrl,omitempty"`
	Gender     string `json:"gender,omitempty"`
}

type FacebookUser struct {
	ID            ID     `json:"id"`
	Name          string `json:"name,omitempty"`
	FirstName     string `json:"firstName,omitempty"`
	Username      string `json:"username,omitempty"`
	ProfileURL    string `json:"profileUrl,omitempty"`
	AvatarURL     string `json:"avatarUrl,omitempty"`
	Gender        string `json:"gender,omitempty"`
	AlternateName string `json:"alternateName,omitempty"`
	NonFriend     bool   `json:"nonFriend,omitempty"`
}

type SearchResult struct {
	ID   ID     `json:"id"`
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type Notification struct {
	ID        ID        `json:"id,omitempty"`
	Text      string    `json:"text"`
	URL       string    `json:"url,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type Post struct {
	ID  ID     `json:"id,omitempty"`
	URL string `json:"url,omitempty"`
}

type PostOwnership string

const (
	PostOwned  PostOwnership = "owned"
	PostShared PostOwnership = "shared"
)

type MarketplaceLocation struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Name      string  `json:"name,omitempty"`
}

type MarketplaceListingInput struct {
	Title       string              `json:"title"`
	Brand       string              `json:"brand,omitempty"`
	Price       string              `json:"price"`
	Currency    string              `json:"currency"`
	Description string              `json:"description,omitempty"`
	Hashtags    []string            `json:"hashtags,omitempty"`
	Category    string              `json:"category"`
	PhotoIDs    []ID                `json:"photoIds"`
	Location    MarketplaceLocation `json:"location"`
}

type MarketplaceListing struct {
	ID          ID                  `json:"id"`
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Price       string              `json:"price,omitempty"`
	Currency    string              `json:"currency,omitempty"`
	Seller      FacebookUser        `json:"seller,omitempty"`
	Location    MarketplaceLocation `json:"location,omitempty"`
	URL         string              `json:"url,omitempty"`
	CreatedAt   time.Time           `json:"createdAt,omitempty"`
}

type Thread struct {
	ID           ID            `json:"id"`
	Name         string        `json:"name,omitempty"`
	Type         string        `json:"type,omitempty"`
	Participants []User        `json:"participants,omitempty"`
	Admins       []ID          `json:"admins,omitempty"`
	Nicknames    map[ID]string `json:"nicknames,omitempty"`
	Emoji        string        `json:"emoji,omitempty"`
	MessageCount int64         `json:"messageCount,omitempty"`
	ApprovalMode bool          `json:"approvalMode,omitempty"`
	Joinable     bool          `json:"joinable,omitempty"`
	JoinableURL  string        `json:"joinableUrl,omitempty"`
	LastActivity time.Time     `json:"lastActivity,omitempty"`
}

type ThreadList struct {
	Threads        []Thread `json:"threads"`
	SyncSequenceID int64    `json:"syncSequenceId"`
}

type Attachment struct {
	ID          ID                  `json:"id,omitempty"`
	Type        string              `json:"type"`
	URL         string              `json:"url,omitempty"`
	PreviewURL  string              `json:"previewUrl,omitempty"`
	FileName    string              `json:"fileName,omitempty"`
	ContentType string              `json:"contentType,omitempty"`
	Size        int64               `json:"size,omitempty"`
	Width       int                 `json:"width,omitempty"`
	Height      int                 `json:"height,omitempty"`
	Duration    int                 `json:"duration,omitempty"`
	E2EE        *E2EEMediaReference `json:"e2ee,omitempty"`
}

type E2EEMediaKind string

const (
	E2EEMediaImage    E2EEMediaKind = "image"
	E2EEMediaVideo    E2EEMediaKind = "video"
	E2EEMediaAudio    E2EEMediaKind = "audio"
	E2EEMediaDocument E2EEMediaKind = "document"
	E2EEMediaSticker  E2EEMediaKind = "sticker"
)

type E2EEMediaReference struct {
	Kind          E2EEMediaKind `json:"kind"`
	DirectPath    string        `json:"directPath"`
	MediaKey      []byte        `json:"mediaKey"`
	FileSHA256    []byte        `json:"fileSha256"`
	FileEncSHA256 []byte        `json:"fileEncSha256,omitempty"`
	ContentType   string        `json:"contentType,omitempty"`
	Size          int64         `json:"size,omitempty"`
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

type UploadInput struct {
	ThreadID    ID
	Name        string
	ContentType string
	Reader      io.Reader
	Size        int64
	Voice       bool
}

type UploadResult struct {
	ID          ID     `json:"id"`
	Name        string `json:"name,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Type        string `json:"type,omitempty"`
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

type E2EESendRequest struct {
	ChatJID        string
	FacebookUserID ID
	Text           string
	ReplyTo        ID
	ReplySenderJID string
}

type E2EEMediaInput struct {
	ChatJID        string
	FacebookUserID ID
	Kind           E2EEMediaKind
	Name           string
	ContentType    string
	Reader         io.Reader
	Size           int64
	Caption        string
	Width          int
	Height         int
	Duration       int
	Voice          bool
	ReplyTo        ID
	ReplySenderJID string
}

type E2EEMediaDownload struct {
	Reference E2EEMediaReference
}

type E2EEReactionRequest struct {
	ChatJID   string
	MessageID ID
	SenderJID string
	Reaction  string
}

type E2EEReadRequest struct {
	ChatJID    string
	SenderJID  string
	MessageIDs []ID
	Timestamp  time.Time
}

type ReactionEvent struct {
	MessageID ID     `json:"messageId"`
	ThreadID  ID     `json:"threadId"`
	ActorID   ID     `json:"actorId"`
	Reaction  string `json:"reaction,omitempty"`
}

type TypingEvent struct {
	ThreadID ID   `json:"threadId"`
	SenderID ID   `json:"senderId"`
	Typing   bool `json:"typing"`
}

type ReadReceiptEvent struct {
	ThreadID  ID        `json:"threadId"`
	ReaderID  ID        `json:"readerId"`
	Watermark time.Time `json:"watermark"`
}

type DeliveryReceiptEvent struct {
	ThreadID    ID        `json:"threadId"`
	RecipientID ID        `json:"recipientId"`
	Watermark   time.Time `json:"watermark"`
}

type MessageEditEvent struct {
	MessageID ID     `json:"messageId"`
	Text      string `json:"text"`
	EditCount int64  `json:"editCount,omitempty"`
}

type MessageUnsendEvent struct {
	MessageID ID `json:"messageId"`
	ThreadID  ID `json:"threadId"`
}

type ThreadUpdateEvent struct {
	ThreadID ID     `json:"threadId"`
	Field    string `json:"field"`
	Value    string `json:"value,omitempty"`
}

type E2EEReceiptEvent struct {
	Type       string `json:"type"`
	ChatJID    string `json:"chatJid"`
	SenderJID  string `json:"senderJid"`
	MessageIDs []ID   `json:"messageIds,omitempty"`
}

type MessageRequest struct {
	SenderID  ID        `json:"senderId"`
	Snippet   string    `json:"snippet,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type Theme struct {
	ID                           ID       `json:"id"`
	Name                         string   `json:"name,omitempty"`
	Description                  string   `json:"description,omitempty"`
	AppColorMode                 string   `json:"appColorMode,omitempty"`
	ComposerBackgroundColor      string   `json:"composerBackgroundColor,omitempty"`
	BackgroundGradientColors     []string `json:"backgroundGradientColors,omitempty"`
	TitleBarButtonTintColor      string   `json:"titleBarButtonTintColor,omitempty"`
	InboundMessageGradientColors []string `json:"inboundMessageGradientColors,omitempty"`
	TitleBarTextColor            string   `json:"titleBarTextColor,omitempty"`
	ComposerTintColor            string   `json:"composerTintColor,omitempty"`
	TitleBarAttributionColor     string   `json:"titleBarAttributionColor,omitempty"`
	ComposerInputBackgroundColor string   `json:"composerInputBackgroundColor,omitempty"`
	HotLikeColor                 string   `json:"hotLikeColor,omitempty"`
	BackgroundImage              string   `json:"backgroundImage,omitempty"`
	MessageTextColor             string   `json:"messageTextColor,omitempty"`
	InboundMessageTextColor      string   `json:"inboundMessageTextColor,omitempty"`
	PrimaryButtonBackgroundColor string   `json:"primaryButtonBackgroundColor,omitempty"`
	TitleBarBackgroundColor      string   `json:"titleBarBackgroundColor,omitempty"`
	TertiaryTextColor            string   `json:"tertiaryTextColor,omitempty"`
	ReactionPillBackgroundColor  string   `json:"reactionPillBackgroundColor,omitempty"`
	SecondaryTextColor           string   `json:"secondaryTextColor,omitempty"`
	FallbackColor                string   `json:"fallbackColor,omitempty"`
	GradientColors               []string `json:"gradientColors,omitempty"`
	NormalThemeID                ID       `json:"normalThemeId,omitempty"`
	IconAsset                    string   `json:"iconAsset,omitempty"`
}

type Note struct {
	ID          ID     `json:"id,omitempty"`
	Description string `json:"description,omitempty"`
}

type EventKind string

const (
	EventReady           EventKind = "ready"
	EventReconnected     EventKind = "reconnected"
	EventDisconnected    EventKind = "disconnected"
	EventError           EventKind = "error"
	EventMessage         EventKind = "message"
	EventMessageEdit     EventKind = "messageEdit"
	EventMessageUnsend   EventKind = "messageUnsend"
	EventReaction        EventKind = "reaction"
	EventTyping          EventKind = "typing"
	EventReadReceipt     EventKind = "readReceipt"
	EventDeliveryReceipt EventKind = "deliveryReceipt"
	EventThreadUpdate    EventKind = "threadUpdate"
	EventE2EEReady       EventKind = "e2eeReady"
	EventE2EEReceipt     EventKind = "e2eeReceipt"
)

type Event struct {
	Kind EventKind `json:"kind"`
	Data any       `json:"data,omitempty"`
}
