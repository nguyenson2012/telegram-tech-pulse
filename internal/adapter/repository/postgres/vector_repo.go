package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type vectorRepo struct {
	db *gorm.DB
}

// NewVectorRepository instantiates a new PostgreSQL + pgvector VectorRepository.
func NewVectorRepository(db *gorm.DB) repository.VectorRepository {
	return &vectorRepo{db: db}
}

// SaveEmbedding persists the 768-dim vector embedding associated with an article.
func (r *vectorRepo) SaveEmbedding(ctx context.Context, articleID string, embedding []float32) error {
	uid, err := uuid.Parse(articleID)
	if err != nil {
		return fmt.Errorf("invalid article id: %w", err)
	}

	vec := pgvector.NewVector(embedding)
	model := ArticleEmbeddingModel{
		ArticleID: uid,
		Embedding: vec,
		CreatedAt: time.Now().UTC(),
	}

	// Upsert on primary key conflict
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "article_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"embedding", "created_at"}),
	}).Create(&model).Error
}

type searchRawResult struct {
	ID              uuid.UUID  `gorm:"column:id"`
	FeedID          uuid.UUID  `gorm:"column:feed_id"`
	Title           string     `gorm:"column:title"`
	URL             string     `gorm:"column:url"`
	RawContent      string     `gorm:"column:raw_content"`
	Summary         string     `gorm:"column:summary"`
	QualityScore    int        `gorm:"column:quality_score"`
	PublishedAt     *time.Time `gorm:"column:published_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	SimilarityScore float64    `gorm:"column:similarity_score"`
}

// SearchSimilar executes a cosine similarity search against pgvector.
func (r *vectorRepo) SearchSimilar(ctx context.Context, queryEmbedding []float32, topK int) ([]*entity.SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	vec := pgvector.NewVector(queryEmbedding)

	querySQL := `
		SELECT 
			a.id, 
			a.feed_id, 
			a.title, 
			a.url, 
			a.raw_content, 
			a.summary, 
			a.quality_score, 
			a.published_at, 
			a.created_at,
			(1.0 - (ae.embedding <=> ?)) AS similarity_score
		FROM article_embeddings ae
		JOIN articles a ON a.id = ae.article_id
		ORDER BY ae.embedding <=> ?
		LIMIT ?
	`

	var rows []searchRawResult
	if err := r.db.WithContext(ctx).Raw(querySQL, vec, vec, topK).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("vector similarity query failed: %w", err)
	}

	results := make([]*entity.SearchResult, len(rows))
	for i, row := range rows {
		results[i] = &entity.SearchResult{
			Article: &entity.Article{
				ID:           row.ID.String(),
				FeedID:       row.FeedID.String(),
				Title:        row.Title,
				URL:          row.URL,
				RawContent:   row.RawContent,
				Summary:      row.Summary,
				QualityScore: row.QualityScore,
				PublishedAt:  row.PublishedAt,
				CreatedAt:    row.CreatedAt,
			},
			SimilarityScore: row.SimilarityScore,
		}
	}

	return results, nil
}
