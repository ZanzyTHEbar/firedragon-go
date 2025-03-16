package interfaces

import (
	"time"
)

type StorageAdapter interface {
	// StoreEvent stores an event in the storage
	StoreEvent(event Event) error

	// GetEventsByClientID retrieves events for a specific client
	GetEventsByClientID(clientID string, limit int) ([]Event, error)

	// GetEventsBySessionID retrieves events for a specific session
	GetEventsBySessionID(sessionID string) ([]Event, error)

	// GetEventsByClientIDAndSessionID retrieves events for a specific client and session
	GetEventsByClientIDAndSessionID(clientID, sessionID string) ([]Event, error)

	// GetEventsByClientIDAndTimeRange retrieves events for a specific client and time range
	GetEventsByClientIDAndTimeRange(clientID string, startTime, endTime time.Time) ([]Event, error)

	GetEventsBySessionIDAndTimeRange(sessionID string, startTime, endTime time.Time) ([]Event, error)

	// GetEventsByTimeRange retrieves events within a time range
	GetEventsByTimeRange(clientID string, startTime, endTime time.Time) ([]Event, error)

	// GetEventsByType retrieves events of a specific type
	GetEventsByType(clientID string, eventType EventType, limit int) ([]Event, error)

	// Close closes the storage connection
	Close() error
}
