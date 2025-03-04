package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	defaultOllamaHost = "http://localhost:11434"
	defaultModel      = "llama2"
)

type OllamaClient struct {
	host  string
	model string
}

type GenerateEmbeddingRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type GenerateEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewOllamaClient creates a new Ollama client with the given configuration
func NewOllamaClient(host string, model string) *OllamaClient {
	if host == "" {
		host = defaultOllamaHost
	}
	if model == "" {
		model = defaultModel
	}
	return &OllamaClient{
		host:  host,
		model: model,
	}
}

// GenerateEmbedding generates an embedding for the given text
func (c *OllamaClient) GenerateEmbedding(text string) ([]float32, error) {
	reqBody := GenerateEmbeddingRequest{
		Model:  c.model,
		Prompt: text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(c.host+"/api/embeddings", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result GenerateEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Embedding, nil
}

// ConcatenateMetadata concatenates metadata into a single string for embedding
func ConcatenateMetadata(metadata map[string]string) string {
	var text string
	for key, value := range metadata {
		text += fmt.Sprintf("%s: %s\n", key, value)
	}
	return text
}
