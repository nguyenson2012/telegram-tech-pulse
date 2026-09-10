package gemini

import (
	"context"
	"math"
	"os"
	"testing"
)

func TestMockAIService_Fallback(t *testing.T) {
	ctx := context.Background()
	svc, err := NewGeminiService(ctx, "", "gemini-3.6-flash", "gemini-embedding-2")
	if err != nil {
		t.Fatalf("unexpected error initializing mock: %v", err)
	}

	// 1. Test Mock Summarization
	summary, err := svc.SummarizeAndScore(ctx, "Test Title", "Test Content")
	if err != nil {
		t.Fatalf("unexpected error in mock summarize: %v", err)
	}
	if summary.QualityScore < 1 || summary.QualityScore > 10 {
		t.Errorf("expected quality score 1-10, got %d", summary.QualityScore)
	}
	if summary.ThreeLineSummary == "" {
		t.Errorf("expected non-empty three line summary")
	}

	// 2. Test Mock Embedding
	vec, err := svc.GenerateEmbedding(ctx, "Test Text")
	if err != nil {
		t.Fatalf("unexpected error in mock embedding: %v", err)
	}
	if len(vec) != 768 {
		t.Errorf("expected 768 dimensions for pgvector, got %d", len(vec))
	}
}

func TestTruncateAndNormalize_MRL(t *testing.T) {
	// Create mock 3072-dim vector
	input := make([]float32, 3072)
	for i := range input {
		input[i] = float32(i%17) - 8.5
	}

	result := truncateAndNormalize(input, 768)
	if len(result) != 768 {
		t.Fatalf("expected 768 dimensions, got %d", len(result))
	}

	// Verify L2 norm is ~ 1.0
	var sumSq float64
	for _, v := range result {
		sumSq += float64(v) * float64(v)
	}
	norm := math.Sqrt(sumSq)
	if math.Abs(norm-1.0) > 1e-5 {
		t.Errorf("expected normalized L2 norm of 1.0, got %f", norm)
	}
}

func TestLiveEmbedding(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping live embedding test: GEMINI_API_KEY environment variable is not set")
	}
	ctx := context.Background()
	svc, err := NewGeminiService(ctx, apiKey, "gemini-3.5-flash", "gemini-embedding-2")
	if err != nil {
		t.Fatalf("failed to initialize gemini client: %v", err)
	}

	vec, err := svc.GenerateEmbedding(ctx, "Vector database and pgvector in PostgreSQL")
	if err != nil {
		t.Fatalf("failed to generate live embedding: %v", err)
	}

	if len(vec) != 768 {
		t.Fatalf("expected 768-dim vector, got %d", len(vec))
	}
	t.Logf("Successfully generated live embedding with %d dimensions!", len(vec))
}
