package interfaces

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventTypeBrowser       EventType = "browser.data.browser"
	EventTypeHID           EventType = "client.data.hidEvents"
	EventTypeScreen        EventType = "client.data.screen"
	EventTypeTranscription EventType = "client.data.transcription"
	EventTypeMachineInfo   EventType = "client.data.machineInfo"
	EventTypeOpenApps      EventType = "client.data.openApps"
	EventTypeCodeEditor    EventType = "client.data.codeEditor"
)

type Event struct {
	ID         string            `json:"id"`
	Type       EventType         `json:"type"`
	ClientID   string            `json:"client_id"`
	SessionID  string            `json:"session_id"`
	Timestamp  time.Time         `json:"timestamp"` // UnixNano
	RawPayload []byte            `json:"raw_payload,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
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
