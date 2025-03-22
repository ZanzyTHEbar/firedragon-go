package interfaces

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventTypeMachineInfo   EventType = "client.data.machineInfo"
	EventTypeOpenApps      EventType = "client.data.openApps"
	EventTypeCodeEditor    EventType = "client.data.codeEditor"
	// System events
	EventTypeStart         = "system.start"
	EventTypeStop          = "system.stop"
	EventTypeConfig        = "system.config"
	EventTypeStatus        = "system.status"
	
	// Transaction events
	EventTypeSyncRequest   = "tx.sync.request"
	EventTypeSyncComplete  = "tx.sync.complete"
	EventTypeSyncError     = "tx.sync.error"
	
	// Account events
	EventTypeBalanceUpdate = "account.balance.update"
	EventTypeTokenRefresh  = "account.token.refresh"
	
	// Error events
	EventTypeError        = "error"
)

type Event struct {
	ID         string            `json:"id"`
	Type       EventType         `json:"type"`
	ClientID   string            `json:"client_id"`
	SessionID  string            `json:"session_id"`
	Timestamp  time.Time         `json:"timestamp"` // UnixNano
	RawPayload []byte            `json:"raw_payload,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     error                  `json:"error,omitempty"`
}

// EventHandler defines a function that processes NATS messages
type EventHandler func(data []byte) error

// EventProcessor defines the interface for processing events
type EventProcessor interface {
	// ProcessEvent processes an incoming event
	ProcessEvent(event Event) error

	// ReplayEvents replays events from a specific time range for a client
	ReplayEvents(clientID string, startTime, endTime time.Time) error

	// StoreEvent stores an event in the event store
	StoreEvent(event Event) error
}

func (e *Event) String() string {
	// Serialize the entire event structure to JSON
	jsonData, err := json.Marshal(e)
	if err != nil {
		return "Error serializing event"
	}
	return string(jsonData)
}

// NewEvent creates a new event with the current timestamp
func NewEvent(eventType EventType, source string) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Source:    source,
		Data:      make(map[string]interface{}),
	}
}

// WithTarget sets the target for the event
func (e *Event) WithTarget(target string) *Event {
	e.Target = target
	return e
}

// WithData adds data to the event
func (e *Event) WithData(key string, value interface{}) *Event {
	e.Data[key] = value
	return e
}

// WithError adds an error to the event
func (e *Event) WithError(err error) *Event {
	e.Error = err
	return e
}

// SyncRequest represents a request to sync transactions
type SyncRequest struct {
	Source    string    `json:"source"`     // blockchain or bank
	Provider  string    `json:"provider"`   // specific provider
	AccountID string    `json:"account_id"` // account to sync
	Since     time.Time `json:"since"`      // sync transactions since this time
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	Request       SyncRequest `json:"request"`
	Transactions  int         `json:"transactions"`  // number of transactions processed
	NewImports    int         `json:"new_imports"`  // number of new transactions imported
	Errors        []error     `json:"errors"`       // any errors encountered
	Duration      float64     `json:"duration"`     // processing time in seconds
	LastProcessed time.Time   `json:"last_processed"`
}

// TokenData represents OAuth token information
type TokenData struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
	Scope        string    `json:"scope"`
}

// AccountInfo represents basic account information
type AccountInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`     // checking, savings, wallet, etc.
	Currency string `json:"currency"` // primary currency
	Provider string `json:"provider"` // bank or blockchain provider
}

// ErrorInfo represents detailed error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Source  string `json:"source"`
	Context map[string]interface{} `json:"context,omitempty"`
}
