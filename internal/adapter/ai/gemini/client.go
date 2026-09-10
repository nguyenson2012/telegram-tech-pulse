package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/ai-tech-pulse/digest/internal/domain/service"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type geminiClient struct {
	client              *genai.Client
	modelName           string
	embeddingModelName  string
	apiKey              string
}

// NewGeminiService initializes the Gemini AI client with official Google GenAI SDK.
func NewGeminiService(ctx context.Context, apiKey, modelName, embeddingModelName string) (service.AIService, error) {
	if apiKey == "" {
		log.Println("[Gemini] Warning: GEMINI_API_KEY is not set. Mock AI mode will be used as fallback.")
		return &mockAIService{}, nil
	}

	if modelName == "" {
		modelName = "gemini-3.6-flash"
	}
	if embeddingModelName == "" {
		embeddingModelName = "gemini-embedding-2"
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &geminiClient{
		client:             client,
		modelName:          modelName,
		embeddingModelName: embeddingModelName,
		apiKey:             apiKey,
	}, nil
}

// SummarizeAndScore prompts Gemini with structured JSON schema.
func (g *geminiClient) SummarizeAndScore(ctx context.Context, title, content string) (*service.SummaryResult, error) {
	model := g.client.GenerativeModel(g.modelName)
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"three_line_summary": {
				Type:        genai.TypeString,
				Description: "Exactly 3 concise lines/sentences summarizing the key message of the article.",
			},
			"key_takeaways": {
				Type:        genai.TypeArray,
				Items:       &genai.Schema{Type: genai.TypeString},
				Description: "Exactly 2 high-impact key takeaways or insights.",
			},
			"quality_score": {
				Type:        genai.TypeInteger,
				Description: "Integer score from 1 to 10 assessing technical depth, clarity, and novelty.",
			},
		},
		Required: []string{"three_line_summary", "key_takeaways", "quality_score"},
	}

	prompt := fmt.Sprintf(`You are an expert tech curator and AI engineer analyzing engineering articles for AI Tech Pulse Digest.
Analyze the following article and output strictly according to the provided JSON schema:
- three_line_summary: Exactly 3 concise sentences capturing the core essence.
- key_takeaways: Exactly 2 actionable takeaways.
- quality_score: An integer from 1 (clickbait/fluff) to 10 (exceptional deep-dive).

Article Title: %s
Article Content:
%s
`, title, truncateText(content, 12000))

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		// If gemini-3.6-flash hits quota limit (429), attempt fallback to gemini-3.5-flash
		if g.modelName != "gemini-3.5-flash" {
			log.Printf("[Gemini] Warning: Model %s call failed (%v). Attempting fallback to gemini-3.5-flash...", g.modelName, err)
			fallbackModel := g.client.GenerativeModel("gemini-3.5-flash")
			fallbackModel.ResponseMIMEType = "application/json"
			fallbackModel.ResponseSchema = model.ResponseSchema
			resp, err = fallbackModel.GenerateContent(ctx, genai.Text(prompt))
		}
		if err != nil {
			return nil, fmt.Errorf("gemini generate content failed: %w", err)
		}
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty response returned by Gemini")
	}

	var jsonOutput string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			jsonOutput += string(txt)
		}
	}

	var result service.SummaryResult
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini JSON schema response: %w, raw output: %s", err, jsonOutput)
	}

	// Clamp quality score to 1-10
	if result.QualityScore < 1 {
		result.QualityScore = 1
	}
	if result.QualityScore > 10 {
		result.QualityScore = 10
	}

	return &result, nil
}

// GenerateEmbedding creates a 768-dimensional float32 vector embedding for text.
// gemini-embedding-2 outputs native 3072-dimensional vectors. To maintain compatibility
// with pgvector HNSW index (max 2000 dims), we apply Matryoshka Representation Learning
// (MRL) reduction by taking the first 768 dimensions and L2-normalizing the vector.
func (g *geminiClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	cleanText := strings.TrimSpace(text)
	if cleanText == "" {
		return nil, errors.New("cannot embed empty text")
	}

	em := g.client.EmbeddingModel(g.embeddingModelName)
	res, err := em.EmbedContent(ctx, genai.Text(truncateText(cleanText, 8000)))
	if err != nil {
		return nil, fmt.Errorf("gemini embed content failed: %w", err)
	}

	if res.Embedding == nil || len(res.Embedding.Values) == 0 {
		return nil, errors.New("empty embedding returned by Gemini")
	}

	return truncateAndNormalize(res.Embedding.Values, 768), nil
}

// truncateAndNormalize reduces vector dimensions using Matryoshka Representation Learning (MRL)
// and normalizes the slice using L2 norm.
func truncateAndNormalize(vec []float32, targetDim int) []float32 {
	if len(vec) <= targetDim {
		return vec
	}
	sub := vec[:targetDim]
	var sumSq float64
	for _, v := range sub {
		sumSq += float64(v) * float64(v)
	}
	if sumSq == 0 {
		return sub
	}
	norm := float32(math.Sqrt(sumSq))
	res := make([]float32, targetDim)
	for i, v := range sub {
		res[i] = v / norm
	}
	return res
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// mockAIService is a fallback when no API key is set (e.g. in offline tests).
type mockAIService struct{}

func (m *mockAIService) SummarizeAndScore(ctx context.Context, title, content string) (*service.SummaryResult, error) {
	return &service.SummaryResult{
		ThreeLineSummary: fmt.Sprintf("1. %s discusses core concepts.\n2. Deep dive into practical implementation.\n3. Highlights future directions and impact.", title),
		KeyTakeaways: []string{
			"Modular architecture improves long-term maintainability.",
			"Vector embeddings enable fast semantic retrieval.",
		},
		QualityScore: 8,
	}, nil
}

func (m *mockAIService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Return deterministic mock 768-dim vector for testing
	vec := make([]float32, 768)
	for i := range vec {
		vec[i] = float32(i%10) / 10.0
	}
	return vec, nil
}
