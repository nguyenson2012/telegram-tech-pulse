package service

import (
	"context"
	"fmt"
	"strings"
)

// SummaryResult holds the structured AI output for an article.
type SummaryResult struct {
	ThreeLineSummary string   `json:"three_line_summary"`
	KeyTakeaways     []string `json:"key_takeaways"`
	QualityScore     int      `json:"quality_score"`
}

// FormattedSummary generates a cohesive text representation of the summary and takeaways.
func (s *SummaryResult) FormattedSummary() string {
	var sb strings.Builder
	sb.WriteString(s.ThreeLineSummary)
	if len(s.KeyTakeaways) > 0 {
		sb.WriteString("\n\n💡 Key Takeaways:\n")
		for _, kt := range s.KeyTakeaways {
			sb.WriteString(fmt.Sprintf("• %s\n", kt))
		}
	}
	return strings.TrimSpace(sb.String())
}

// AIService defines the port for LLM-based summarization, scoring, and vector embedding.
type AIService interface {
	// SummarizeAndScore produces a 3-line summary, 2 key takeaways, and a quality score (1-10).
	SummarizeAndScore(ctx context.Context, title, content string) (*SummaryResult, error)

	// GenerateEmbedding creates a 768-dimensional float32 vector embedding for text.
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}
