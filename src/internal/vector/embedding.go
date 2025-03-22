package embedding

// EmbeddingService defines the interface for generating embeddings
type EmbeddingService interface {
	GenerateEmbedding(text string) ([]float32, error)
}

// MetadataToEmbedding converts metadata map to an embedding using the provided service
func MetadataToEmbedding(service EmbeddingService, metadata map[string]string) ([]float32, error) {
	text := ConcatenateMetadata(metadata)
	return service.GenerateEmbedding(text)
}
