package repository

import (
	"context"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
)

// VectorRepository defines the port for vector embedding persistence and similarity search.
type VectorRepository interface {
	// SaveEmbedding persists an article embedding vector into the vector store.
	SaveEmbedding(ctx context.Context, articleID string, embedding []float32) error

	// SearchSimilar executes a cosine similarity search returning the topK most similar articles.
	SearchSimilar(ctx context.Context, queryEmbedding []float32, topK int) ([]*entity.SearchResult, error)
}
