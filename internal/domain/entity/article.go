package entity

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidQualityScore = errors.New("quality score must be between 1 and 10")
	ErrEmptyArticleTitle   = errors.New("article title cannot be empty")
	ErrEmptyArticleURL     = errors.New("article url cannot be empty")
	ErrArticleNotFound     = errors.New("article not found")
)

// Article represents a tech article fetched from an RSS feed and processed by AI.
type Article struct {
	ID           string     `json:"id"`
	FeedID       string     `json:"feed_id"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	RawContent   string     `json:"raw_content,omitempty"`
	Summary      string     `json:"summary"`
	QualityScore int        `json:"quality_score"`
	PublishedAt  *time.Time `json:"published_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ArticleEmbedding represents the vector embedding for an article's summary.
type ArticleEmbedding struct {
	ArticleID string    `json:"article_id"`
	Embedding []float32 `json:"embedding"`
	CreatedAt time.Time `json:"created_at"`
}

// NewArticle creates a new validated Article.
func NewArticle(feedID, title, articleURL, rawContent, summary string, score int, publishedAt *time.Time) (*Article, error) {
	cleanTitle := strings.TrimSpace(title)
	cleanURL := strings.TrimSpace(articleURL)

	if cleanTitle == "" {
		return nil, ErrEmptyArticleTitle
	}
	if cleanURL == "" {
		return nil, ErrEmptyArticleURL
	}
	if score < 1 || score > 10 {
		return nil, ErrInvalidQualityScore
	}

	return &Article{
		FeedID:       feedID,
		Title:        cleanTitle,
		URL:          cleanURL,
		RawContent:   rawContent,
		Summary:      strings.TrimSpace(summary),
		QualityScore: score,
		PublishedAt:  publishedAt,
		CreatedAt:    time.Now().UTC(),
	}, nil
}
