package internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
)

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

func GenerateUUID() string {
	// For simplicity, we use a UUID library here.
	// In a real-world scenario, you might want to use a more efficient method.

	_uuid := uuid.New().String()

	if _uuid == "" {
		return "00000000-0000-0000-0000-000000000000"
	}
	return _uuid
}

func GenerateClientID() string {
	clientID, err := os.Hostname()
	if err != nil {
		GetLogger().Warn(ComponentGeneral, "Error getting hostname: %v", err)
		os.Exit(1)
	}
	return clientID
}
