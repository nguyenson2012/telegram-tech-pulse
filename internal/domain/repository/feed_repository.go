package repository

import (
	"context"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
)

// FeedRepository defines persistence operations for feeds and articles.
type FeedRepository interface {
	// Feed operations
	CreateFeed(ctx context.Context, feed *entity.Feed) error
	GetFeedByID(ctx context.Context, id string) (*entity.Feed, error)
	GetFeedByURL(ctx context.Context, url string) (*entity.Feed, error)
	GetActiveFeeds(ctx context.Context) ([]*entity.Feed, error)
	GetAllFeeds(ctx context.Context) ([]*entity.Feed, error)
	UpdateFeedStatus(ctx context.Context, id string, isActive bool) error
	DeleteFeed(ctx context.Context, id string) error

	// Article operations
	CreateArticle(ctx context.Context, article *entity.Article) error
	ArticleExistsByURL(ctx context.Context, url string) (bool, error)
	GetArticleByID(ctx context.Context, id string) (*entity.Article, error)
	GetTopArticlesToday(ctx context.Context, limit int) ([]*entity.Article, error)
}
