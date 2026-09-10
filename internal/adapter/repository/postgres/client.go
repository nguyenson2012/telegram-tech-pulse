package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// FeedModel maps to the `feeds` table.
type FeedModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	URL       string    `gorm:"type:varchar(2048);uniqueIndex;not null"`
	Name      string    `gorm:"type:varchar(255);not null"`
	IsActive  bool      `gorm:"type:boolean;default:true;not null"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (FeedModel) TableName() string { return "feeds" }

// ArticleModel maps to the `articles` table.
type ArticleModel struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	FeedID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	Title        string     `gorm:"type:varchar(1024);not null"`
	URL          string     `gorm:"type:varchar(2048);uniqueIndex;not null"`
	RawContent   string     `gorm:"type:text;not null"`
	Summary      string     `gorm:"type:text;not null"`
	QualityScore int        `gorm:"type:integer;not null"`
	PublishedAt  *time.Time `gorm:"type:timestamptz"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (ArticleModel) TableName() string { return "articles" }

// ArticleEmbeddingModel maps to the `article_embeddings` table with vector(768).
type ArticleEmbeddingModel struct {
	ArticleID uuid.UUID       `gorm:"type:uuid;primaryKey"`
	Embedding pgvector.Vector `gorm:"type:vector(768);not null"`
	CreatedAt time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (ArticleEmbeddingModel) TableName() string { return "article_embeddings" }

// NewPostgresDB opens a GORM connection and configures pgvector extension.
func NewPostgresDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get generic database object: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Ensure vector extension exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.WithContext(ctx).Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		log.Printf("[Postgres] Warning: could not enable vector extension automatically: %v", err)
	}

	return db, nil
}
