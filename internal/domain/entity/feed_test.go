package entity_test

import (
	"testing"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
)

func TestNewFeed_Valid(t *testing.T) {
	feed, err := entity.NewFeed("https://techcrunch.com/feed/", "TechCrunch")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if feed.URL != "https://techcrunch.com/feed/" {
		t.Errorf("expected URL https://techcrunch.com/feed/, got %s", feed.URL)
	}
	if feed.Name != "TechCrunch" {
		t.Errorf("expected Name TechCrunch, got %s", feed.Name)
	}
	if !feed.IsActive {
		t.Errorf("expected IsActive to be true")
	}
}

func TestNewFeed_InvalidURL(t *testing.T) {
	_, err := entity.NewFeed("not-a-valid-url", "Invalid Feed")
	if err != entity.ErrInvalidFeedURL {
		t.Fatalf("expected ErrInvalidFeedURL, got: %v", err)
	}
}

func TestNewFeed_EmptyName(t *testing.T) {
	_, err := entity.NewFeed("https://example.com/rss", "   ")
	if err != entity.ErrEmptyFeedName {
		t.Fatalf("expected ErrEmptyFeedName, got: %v", err)
	}
}
