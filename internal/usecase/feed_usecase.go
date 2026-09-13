package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/domain/repository"
	"github.com/ai-tech-pulse/digest/internal/domain/service"
)

// AddFeedRequest represents input payload for adding a new feed.
type AddFeedRequest struct {
	URL  string `json:"url"`
	Name string `json:"name,omitempty"`
}

// FeedUseCase coordinates feed management operations.
type FeedUseCase struct {
	feedRepo  repository.FeedRepository
	rssParser service.RSSParser
}

// NewFeedUseCase creates a new FeedUseCase.
func NewFeedUseCase(feedRepo repository.FeedRepository, rssParser service.RSSParser) *FeedUseCase {
	return &FeedUseCase{
		feedRepo:  feedRepo,
		rssParser: rssParser,
	}
}

// AddFeed validates the RSS feed, auto-discovers title if empty, and stores the feed.
func (uc *FeedUseCase) AddFeed(ctx context.Context, req AddFeedRequest) (*entity.Feed, error) {
	cleanURL := strings.TrimSpace(req.URL)
	if cleanURL == "" {
		return nil, entity.ErrInvalidFeedURL
	}

	// Check if feed URL already exists
	existing, err := uc.feedRepo.GetFeedByURL(ctx, cleanURL)
	if err == nil && existing != nil {
		return nil, entity.ErrFeedURLConflict
	}

	// Auto-discover title from RSS if not provided
	name := strings.TrimSpace(req.Name)
	if name == "" && uc.rssParser != nil {
		discoveredTitle, err := uc.rssParser.ValidateFeed(ctx, cleanURL)
		if err == nil && discoveredTitle != "" {
			name = discoveredTitle
		}
	}
	if name == "" {
		name = cleanURL
	}

	feed, err := entity.NewFeed(cleanURL, name)
	if err != nil {
		return nil, err
	}

	if err := uc.feedRepo.CreateFeed(ctx, feed); err != nil {
		return nil, fmt.Errorf("failed to save feed: %w", err)
	}

	return feed, nil
}

// ListFeeds returns all tracked feeds.
func (uc *FeedUseCase) ListFeeds(ctx context.Context) ([]*entity.Feed, error) {
	return uc.feedRepo.GetAllFeeds(ctx)
}

// ToggleFeedStatus activates or deactivates a feed.
func (uc *FeedUseCase) ToggleFeedStatus(ctx context.Context, id string, isActive bool) error {
	return uc.feedRepo.UpdateFeedStatus(ctx, id, isActive)
}

// DeleteFeed permanently removes a feed by ID.
func (uc *FeedUseCase) DeleteFeed(ctx context.Context, id string) error {
	return uc.feedRepo.DeleteFeed(ctx, id)
}

