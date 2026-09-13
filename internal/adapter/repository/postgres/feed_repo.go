package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/domain/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type feedRepo struct {
	db *gorm.DB
}

// NewFeedRepository instantiates a new PostgreSQL-backed FeedRepository.
func NewFeedRepository(db *gorm.DB) repository.FeedRepository {
	return &feedRepo{db: db}
}

func (r *feedRepo) CreateFeed(ctx context.Context, feed *entity.Feed) error {
	feedID, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	feed.ID = feedID.String()

	model := FeedModel{
		ID:        feedID,
		URL:       feed.URL,
		Name:      feed.Name,
		IsActive:  feed.IsActive,
		CreatedAt: feed.CreatedAt,
		UpdatedAt: feed.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}

	return nil
}

func (r *feedRepo) GetFeedByID(ctx context.Context, id string) (*entity.Feed, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, entity.ErrFeedNotFound
	}

	var model FeedModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrFeedNotFound
		}
		return nil, err
	}

	return toFeedEntity(&model), nil
}

func (r *feedRepo) GetFeedByURL(ctx context.Context, feedURL string) (*entity.Feed, error) {
	var model FeedModel
	if err := r.db.WithContext(ctx).First(&model, "url = ?", feedURL).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return toFeedEntity(&model), nil
}

func (r *feedRepo) GetActiveFeeds(ctx context.Context) ([]*entity.Feed, error) {
	var models []FeedModel
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("name ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]*entity.Feed, len(models))
	for i := range models {
		result[i] = toFeedEntity(&models[i])
	}
	return result, nil
}

func (r *feedRepo) GetAllFeeds(ctx context.Context) ([]*entity.Feed, error) {
	var models []FeedModel
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]*entity.Feed, len(models))
	for i := range models {
		result[i] = toFeedEntity(&models[i])
	}
	return result, nil
}

func (r *feedRepo) UpdateFeedStatus(ctx context.Context, id string, isActive bool) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return entity.ErrFeedNotFound
	}

	res := r.db.WithContext(ctx).Model(&FeedModel{}).Where("id = ?", uid).Updates(map[string]interface{}{
		"is_active":  isActive,
		"updated_at": time.Now().UTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return entity.ErrFeedNotFound
	}

	return nil
}

func (r *feedRepo) DeleteFeed(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return entity.ErrFeedNotFound
	}

	res := r.db.WithContext(ctx).Delete(&FeedModel{}, "id = ?", uid)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return entity.ErrFeedNotFound
	}

	return nil
}

func (r *feedRepo) CreateArticle(ctx context.Context, article *entity.Article) error {
	articleID, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	article.ID = articleID.String()

	feedUID, err := uuid.Parse(article.FeedID)
	if err != nil {
		return fmt.Errorf("invalid feed id: %w", err)
	}

	model := ArticleModel{
		ID:           articleID,
		FeedID:       feedUID,
		Title:        article.Title,
		URL:          article.URL,
		RawContent:   article.RawContent,
		Summary:      article.Summary,
		QualityScore: article.QualityScore,
		PublishedAt:  article.PublishedAt,
		CreatedAt:    article.CreatedAt,
	}

	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *feedRepo) ArticleExistsByURL(ctx context.Context, articleURL string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&ArticleModel{}).Where("url = ?", articleURL).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *feedRepo) GetArticleByID(ctx context.Context, id string) (*entity.Article, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, entity.ErrArticleNotFound
	}

	var model ArticleModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrArticleNotFound
		}
		return nil, err
	}

	return toArticleEntity(&model), nil
}

func (r *feedRepo) GetTopArticlesToday(ctx context.Context, limit int) ([]*entity.Article, error) {
	if limit <= 0 {
		limit = 5
	}

	// Beginning of today (UTC)
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var models []ArticleModel
	// Query articles from today, or fallback to most recent articles if none today
	err := r.db.WithContext(ctx).
		Where("created_at >= ?", startOfDay).
		Order("quality_score DESC, created_at DESC").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	// Fallback to latest highest score if none ingested today
	if len(models) == 0 {
		err = r.db.WithContext(ctx).
			Order("quality_score DESC, created_at DESC").
			Limit(limit).
			Find(&models).Error
		if err != nil {
			return nil, err
		}
	}

	result := make([]*entity.Article, len(models))
	for i := range models {
		result[i] = toArticleEntity(&models[i])
	}
	return result, nil
}

func toFeedEntity(m *FeedModel) *entity.Feed {
	return &entity.Feed{
		ID:        m.ID.String(),
		URL:       m.URL,
		Name:      m.Name,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func toArticleEntity(m *ArticleModel) *entity.Article {
	return &entity.Article{
		ID:           m.ID.String(),
		FeedID:       m.FeedID.String(),
		Title:        m.Title,
		URL:          m.URL,
		RawContent:   m.RawContent,
		Summary:      m.Summary,
		QualityScore: m.QualityScore,
		PublishedAt:  m.PublishedAt,
		CreatedAt:    m.CreatedAt,
	}
}
