package internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
)

// DecodePayload decodes a base64-encoded JSON payload
func DecodePayload(encoded string) (map[string]interface{}, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(decoded, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decoded payload: %w", err)
	}
	return result, nil
}

// GenerateUUID generates a random UUID string
func GenerateUUID() string {
	// For simplicity, we use a UUID library here.
	// In a real-world scenario, you might want to use a more efficient method.
	_uuid := uuid.New().String()
	if _uuid == "" {
		return "00000000-0000-0000-0000-000000000000"
	}
	return _uuid
}

// GenerateClientID generates a client identifier based on hostname
func GenerateClientID() string {
	clientID, err := os.Hostname()
	if err != nil {
		GetLogger().Warn(ComponentGeneral, "Error getting hostname: %v", err)
		os.Exit(1)
	}
	return clientID
}

// JSONMarshal marshals an object to JSON
func JSONMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// JSONUnmarshal unmarshals JSON into an object
func JSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// EnsureDir ensures a directory exists, creating it if necessary
func EnsureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// FileExists checks if a file exists at the specified path
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
