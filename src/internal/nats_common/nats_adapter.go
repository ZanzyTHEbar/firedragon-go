package nats_common

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// BaseNATSAdapter provides common NATS functionality for both client and server
type BaseNATSAdapter struct {
	Conn      *nats.Conn
	JS        jetstream.JetStream
	Config    NATSConfig
	Logger    *internal.Logger
	Streams   map[string]jetstream.Stream
	Consumers map[string]jetstream.Consumer
	handlers  map[string]internal.EventHandler
	opts      []nats.Option

	// Add these new fields
	subscriptionsMu sync.RWMutex
	subscriptions   map[string]Subscription
}

// NewBaseNATSAdapter creates a new BaseNATSAdapter instance
func NewBaseNATSAdapter(config NATSConfig, logger *internal.Logger) *BaseNATSAdapter {
	return &BaseNATSAdapter{
		Config:        config,
		Logger:        logger,
		Streams:       make(map[string]jetstream.Stream),
		Consumers:     make(map[string]jetstream.Consumer),
		handlers:      make(map[string]internal.EventHandler),
		subscriptions: make(map[string]Subscription), // Initialize the new map
	}
}

// Close closes the NATS connection
func (a *BaseNATSAdapter) Close() error {
	if a.Conn != nil {
		a.Conn.Close()
	}

	return nil
}

func (a *BaseNATSAdapter) Connect() error {
	a.Logger.Debug(internal.ComponentNATS, "Starting NATS connection attempt...")

	// Debug connection parameters
	a.Logger.Debug(internal.ComponentNATS, "Connection parameters: URL=%s, User=%s",
		a.Config.ServerURL,
		a.Config.Username)

	// Initialize connection options if not already done
	if a.opts == nil {
		a.opts = []nats.Option{
			nats.Name(a.Config.ClientID),
			nats.MaxReconnects(-1),
			nats.ReconnectWait(time.Second),
			// Add max payload option (8MB)
			//nats.Pa MaxPayload(8 * 1024 * 1024),
			nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
				a.Logger.Error(internal.ComponentNATS, "NATS error: %v", err)
			}),
			nats.DisconnectHandler(func(_ *nats.Conn) {
				a.Logger.Warn(internal.ComponentNATS, "Disconnected from NATS server")
			}),
			nats.ReconnectHandler(func(_ *nats.Conn) {
				a.Logger.Info(internal.ComponentNATS, "Reconnected to NATS server")

				// Verify JetStream connection
				jsCtx, err := jetstream.New(a.Conn)
				if err != nil {
					a.Logger.Error(internal.ComponentNATS, "Failed to recreate JetStream context after reconnect: %v", err)
					return
				}
				a.JS = jsCtx

				// Reset stream cache as they might have changed
				a.Streams = make(map[string]jetstream.Stream)
				a.Consumers = make(map[string]jetstream.Consumer)

				a.Logger.Info(internal.ComponentNATS, "Successfully reinitialized NATS connection")
			}),
		}

		// Add authentication options
		if a.Config.Username != "" && a.Config.Password != "" {
			a.opts = append(a.opts, nats.UserInfo(a.Config.Username, a.Config.Password))
		}
	}

	// Log all options (excluding sensitive data)
	a.Logger.Debug(internal.ComponentNATS, "Connecting with %d options configured", len(a.opts))

	// Attempt connection with timeout
	var nc *nats.Conn
	var err error

	// Create connection with explicit timeout
	connectChan := make(chan struct{})
	go func() {
		nc, err = nats.Connect(a.Config.ServerURL, a.opts...)
		close(connectChan)
	}()

	// Wait for connection with timeout
	select {
	case <-connectChan:
		if err != nil {
			a.Logger.Error(internal.ComponentNATS, "Connection failed: %v", err)
			return fmt.Errorf("NATS connection failed: %w", err)
		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("NATS connection timeout after 5 seconds")
	}

	if nc == nil {
		a.Logger.Error(internal.ComponentNATS, "Connection succeeded but connection object is nil")
		return fmt.Errorf("NATS connection object is nil after successful connection")
	}

	a.Conn = nc
	a.Logger.Info(internal.ComponentNATS, "Successfully connected to NATS server")

	// Convert nats.JetStreamContext to jetstream.JetStream
	jsCtx, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		a.Logger.Error(internal.ComponentNATS, "Failed to create JetStream context: %v", err)
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}
	a.JS = jsCtx
	return nil
}

// GetConn returns the NATS connection
func (a *BaseNATSAdapter) GetConn() *nats.Conn {
	return a.Conn
}

// EnsureStream is now the only public method for stream operations
func (a *BaseNATSAdapter) EnsureStream(name string, subjects []string) error {
	a.Logger.Debug(internal.ComponentNATS, "Ensuring stream %s exists with subjects: %v", name, subjects)

	js, err := a.GetConn().JetStream()
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Check if stream exists
	streamInfo, err := js.StreamInfo(name)
	if err == nil {
		// Stream exists, check if we need to update
		if !streamsEqual(streamInfo.Config.Subjects, subjects) {
			a.Logger.Info(internal.ComponentNATS, "Updating existing stream %s with new subjects: %v", name, subjects)

			cfg := streamInfo.Config
			cfg.Subjects = subjects

			_, err = js.UpdateStream(&cfg)
			if err != nil {
				return fmt.Errorf("failed to update stream: %w", err)
			}
			a.Logger.Info(internal.ComponentNATS, "Successfully updated stream %s", name)
		}
		return nil
	}

	// Create new stream with default configuration
	a.Logger.Info(internal.ComponentNATS, "Creating new stream %s with subjects: %v", name, subjects)
	cfg := &nats.StreamConfig{
		Name:      name,
		Subjects:  subjects,
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    24 * time.Hour,
	}

	_, err = js.AddStream(cfg)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	a.Logger.Info(internal.ComponentNATS, "Successfully created stream %s", name)
	return nil
}

/* func (a *BaseNATSAdapter) EnsureStream(name string, subjects []string) error {
	// Get JetStream context
	js, err := a.Conn.JetStream()
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	a.Logger.Debug(internal.ComponentNATS, "Ensuring stream %s exists with subjects: %v", name, subjects)

	// Check if stream exists
	stream, err := js.StreamInfo(name)
	if err == nil {
		// Stream exists, check if we need to update
		if !streamsEqual(stream.Config.Subjects, subjects) {
			a.Logger.Info(internal.ComponentNATS, "Updating existing stream %s with new subjects: %v", name, subjects)

			// Preserve existing stream configuration
			updatedConfig := stream.Config
			updatedConfig.Subjects = subjects

			_, err = js.UpdateStream(&updatedConfig)
			if err != nil {
				return fmt.Errorf("failed to update stream: %w", err)
			}
			a.Logger.Info(internal.ComponentNATS, "Successfully updated stream %s", name)
		} else {
			a.Logger.Debug(internal.ComponentNATS, "Stream %s already exists with correct subjects", name)
		}
		return nil
	}

	// If stream doesn't exist, check for overlapping subjects

	streams := js.StreamNames()
	for streamName := range streams {
		info, err := js.StreamInfo(streamName)
		if err != nil {
			continue
		}

		// Check for subject overlap
		for _, existingSubject := range info.Config.Subjects {
			for _, newSubject := range subjects {
				if subjectsOverlap(existingSubject, newSubject) {
					a.Logger.Warn(internal.ComponentNATS, "Subject %s overlaps with existing stream %s subject %s",
						newSubject, streamName, existingSubject)
					return fmt.Errorf("subjects overlap with existing stream %s", streamName)
				}
			}
		}
	}

	// Create new stream if no overlaps found
	a.Logger.Info(internal.ComponentNATS, "Creating new stream %s with subjects: %v", name, subjects)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     name,
		Subjects: subjects,
		Storage:  nats.FileStorage,
		MaxAge:   24 * time.Hour, // Default retention period
	})
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	a.Logger.Info(internal.ComponentNATS, "Successfully created stream %s", name)
	return nil
} */

// createStream is now private
func (a *BaseNATSAdapter) createStream(name string, subjects []string) error {
	return a.EnsureStream(name, subjects)
}

// createStreamWithOptions is now private
func (a *BaseNATSAdapter) createStreamWithOptions(name string, subjects []string, options jetstream.StreamConfig) error {
	return a.EnsureStream(name, subjects)
}

// PublishToStream publishes a message to a JetStream stream
func (a *BaseNATSAdapter) PublishToStream(streamSubject string, data []byte) error {
	a.Logger.Debug(internal.ComponentNATS, "Publishing message to stream subject: %s (size: %d bytes)", streamSubject, len(data))

	jetStreamAck, err := a.JS.Publish(context.Background(), streamSubject, data)

	if err == nil {
		a.Logger.Debug(internal.ComponentNATS, "Published message to stream subject: %s with sequence: %d", streamSubject, jetStreamAck.Sequence)
	}

	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to publish to stream subject %s: %v", streamSubject, err)
		return fmt.Errorf("failed to publish to stream subject %s: %w", streamSubject, err)
	}

	return nil
}

// Publish publishes a message to a regular NATS subject
func (a *BaseNATSAdapter) Publish(subject string, data []byte) error {
	a.Logger.Debug(internal.ComponentNATS, "Publishing message to subject: %s (size: %d bytes)", subject, len(data))
	err := a.Conn.Publish(subject, data)
	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to publish message to %s: %v", subject, err)
	}
	return err
}

// PublishWithContext publishes a message to a regular NATS subject with context support
func (a *BaseNATSAdapter) PublishWithContext(ctx context.Context, subject string, data interface{}) error {

	if ctx == nil {
		return a.PublishMessage(subject, data)
	}

	if ctx.Err() != nil {
		return fmt.Errorf("publish operation cancelled: %w", ctx.Err())
	}

	// Create a channel to signal completion
	done := make(chan error, 1)

	go func() {
		done <- a.PublishMessage(subject, data)
	}()

	// Wait for either context cancellation or publish completion
	select {
	case <-ctx.Done():
		return fmt.Errorf("publish operation cancelled: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			a.Logger.Error(internal.ComponentNATS, "Failed to publish message to %s: %v", subject, err)
		}
		return err
	}
}

// PublishMessage publishes a structured message to a subject
func (a *BaseNATSAdapter) PublishMessage(subject string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to marshal message data: %v", err)
		return fmt.Errorf("failed to marshal message data: %w", err)
	}

	a.Logger.Debug(internal.ComponentNATS, "Publishing message to subject: %s (size: %d bytes)", subject, len(jsonData))

	if a.JS != nil && a.Config.StreamName != "" && a.Streams[a.Config.StreamName] != nil {
		return a.PublishToStream(subject, jsonData)
	}

	return a.Publish(subject, jsonData)
}

// SubscribeToStream subscribes to a JetStream stream with a consumer
func (a *BaseNATSAdapter) SubscribeToStream(stream, consumer string, callback func([]byte)) (Subscription, error) {
	a.Logger.Debug(internal.ComponentNATS, "Subscribing to stream: %s with consumer: %s", stream, consumer)

	s, streamErr := a.checkIfStreamExists(stream, false, nil)
	if streamErr != nil {
		return nil, streamErr
	}

	a.Logger.Debug(internal.ComponentNATS, "Stream %s found", stream)

	// Create or get consumer
	consumerConfig := jetstream.ConsumerConfig{
		Name:          consumer,
		Durable:       consumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}

	cons, err := s.CreateOrUpdateConsumer(context.Background(), consumerConfig)
	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to create consumer %s: %v", consumer, err)
		return nil, fmt.Errorf("failed to create consumer %s: %w", consumer, err)
	}

	a.Consumers[consumer] = cons

	a.Logger.Debug(internal.ComponentNATS, "Consumer %s created/updated", consumer)

	// Create subscription
	sub, err := cons.Consume(func(msg jetstream.Msg) {
		a.Logger.Debug(internal.ComponentNATS, "Received message on stream: %s with consumer: %s (size: %d bytes)", stream, consumer, len(msg.Data()))
		callback(msg.Data())
		msg.Ack()
		a.Logger.Debug(internal.ComponentNATS, "Acknowledged message on stream: %s with consumer: %s", stream, consumer)
	})
	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to subscribe to stream %s with consumer %s: %v", stream, consumer, err)
		return nil, fmt.Errorf("failed to subscribe to stream %s with consumer %s: %w", stream, consumer, err)
	}

	a.Logger.Debug(internal.ComponentNATS, "Subscribed to stream: %s with consumer: %s", stream, consumer)

	return &BaseJetStreamSubscription{Sub: sub}, nil
}

// Subscribe subscribes to a regular NATS subject
func (a *BaseNATSAdapter) Subscribe(subject string, callback func([]byte)) (Subscription, error) {
	a.Logger.Debug(internal.ComponentNATS, "Subscribing to subject: %s", subject)

	sub, err := a.Conn.Subscribe(subject, func(msg *nats.Msg) {
		a.Logger.Debug(internal.ComponentNATS, "Received message on subject: %s (size: %d bytes)", subject, len(msg.Data))
		callback(msg.Data)
	})
	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to subscribe to %s: %v", subject, err)
		return nil, err
	}

	// Store the subscription
	a.subscriptionsMu.Lock()
	subscription := BaseNATSSubscription{Sub: sub}
	a.subscriptions[subject] = subscription
	a.subscriptionsMu.Unlock()

	a.Logger.Debug(internal.ComponentNATS, "Subscribed to subject: %s", subject)
	return subscription, nil
}

// RewindStream rewinds a stream to a specific point in time for a consumer
func (a *BaseNATSAdapter) RewindStream(stream, consumer string, startTime time.Time) error {
	a.Logger.Debug(internal.ComponentNATS, "Rewinding stream: %s consumer: %s to %v", stream, consumer, startTime)

	s, streamErr := a.checkIfStreamExists(stream, true, nil)
	if streamErr != nil {
		return streamErr
	}

	// Create a new consumer with a start time
	consumerConfig := jetstream.ConsumerConfig{
		Name:          consumer,
		Durable:       consumer,
		DeliverPolicy: jetstream.DeliverByStartTimePolicy,
		OptStartTime:  &startTime,
		AckPolicy:     jetstream.AckExplicitPolicy,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}

	_, err := s.CreateOrUpdateConsumer(context.Background(), consumerConfig)
	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to rewind consumer %s: %v", consumer, err)
		return fmt.Errorf("failed to rewind consumer %s: %w", consumer, err)
	}

	a.Logger.Debug(internal.ComponentNATS, "Rewound stream %s consumer %s to %v", stream, consumer, startTime)
	return nil
}

// RewindStreamBySequence rewinds a stream to a specific sequence number for a consumer
func (a *BaseNATSAdapter) RewindStreamBySequence(stream, consumer string, sequence uint64) error {
	s, streamErr := a.checkIfStreamExists(stream, true, nil)
	if streamErr != nil {
		return streamErr
	}

	// Create a new consumer with a start sequence
	consumerConfig := jetstream.ConsumerConfig{
		Name:          consumer,
		Durable:       consumer,
		DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
		OptStartSeq:   sequence,
		AckPolicy:     jetstream.AckExplicitPolicy,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}

	_, err := s.CreateOrUpdateConsumer(context.Background(), consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to rewind consumer %s to sequence %d: %w", consumer, sequence, err)
	}

	a.Logger.Debug(internal.ComponentNATS, "Rewound stream %s consumer %s to sequence %d", stream, consumer, sequence)
	return nil
}

// GetStreamInfo returns information about a stream
func (a *BaseNATSAdapter) GetStreamInfo(stream string) (*jetstream.StreamInfo, error) {
	s, err := a.checkIfStreamExists(stream, false, nil)
	if err != nil {
		return nil, err
	}

	info, err := s.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	return info, nil
}

// RegisterEventHandler registers a handler function for a specific subject
func (a *BaseNATSAdapter) RegisterEventHandler(subject string, handler internal.EventHandler) error {
	// Check if subject is included in the stream
	if a.Config.StreamName != "" {
		found := false
		for _, streamSubject := range a.Config.Subjects {
			if MatchSubject(streamSubject, subject) {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("subject %s is not included in the stream %s", subject, a.Config.StreamName)
		}
	}

	a.handlers[subject] = handler
	return nil
}

// UnregisterEventHandler removes a subscription for the given subject
func (a *BaseNATSAdapter) UnregisterEventHandler(subject string) error {
	a.subscriptionsMu.Lock()
	defer a.subscriptionsMu.Unlock()

	sub, exists := a.subscriptions[subject]
	if !exists {
		return fmt.Errorf("no subscription found for subject: %s", subject)
	}

	if err := sub.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe from subject %s: %w", subject, err)
	}

	delete(a.subscriptions, subject)
	return nil
}

// StartConsumer starts consuming messages from the stream
func (a *BaseNATSAdapter) StartConsumer(consumerName string) error {
	// Add debug logging
	a.Logger.Debug(internal.ComponentNATS, "Starting consumer %s for stream %s", consumerName, a.Config.StreamName)

	// Check if JetStream is available and we have a stream to consume
	streamName := a.Config.StreamName
	if streamName == "" || a.Streams[streamName] == nil {
		a.Logger.Error(internal.ComponentNATS, "Stream name is empty or stream not found")
		return fmt.Errorf("invalid stream configuration")
	}

	// Create or get the consumer with explicit configuration
	ctx := context.Background()
	consumerConfig := &ConsumerConfig{
		Name:           consumerName,
		Stream:         streamName,
		FilterSubjects: a.Config.Subjects,
		DeliverPolicy:  "all",
		AckPolicy:      "explicit",
		ReplayPolicy:   jetstream.ReplayInstantPolicy,
	}

	consumer, err := a.SetupStreamConsumerWithConfig(ctx, streamName, consumerName, consumerConfig)
	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to setup consumer: %v", err)
		return fmt.Errorf("could not setup stream consumer: %w", err)
	}

	// Add more debug logging
	a.Logger.Debug(internal.ComponentNATS, "Consumer created successfully, starting consumption")

	// Consume messages with explicit handler
	_, err = consumer.Consume(func(msg jetstream.Msg) {
		subject := msg.Subject()
		a.Logger.Debug(internal.ComponentNATS, "Received message on subject: %s", subject)

		if handler, exists := a.handlers[subject]; exists {
			if err := handler(msg.Data()); err != nil {
				a.Logger.Error(internal.ComponentNATS, "Error handling message: %v", err)
			}
		}

		if err := msg.Ack(); err != nil {
			a.Logger.Error(internal.ComponentNATS, "Error acknowledging message: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("could not create consumer subscription: %w", err)
	}

	return nil
}

// startRegularSubscription sets up regular NATS subscriptions instead of JetStream
func (a *BaseNATSAdapter) startRegularSubscription() error {
	// For each handler, create a subscription
	for pattern, handler := range a.handlers {
		// Create a closure to capture the handler
		h := handler // capture to avoid issues with closures
		_, err := a.Conn.Subscribe(pattern, func(msg *nats.Msg) {
			if err := h(msg.Data); err != nil {
				a.Logger.Error(internal.ComponentNATS, "Error handling message on subject %s: %v", msg.Subject, err)
			}
		})

		if err != nil {
			return fmt.Errorf("could not subscribe to %s: %w", pattern, err)
		}

		a.Logger.Info(internal.ComponentNATS, "Subscribed to regular NATS subject: %s", pattern)
	}

	return nil
}

// checkIfStreamExists checks if a stream exists and optionally creates it if it doesn't
func (a *BaseNATSAdapter) checkIfStreamExists(stream string, create bool, config *jetstream.StreamConfig) (jetstream.Stream, error) {
	if s, ok := a.Streams[stream]; ok {
		return s, nil
	}

	ctx := context.Background()
	s, err := a.JS.Stream(ctx, stream)
	if err != nil {
		// If the stream is not found and creation is allowed, use shared helper
		if create {
			if config == nil {
				config = &jetstream.StreamConfig{
					Storage:   jetstream.FileStorage,
					Retention: jetstream.LimitsPolicy,
					MaxAge:    24 * time.Hour,
				}
			}
			// Use stream pattern as default
			s, err = EnsureStreamExists(a.JS, stream, []string{stream + ".>"}, *config)
			if err != nil {
				return nil, err
			}
			a.Streams[stream] = s
			return s, nil
		}
		return nil, err
	}
	// Cache and return the found stream
	a.Streams[stream] = s
	return s, nil
}

func (a *BaseNATSAdapter) SetupStreamConsumerWithConfig(ctx context.Context, stream string, consumerName string, config *ConsumerConfig) (jetstream.Consumer, error) {
	s, ok := a.Streams[stream]
	if !ok {
		var err error
		s, err = a.JS.Stream(ctx, stream)
		if err != nil {
			return nil, fmt.Errorf("stream not found: %w", err)
		}
		a.Streams[stream] = s
	}

	// Convert ConsumerConfig to jetstream.ConsumerConfig
	jsConfig := jetstream.ConsumerConfig{
		Name:           consumerName,
		Durable:        consumerName,
		AckPolicy:      jetstream.AckExplicitPolicy,
		FilterSubjects: config.FilterSubjects,                      // Changed from FilterSubject to FilterSubjects
		DeliverPolicy:  convertDeliverPolicy(config.DeliverPolicy), // Convert string to DeliverPolicy
		OptStartTime:   config.StartTime,
		OptStartSeq:    config.StartSequence,
		ReplayPolicy:   config.ReplayPolicy,
	}

	consumer, err := s.CreateOrUpdateConsumer(ctx, jsConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create consumer: %w", err)
	}

	a.Consumers[consumerName] = consumer
	return consumer, nil
}

func (a *BaseNATSAdapter) StartConsumerWithConfig(consumerName string, config *ConsumerConfig) error {
	// Check if JetStream is available and we have a stream to consume
	streamName := a.Config.StreamName
	if streamName == "" || a.Streams[streamName] == nil {
		// Fall back to regular NATS subscription when JetStream isn't available
		return a.startRegularSubscription()
	}

	// Create or get the consumer
	ctx := context.Background()
	consumer, err := a.SetupStreamConsumerWithConfig(ctx, streamName, consumerName, config)
	if err != nil {
		return fmt.Errorf("could not setup stream consumer: %w", err)
	}

	// Consume messages
	_, err = consumer.Consume(func(msg jetstream.Msg) {
		subject := msg.Subject()
		handler, exists := a.handlers[subject]
		if exists {
			if err := handler(msg.Data()); err != nil {
				a.Logger.Error(internal.ComponentNATS, "Error handling message on subject %s: %v", subject, err)
			}
		} else {
			// Try to find a wildcard handler
			for pattern, h := range a.handlers {
				if MatchSubject(pattern, subject) {
					if err := h(msg.Data()); err != nil {
						a.Logger.Error(internal.ComponentNATS, "Error handling message on subject %s with pattern %s: %v", subject, pattern, err)
					}
					break
				}
			}
		}

		// Acknowledge message
		if err := msg.Ack(); err != nil {
			a.Logger.Error(internal.ComponentNATS, "Error acknowledging message: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("could not create consumer subscription: %w", err)
	}

	return nil
}

// SetupStreamConsumer creates a new consumer for a stream with default settings
func (a *BaseNATSAdapter) SetupStreamConsumer(ctx context.Context, stream string, consumerName string) (jetstream.Consumer, error) {
	// Create a durable consumer
	consumerConfig := jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: ">", // Subscribe to all subjects in the stream
	}

	s, ok := a.Streams[stream]
	if !ok {
		var err error
		s, err = a.JS.Stream(ctx, stream)
		if err != nil {
			return nil, fmt.Errorf("stream not found: %w", err)
		}
		a.Streams[stream] = s
	}

	consumer, err := s.CreateOrUpdateConsumer(ctx, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create consumer: %w", err)
	}

	a.Consumers[consumerName] = consumer
	return consumer, nil
}

// SetupStream ensures a stream exists or creates it with default settings
func (a *BaseNATSAdapter) SetupStream(streamName string, subjects []string) error {
	ctx := context.Background()

	// Check if JetStream is supported by trying to access JetStream info
	_, err := a.JS.AccountInfo(ctx)
	if err != nil {
		a.Logger.Debug(internal.ComponentNATS, "JetStream not enabled on the server or not available: %v", err)
		a.Logger.Debug(internal.ComponentNATS, "Using basic NATS functionality without JetStream")
		return nil // Continue without JetStream
	}

	// Check if the stream already exists
	streamInfo, err := a.JS.Stream(ctx, streamName)
	if err == nil && streamInfo != nil {
		a.Logger.Warn(internal.ComponentNATS, "Using existing stream: %s", streamName)
		a.Streams[streamName] = streamInfo
		return nil
	}

	// Create stream if it doesn't exist
	cfg := jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  subjects,
		Storage:   jetstream.FileStorage,
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    7 * 24 * time.Hour,
	}

	stream, err := a.JS.CreateStream(ctx, cfg)
	if err != nil {
		return fmt.Errorf("could not create stream: %w", err)
	}

	a.Streams[streamName] = stream
	a.Logger.Info(internal.ComponentNATS, "Created new stream: %s", streamName)

	return nil
}

// RegisterRequestHandler registers a handler function for request-reply pattern
func (a *BaseNATSAdapter) RegisterRequestHandler(subject string, handler func([]byte) ([]byte, error)) error {
	a.Logger.Debug(internal.ComponentNATS, "Registering request handler for subject: %s", subject)

	if a.Conn == nil {
		if err := a.Connect(); err != nil {
			a.Logger.Error(internal.ComponentNATS, "Failed to connect to NATS when registering request handler: %v", err)
			return err
		}
	}

	_, err := a.Conn.Subscribe(subject, func(msg *nats.Msg) {
		a.Logger.Debug(internal.ComponentNATS, "Received request on subject: %s", subject)

		response, err := handler(msg.Data)
		if err != nil {
			a.Logger.Error(internal.ComponentNATS, "Error handling request on subject %s: %v", subject, err)
			// Send error response if reply subject is provided
			if msg.Reply != "" {
				errMsg := []byte(fmt.Sprintf("error: %v", err))
				if err := a.Conn.Publish(msg.Reply, errMsg); err != nil {
					a.Logger.Error(internal.ComponentNATS, "Failed to publish error response: %v", err)
				}
			}
			return
		}

		// Send response if reply subject is provided
		if msg.Reply != "" {
			if err := a.Conn.Publish(msg.Reply, response); err != nil {
				a.Logger.Error(internal.ComponentNATS, "Failed to publish response: %v", err)
			} else {
				a.Logger.Debug(internal.ComponentNATS, "Sent response to subject: %s", msg.Reply)
			}
		}
	})

	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to register request handler for %s: %v", subject, err)
		return fmt.Errorf("failed to register request handler: %w", err)
	}

	a.Logger.Debug(internal.ComponentNATS, "Registered request handler for subject: %s", subject)
	return nil
}

func (a *BaseNATSAdapter) GetJetStream() jetstream.JetStream {
	return a.JS
}

func (a *BaseNATSAdapter) GetClientID() string {
	return a.Config.ClientID
}

func (a *BaseNATSAdapter) GetSessionID() string {
	return fmt.Sprintf("%s-%d", a.Config.ClientID, time.Now().Unix())
}

func (a *BaseNATSAdapter) GetConfig() NATSConfig {
	return a.Config
}
func (a *BaseNATSAdapter) IsConnected() bool {
	return a.Conn != nil && a.Conn.IsConnected()
}

// BaseNATSSubscription implements a subscription for regular NATS
type BaseNATSSubscription struct {
	Sub *nats.Subscription
}

func (s BaseNATSSubscription) Unsubscribe() error {
	return s.Sub.Unsubscribe()
}

// BaseJetStreamSubscription implements a subscription for JetStream
type BaseJetStreamSubscription struct {
	Sub jetstream.ConsumeContext
}

func (s BaseJetStreamSubscription) Unsubscribe() error {
	s.Sub.Stop()
	return nil
}

// convertDeliverPolicy converts a string to jetstream.DeliverPolicy
func convertDeliverPolicy(policy string) jetstream.DeliverPolicy {
	switch policy {
	case "all":
		return jetstream.DeliverAllPolicy
	case "last":
		return jetstream.DeliverLastPolicy
	case "new":
		return jetstream.DeliverNewPolicy
	case "by_start_sequence":
		return jetstream.DeliverByStartSequencePolicy
	case "by_start_time":
		return jetstream.DeliverByStartTimePolicy
	case "last_per_subject":
		return jetstream.DeliverLastPerSubjectPolicy
	default:
		return jetstream.DeliverAllPolicy // Default to delivering all messages
	}
}

// Helper function to compare subject slices
func streamsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Helper function to check if two subjects overlap
func subjectsOverlap(subject1, subject2 string) bool {
	// Convert subjects to regex patterns
	pattern1 := strings.ReplaceAll(subject1, ".", "\\.")
	pattern1 = strings.ReplaceAll(pattern1, "*", "[^.]+")
	pattern1 = strings.ReplaceAll(pattern1, ">", ".*")
	pattern1 = "^" + pattern1 + "$"

	pattern2 := strings.ReplaceAll(subject2, ".", "\\.")
	pattern2 = strings.ReplaceAll(pattern2, "*", "[^.]+")
	pattern2 = strings.ReplaceAll(pattern2, ">", ".*")
	pattern2 = "^" + pattern2 + "$"

	// Check if either pattern matches the other subject
	match1, _ := regexp.MatchString(pattern1, subject2)
	match2, _ := regexp.MatchString(pattern2, subject1)

	return match1 || match2
}

// RequestMessage sends a request and waits for a response
func (a *BaseNATSAdapter) RequestMessage(subject string, data []byte) ([]byte, error) {
	a.Logger.Debug(internal.ComponentNATS, "Sending request to subject: %s", subject)

	msg, err := a.Conn.Request(subject, data, 30*time.Second)
	if err != nil {
		a.Logger.Error(internal.ComponentNATS, "Failed to send request to %s: %v", subject, err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return msg.Data, nil
}
