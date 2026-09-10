package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/domain/service"
	"github.com/ai-tech-pulse/digest/internal/usecase"
)

// MockVectorRepo implements repository.VectorRepository for testing.
type MockVectorRepo struct{}

func (m *MockVectorRepo) SaveEmbedding(ctx context.Context, articleID string, embedding []float32) error {
	return nil
}

func (m *MockVectorRepo) SearchSimilar(ctx context.Context, queryEmbedding []float32, topK int) ([]*entity.SearchResult, error) {
	now := time.Now().UTC()
	art := &entity.Article{
		ID:           "art-1",
		FeedID:       "feed-1",
		Title:        "Autonomous AI Agents in Go",
		URL:          "https://example.com/ai-agents-go",
		Summary:      "Overview of autonomous agents.",
		QualityScore: 9,
		PublishedAt:  &now,
		CreatedAt:    now,
	}
	return []*entity.SearchResult{
		{Article: art, SimilarityScore: 0.94},
	}, nil
}

// MockAIService implements service.AIService for testing.
type MockAIService struct{}

func (m *MockAIService) SummarizeAndScore(ctx context.Context, title, content string) (*service.SummaryResult, error) {
	return &service.SummaryResult{
		ThreeLineSummary: "Test summary",
		KeyTakeaways:     []string{"Takeaway 1", "Takeaway 2"},
		QualityScore:     8,
	}, nil
}

func (m *MockAIService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return make([]float32, 768), nil
}

func TestSearchUseCase_Search(t *testing.T) {
	vecRepo := &MockVectorRepo{}
	aiSvc := &MockAIService{}
	uc := usecase.NewSearchUseCase(vecRepo, aiSvc)

	ctx := context.Background()

	// 1. Valid search query
	results, err := uc.Search(ctx, "autonomous agents in Go", 5)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Article.Title != "Autonomous AI Agents in Go" {
		t.Errorf("expected Autonomous AI Agents in Go, got %s", results[0].Article.Title)
	}
	if results[0].SimilarityScore < 0.9 {
		t.Errorf("expected similarity score > 0.9, got %f", results[0].SimilarityScore)
	}

	// 2. Empty query error
	_, err = uc.Search(ctx, "   ", 5)
	if err != usecase.ErrEmptySearchQuery {
		t.Errorf("expected ErrEmptySearchQuery, got: %v", err)
	}
}
