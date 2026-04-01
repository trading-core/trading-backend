package tastytrade

type MessageType string

const (
	MessageTypeSetup            MessageType = "SETUP"
	MessageTypeAuth             MessageType = "AUTH"
	MessageTypeAuthState        MessageType = "AUTH_STATE"
	MessageTypeChannelRequest   MessageType = "CHANNEL_REQUEST"
	MessageTypeFeedSetup        MessageType = "FEED_SETUP"
	MessageTypeFeedSubscription MessageType = "FEED_SUBSCRIPTION"
	MessageTypeFeedData         MessageType = "FEED_DATA"
	MessageTypeKeepAlive        MessageType = "KEEPALIVE"
	MessageTypeError            MessageType = "ERROR"
)

type MessageBase struct {
	Type    MessageType `json:"type"`
	Channel int         `json:"channel"`
}

type MessageSetup struct {
	MessageBase
	Version                string `json:"version"`
	KeepaliveTimeout       int    `json:"keepaliveTimeout"`
	AcceptKeepaliveTimeout int    `json:"acceptKeepaliveTimeout"`
}

type AuthState string

const (
	AuthStateUnauthorized AuthState = "UNAUTHORIZED"
	AuthStateAuthorized   AuthState = "AUTHORIZED"
)

type MessageAuthState struct {
	MessageBase
	State  AuthState `json:"state,omitempty"`
	UserID string    `json:"userId,omitempty"`
}

type MessageAuth struct {
	MessageBase
	Token string `json:"token,omitempty"`
}

type MessageError struct {
	MessageBase
	Error   string `json:"error"`
	Message string `json:"message"`
}

type MessageChannelRequest struct {
	MessageBase
	Service    string         `json:"service"`
	Parameters map[string]any `json:"parameters"`
}

type MessageFeedSetup struct {
	MessageBase
	AcceptedAggregationPeriod float64        `json:"acceptedAggregationPeriod"`
	AcceptedDataFormat        string         `json:"acceptedDataFormat"`
	AcceptEventFields         map[string]any `json:"acceptEventFields"`
}

type MessageFeedSubscription struct {
	MessageBase
	Reset bool             `json:"reset"`
	Add   []map[string]any `json:"add,omitempty"`
}

type MessageKeepAlive struct {
	MessageBase
}

// MessageFeedData wraps the raw feed data payload from DXLink
type MessageFeedData struct {
	MessageBase
	Data []any `json:"data"`
}

// MessageQuote represents a Quote feed event with market bid/ask data
type MessageQuote struct {
	MessageBase
	EventType   string  `json:"eventType"`
	EventSymbol string  `json:"eventSymbol"`
	BidPrice    float64 `json:"bidPrice"`
	AskPrice    float64 `json:"askPrice"`
	BidSize     float64 `json:"bidSize"`
	AskSize     float64 `json:"askSize"`
}

// MessageTrade represents a Trade feed event with execution data
type MessageTrade struct {
	MessageBase
	EventType   string  `json:"eventType"`
	EventSymbol string  `json:"eventSymbol"`
	Price       float64 `json:"price"`
	DayVolume   float64 `json:"dayVolume"`
	Size        float64 `json:"size"`
}

type Message interface {
	Base() MessageBase
}

// Message interface implementations
func (m *MessageSetup) Base() MessageBase            { return m.MessageBase }
func (m *MessageAuthState) Base() MessageBase        { return m.MessageBase }
func (m *MessageAuth) Base() MessageBase             { return m.MessageBase }
func (m *MessageError) Base() MessageBase            { return m.MessageBase }
func (m *MessageChannelRequest) Base() MessageBase   { return m.MessageBase }
func (m *MessageFeedSetup) Base() MessageBase        { return m.MessageBase }
func (m *MessageFeedSubscription) Base() MessageBase { return m.MessageBase }
func (m *MessageKeepAlive) Base() MessageBase        { return m.MessageBase }
func (m *MessageFeedData) Base() MessageBase         { return m.MessageBase }
func (m *MessageQuote) Base() MessageBase            { return m.MessageBase }
func (m *MessageTrade) Base() MessageBase            { return m.MessageBase }
