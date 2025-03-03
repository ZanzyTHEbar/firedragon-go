package nats_common

import (
	"context"
	"fmt"

	"github.com/ZanzyTHEbar/firedragon-go/internal"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NATSConfig struct {
	ServerURL      string
	StreamName     string
	Subjects       []string
	EventProcessor interface{}
	ClientID       string
	Username       string
	Password       string
	Token          string
}

// MatchSubject returns whether a subject matches a pattern with wildcard support.
func MatchSubject(pattern, subject string) bool {
	if pattern == subject {
		return true
	}

	if pattern == ">" || pattern == "*" {
		return true
	}

	// Check if pattern ends with ".>"
	if len(pattern) >= 2 && pattern[len(pattern)-2:] == ".>" {
		prefix := pattern[:len(pattern)-2]
		return len(subject) >= len(prefix) && subject[:len(prefix)] == prefix
	}

	// Fallback: if pattern ends with '>'
	if len(pattern) > 0 && pattern[len(pattern)-1] == '>' {
		prefix := pattern[:len(pattern)-1]
		return len(subject) >= len(prefix) && subject[:len(prefix)] == prefix
	}

	return false
}

// EnsureStreamExists checks if a stream exists, and if not, creates it with the given configuration.
// It sets the stream name and subjects in the configuration and calls CreateOrUpdateStream.
func EnsureStreamExists(js jetstream.JetStream, streamName string, subjects []string, config jetstream.StreamConfig) (jetstream.Stream, error) {
	// Set required fields
	config.Name = streamName
	config.Subjects = subjects

	// Ensure defaults if not set
	if config.Storage == 0 {
		config.Storage = jetstream.FileStorage
	}
	if config.Retention == 0 {
		config.Retention = jetstream.LimitsPolicy
	}

	ctx := context.Background()
	stream, err := js.CreateOrUpdateStream(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure stream exists: %w", err)
	}
	return stream, nil
}

// Added function to apply NATS authentication options
func ApplyNATSAuthOptions(username, password, token string) []nats.Option {
	opts := []nats.Option{}
	logger := internal.GetLogger()
	if username != "" && password != "" {
		opts = append(opts, nats.UserInfo(username, password))
		logger.Info(internal.ComponentNATS, "Using username/password authentication for NATS")
	} else if token != "" {
		opts = append(opts, nats.Token(token))
		logger.Info(internal.ComponentNATS, "Using token authentication for NATS")
	} else {
		logger.Warn(internal.ComponentNATS, "No authentication provided for NATS connection")
	}
	return opts
}
