package nats_common

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type ServiceStatusResponse struct {
	Status        string
	StartTime     *time.Time
	EventsHandled int64
	ErrorCount    int
	LastErrorTime *time.Time
}

// RequestServiceStatus requests the current status of a service
func RequestServiceStatus(natsURL string, serviceName string) (*ServiceStatusResponse, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	msg, err := nc.Request(fmt.Sprintf("service.%s.status", serviceName), nil, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to get service status: %v", err)
	}

	var response ServiceStatusResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse service status: %v", err)
	}

	return &response, nil
}

// SendServiceCommand sends a command to control a service
func SendServiceCommand(natsURL string, command string) error {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	if err := nc.Publish("service.command", []byte(command)); err != nil {
		return fmt.Errorf("failed to send command: %v", err)
	}

	return nil
}

// RequestServiceData requests data from a service
func RequestServiceData(natsURL string, subject string) (*nats.Msg, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	msg, err := nc.Request(subject, nil, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to get service data: %v", err)
	}

	return msg, nil
}
