package nats_common

import (
	"context"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Subscription represents a subscription to a subject
type Subscription interface {
	Unsubscribe() error
}

// MessagingPort defines the interface for messaging systems
type MessagingPort interface {
	// Connection management
	Connect() error
	Close() error

	RegisterEventHandler(subject string, handler internal.EventHandler) error
	UnregisterEventHandler(subject string) error

	// Request-Reply
	RegisterRequestHandler(subject string, handler func([]byte) ([]byte, error)) error
	RequestMessage(subject string, data []byte) ([]byte, error)

	// Basic pub/sub
	Publish(subject string, data []byte) error
	PublishWithContext(ctx context.Context, subject string, data interface{}) error
	PublishMessage(subject string, data interface{}) error
	Subscribe(subject string, callback func([]byte)) (Subscription, error)

	// JetStream operations
	EnsureStream(name string, subjects []string) error // Ensure a stream exists with the given name and subjects
	PublishToStream(streamSubject string, data []byte) error
	SubscribeToStream(stream, consumer string, callback func([]byte)) (Subscription, error)
	StartConsumer(consumerName string) error
	StartConsumerWithConfig(consumerName string, config *ConsumerConfig) error
	SetupStreamConsumerWithConfig(ctx context.Context, stream string, consumerName string, config *ConsumerConfig) (jetstream.Consumer, error)

	// Rewind/rollback capabilities
	RewindStream(stream, consumer string, startTime time.Time) error
	RewindStreamBySequence(stream, consumer string, sequence uint64) error

	// Stream information
	GetStreamInfo(stream string) (*jetstream.StreamInfo, error)

	// Connection access
	GetConn() *nats.Conn
	GetJetStream() jetstream.JetStream
	GetClientID() string
	GetSessionID() string
	GetConfig() NATSConfig
	IsConnected() bool
}

// ConsumerConfig defines the configuration for a NATS consumer
type ConsumerConfig struct {
	Name           string                 `json:"name"`
	Stream         string                 `json:"stream"`
	FilterSubjects []string               `json:"filter_subjects"`
	DeliverPolicy  string                 `json:"deliver_policy"`
	DeliverGroup   string                 `json:"deliver_group"`
	MaxDeliver     int                    `json:"max_deliver"`
	AckPolicy      string                 `json:"ack_policy"`
	StartTime      *time.Time             `json:"start_time"`
	StartSequence  uint64                 `json:"start_sequence"`
	ReplayPolicy   jetstream.ReplayPolicy `json:"replay_policy"`
}
