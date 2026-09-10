package usecase_test

import (
	"context"
	"testing"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/domain/service"
	"github.com/ai-tech-pulse/digest/internal/usecase"
)

// MockFeedRepo implements repository.FeedRepository for testing.
type MockFeedRepo struct {
	feeds []*entity.Feed
}

func (m *MockFeedRepo) CreateFeed(ctx context.Context, feed *entity.Feed) error {
	m.feeds = append(m.feeds, feed)
	return nil
}

func (m *MockFeedRepo) GetFeedByID(ctx context.Context, id string) (*entity.Feed, error) {
	for _, f := range m.feeds {
		if f.ID == id {
			return f, nil
		}
	}
	return nil, entity.ErrFeedNotFound
}

func (m *MockFeedRepo) GetFeedByURL(ctx context.Context, url string) (*entity.Feed, error) {
	for _, f := range m.feeds {
		if f.URL == url {
			return f, nil
		}
	}
	return nil, nil
}

func (m *MockFeedRepo) GetActiveFeeds(ctx context.Context) ([]*entity.Feed, error) {
	return m.feeds, nil
}

func (m *MockFeedRepo) GetAllFeeds(ctx context.Context) ([]*entity.Feed, error) {
	return m.feeds, nil
}

func (m *MockFeedRepo) UpdateFeedStatus(ctx context.Context, id string, isActive bool) error {
	for _, f := range m.feeds {
		if f.ID == id {
			f.IsActive = isActive
			return nil
		}
	}
	return entity.ErrFeedNotFound
}

func (m *MockFeedRepo) CreateArticle(ctx context.Context, article *entity.Article) error {
	return nil
}

func (m *MockFeedRepo) ArticleExistsByURL(ctx context.Context, url string) (bool, error) {
	return false, nil
}

func (m *MockFeedRepo) GetArticleByID(ctx context.Context, id string) (*entity.Article, error) {
	return nil, entity.ErrArticleNotFound
}

func (m *MockFeedRepo) GetTopArticlesToday(ctx context.Context, limit int) ([]*entity.Article, error) {
	return nil, nil
}

// MockRSSParser implements service.RSSParser for testing.
type MockRSSParser struct{}

func (m *MockRSSParser) ParseURL(ctx context.Context, feedURL string) ([]*service.RSSItem, error) {
	return nil, nil
}

func (m *MockRSSParser) ValidateFeed(ctx context.Context, feedURL string) (string, error) {
	return "Discovered Tech Feed", nil
}

func TestFeedUseCase_AddFeed(t *testing.T) {
	repo := &MockFeedRepo{}
	parser := &MockRSSParser{}
	uc := usecase.NewFeedUseCase(repo, parser)

	ctx := context.Background()

	// 1. Add feed with explicit name
	feed, err := uc.AddFeed(ctx, usecase.AddFeedRequest{
		URL:  "https://news.ycombinator.com/rss",
		Name: "Hacker News",
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if feed.Name != "Hacker News" {
		t.Errorf("expected Hacker News, got %s", feed.Name)
	}

	// 2. Add feed with empty name (auto-discovery)
	feed2, err := uc.AddFeed(ctx, usecase.AddFeedRequest{
		URL: "https://example.com/rss",
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if feed2.Name != "Discovered Tech Feed" {
		t.Errorf("expected Discovered Tech Feed, got %s", feed2.Name)
	}

	// 3. Add duplicate feed URL
	_, err = uc.AddFeed(ctx, usecase.AddFeedRequest{
		URL: "https://news.ycombinator.com/rss",
	})
	if err != entity.ErrFeedURLConflict {
		t.Errorf("expected ErrFeedURLConflict, got: %v", err)
	}
}
