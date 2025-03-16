package nats_common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATSService manages the lifecycle of the NATS adapter
type NATSService struct {
	adapter          MessagingPort
	config           NATSConfig
	clientID         string
	sessionID        string
	logger           *internal.Logger
	handlers         map[string]interfaces.EventHandler
	serviceManager   *internal.ServiceManager
	consumer         string
	reconnectHandler func()
	errorHandler     func(error)
	subscriptions    map[string]Subscription
	subscriptionsMu  sync.Mutex
	initialized      bool
	mu               sync.Mutex
}

// NewNATSService creates a new NATS service
func NewNATSService(consumer string, config NATSConfig, mgr *internal.ServiceManager) *NATSService {
	if config.ClientID == "" {
		config.ClientID = internal.GenerateClientID()
	}

	// Create NATS Adapter
	natsAdapter, err := InitializeNATS(config)
	if err != nil {
		// Instead of panicking, return a service that will report its connection status as false
		service := &NATSService{
			config:         config,
			clientID:       config.ClientID,
			sessionID:      fmt.Sprintf("%s-%d", config.ClientID, time.Now().Unix()),
			logger:         internal.GetLogger(),
			handlers:       make(map[string]interfaces.EventHandler),
			serviceManager: mgr,
			consumer:       consumer,
			subscriptions:  make(map[string]Subscription),
		}
		service.logger.Error(internal.ComponentNATS, "Failed to initialize NATS adapter: %v", err)
		return service
	}

	service := &NATSService{
		adapter:        natsAdapter,
		config:         config,
		clientID:       config.ClientID,
		sessionID:      fmt.Sprintf("%s-%d", config.ClientID, time.Now().Unix()),
		logger:         internal.GetLogger(),
		handlers:       make(map[string]interfaces.EventHandler),
		serviceManager: mgr,
		consumer:       consumer,
		subscriptions:  make(map[string]Subscription),
	}

	// Add reconnection handler
	service.reconnectHandler = func() {
		service.logger.Info(internal.ComponentNATS, "Reconnected to NATS server")
		// Reinitialize subscriptions if needed
		if err := service.reinitializeSubscriptions(); err != nil {
			service.logger.Error(internal.ComponentNATS, "Failed to reinitialize subscriptions: %v", err)
		}
	}

	service.errorHandler = func(err error) {
		service.logger.Error(internal.ComponentNATS, "NATS error: %v", err)
	}

	return service
}

func (s *NATSService) createOrUpdateStream(streamName string, subjects []string) error {
	js, err := s.adapter.GetConn().JetStream()
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// First try to get existing stream info
	streamInfo, err := js.StreamInfo(streamName)
	if err != nil && err != nats.ErrStreamNotFound {
		return fmt.Errorf("failed to get stream info: %w", err)
	}

	// Stream config
	cfg := &nats.StreamConfig{
		Name:      streamName,
		Subjects:  subjects,
		Retention: nats.LimitsPolicy,
		MaxAge:    24 * time.Hour,
	}

	if streamInfo == nil {
		// Stream doesn't exist, create new
		s.logger.Debug(internal.ComponentNATS, "Creating new stream: %s with subjects %v", streamName, subjects)
		_, err = js.AddStream(cfg)
		if err != nil {
			return fmt.Errorf("failed to create stream %s: %w", streamName, err)
		}
	} else {
		// Stream exists, update while preserving retention policy
		cfg.Retention = streamInfo.Config.Retention
		s.logger.Debug(internal.ComponentNATS, "Updating existing stream: %s with subjects %v", streamName, subjects)
		_, err = js.UpdateStream(cfg)
		if err != nil {
			return fmt.Errorf("failed to update stream %s: %w", streamName, err)
		}
	}

	return nil
}

// Start implements the Service interface
func (s *NATSService) Start(ctx context.Context) error {
	s.logger.Info(internal.ComponentNATS, "Starting NATS service...")

	if s.adapter == nil {
		return fmt.Errorf("NATS adapter is nil")
	}

	if !s.IsConnected() {
		if err := s.adapter.Connect(); err != nil {
			return fmt.Errorf("failed to connect to NATS: %w", err)
		}
	}

	// Initialize stream as part of service start
	if err := s.initializeStream(); err != nil {
		return err
	}

	// Setup service handlers
	if err := s.setupServiceHandlers(); err != nil {
		return fmt.Errorf("failed to setup service handlers: %w", err)
	}

	s.logger.Info(internal.ComponentNATS, "NATS service started successfully")

	<-ctx.Done()
	s.logger.Info(internal.ComponentNATS, "NATS service shutting down...")

	return nil
}

// GetAdapter returns the underlying NATS adapter
// This allows other services to use the NATS adapter for messaging
func (s *NATSService) GetAdapter() MessagingPort {
	return s.adapter
}

// GetSessionID returns the current session ID
func (s *NATSService) GetSessionID() string {
	return s.sessionID
}

// GetClientID returns the client ID
func (s *NATSService) GetClientID() string {
	return s.clientID
}

// CreateStream is now deprecated and logs a warning
func (s *NATSService) CreateStream(name string, subjects []string) error {
	s.logger.Warn(internal.ComponentNATS, "CreateStream is deprecated. Streams are automatically initialized during service start")
	return nil
}

// CreateStreamWithOptions is now deprecated and logs a warning
func (s *NATSService) CreateStreamWithOptions(name string, subjects []string, options jetstream.StreamConfig) error {
	s.logger.Warn(internal.ComponentNATS, "CreateStreamWithOptions is deprecated. Streams are automatically initialized during service start")
	return nil
}

// PublishToStream publishes a message to a JetStream stream
func (s *NATSService) PublishToStream(streamSubject string, data []byte) error {
	if s.adapter == nil {
		return fmt.Errorf("NATS adapter not initialized")
	}
	return s.adapter.PublishToStream(streamSubject, data)
}

// SubscribeToStream subscribes to a JetStream stream with a consumer
func (s *NATSService) SubscribeToStream(stream, consumer string, callback func([]byte)) (Subscription, error) {
	if s.adapter == nil {
		return nil, fmt.Errorf("NATS adapter not initialized")
	}
	return s.adapter.SubscribeToStream(stream, consumer, callback)
}

// RewindStream rewinds a stream to a specific point in time for a consumer
func (s *NATSService) RewindStream(stream, consumer string, startTime time.Time) error {
	if s.adapter == nil {
		return fmt.Errorf("NATS adapter not initialized")
	}
	return s.adapter.RewindStream(stream, consumer, startTime)
}

// RewindStreamBySequence rewinds a stream to a specific sequence number for a consumer
func (s *NATSService) RewindStreamBySequence(stream, consumer string, sequence uint64) error {
	if s.adapter == nil {
		return fmt.Errorf("NATS adapter not initialized")
	}
	return s.adapter.RewindStreamBySequence(stream, consumer, sequence)
}

// GetStreamInfo returns information about a stream
func (s *NATSService) GetStreamInfo(stream string) (*jetstream.StreamInfo, error) {
	if s.adapter == nil {
		return nil, fmt.Errorf("NATS adapter not initialized")
	}
	return s.adapter.GetStreamInfo(stream)
}

// Subscribe subscribes to the specified subject
func (s *NATSService) Subscribe(subject string, callback func([]byte)) (Subscription, error) {
	if s.adapter == nil {
		return nil, fmt.Errorf("NATS adapter not initialized")
	}
	return s.adapter.Subscribe(subject, callback)
}

// Publish publishes a message to the specified subject
func (s *NATSService) Publish(subject string, data []byte) error {
	if s.adapter == nil {
		return fmt.Errorf("NATS adapter not initialized")
	}
	return s.adapter.Publish(subject, data)
}

// IsConnected returns whether the NATS adapter is connected
func (s *NATSService) IsConnected() bool {
	if s.adapter == nil {
		s.logger.Debug(internal.ComponentNATS, "IsConnected check failed: adapter is nil")
		return false
	}
	conn := s.adapter.GetConn()
	if conn == nil {
		s.logger.Debug(internal.ComponentNATS, "IsConnected check failed: connection is nil")
		return false
	}
	status := conn.Status()
	s.logger.Debug(internal.ComponentNATS, "Connection status: %v", status)
	return status == nats.CONNECTED
}

// setupServiceHandlers sets up handlers for service management and status
func (s *NATSService) setupServiceHandlers() error {
	// Handle service status requests
	if err := s.adapter.RegisterRequestHandler("service.status.request", func(req []byte) ([]byte, error) {
		// Get service information from the service manager
		services := s.serviceManager.GetAllServicesInfo()

		// Marshal to JSON
		data, err := json.Marshal(services)
		if err != nil {
			s.logger.Error(internal.ComponentNATS, "Failed to marshal service info: %v", err)
			return nil, err
		}

		return data, nil
	}); err != nil {
		return fmt.Errorf("failed to register service status handler: %w", err)
	}

	return nil
}

// InitializeNATS creates and connects a new NATS adapter
func InitializeNATS(config NATSConfig) (*BaseNATSAdapter, error) {
	logger := internal.GetLogger()
	adapter := NewBaseNATSAdapter(config, logger)

	if err := adapter.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return adapter, nil
}

// reinitializeSubscriptions attempts to reestablish all subscriptions after a reconnection
func (s *NATSService) reinitializeSubscriptions() error {
	s.subscriptionsMu.Lock()
	defer s.subscriptionsMu.Unlock()

	var errs []error

	// Re-register all event handlers
	for subject := range s.handlers {
		if err := s.adapter.RegisterEventHandler(subject, s.handlers[subject]); err != nil {
			errs = append(errs, fmt.Errorf("failed to resubscribe to %s: %w", subject, err))
		}
	}

	// If we had any errs, combine them into a single error
	if len(errs) > 0 {
		errStr := "failed to reinitialize some subscriptions:"
		for _, err := range errs {
			errStr += "\n" + err.Error()
		}
		return errors.New(errStr)
	}

	return nil
}

// initializeStream is now a private method that handles stream creation
func (s *NATSService) initializeStream() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		s.logger.Debug(internal.ComponentNATS, "Streams already initialized")
		return nil
	}

	if s.config.StreamName == "" || len(s.config.Subjects) == 0 {
		s.logger.Debug(internal.ComponentNATS, "No stream configuration provided, skipping stream initialization")
		return nil
	}

	s.logger.Debug(internal.ComponentNATS, "Initializing stream %s with subjects: %v",
		s.config.StreamName, s.config.Subjects)

	if err := s.adapter.EnsureStream(s.config.StreamName, s.config.Subjects); err != nil {
		return fmt.Errorf("failed to initialize stream: %w", err)
	}

	s.initialized = true
	s.logger.Info(internal.ComponentNATS, "Stream %s initialized successfully", s.config.StreamName)
	return nil
}
