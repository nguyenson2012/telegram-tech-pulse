package service

import (
	"context"
	"time"
)

// RSSItem represents an individual article item parsed from an RSS/Atom feed.
type RSSItem struct {
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Content     string     `json:"content"`
	PublishedAt *time.Time `json:"published_at"`
}

// RSSParser defines the port for fetching and parsing remote RSS/Atom feeds.
type RSSParser interface {
	// ParseURL fetches and extracts article items from the specified feed URL.
	ParseURL(ctx context.Context, feedURL string) ([]*RSSItem, error)

	// ValidateFeed verifies the feed is reachable and returns its title.
	ValidateFeed(ctx context.Context, feedURL string) (title string, err error)
}
