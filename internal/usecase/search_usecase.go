package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/domain/repository"
	"github.com/ai-tech-pulse/digest/internal/domain/service"
)

var (
	ErrEmptySearchQuery = errors.New("search query cannot be empty")
)

// SearchUseCase coordinates semantic search using vector embeddings.
type SearchUseCase struct {
	vectorRepo repository.VectorRepository
	aiService  service.AIService
}

// NewSearchUseCase creates a new SearchUseCase instance.
func NewSearchUseCase(vectorRepo repository.VectorRepository, aiService service.AIService) *SearchUseCase {
	return &SearchUseCase{
		vectorRepo: vectorRepo,
		aiService:  aiService,
	}
}

// Search generates an embedding for the text query and performs cosine similarity search.
func (uc *SearchUseCase) Search(ctx context.Context, query string, limit int) ([]*entity.SearchResult, error) {
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, ErrEmptySearchQuery
	}

	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	// 1. Generate embedding for query via Gemini
	queryEmbedding, err := uc.aiService.GenerateEmbedding(ctx, cleanQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 2. Perform Cosine Similarity Search in PostgreSQL pgvector
	results, err := uc.vectorRepo.SearchSimilar(ctx, queryEmbedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}

	return results, nil
}
