package rss

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/service"
	"github.com/mmcdole/gofeed"
)

var htmlTagRegex = regexp.MustCompile("<[^>]*>")

type rssParser struct {
	fp *gofeed.Parser
}

// NewRSSParser creates a new RSS/Atom parser adapter using gofeed.
func NewRSSParser() service.RSSParser {
	fp := gofeed.NewParser()
	fp.Client = nil // Uses default http client with standard timeouts
	return &rssParser{fp: fp}
}

// ParseURL fetches the RSS/Atom feed from feedURL and extracts articles.
func (p *rssParser) ParseURL(ctx context.Context, feedURL string) ([]*service.RSSItem, error) {
	feed, err := p.fp.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed from %s: %w", feedURL, err)
	}

	items := make([]*service.RSSItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		if item == nil || strings.TrimSpace(item.Link) == "" {
			continue
		}

		// Prefer full Content over Description/Summary
		content := item.Content
		if strings.TrimSpace(content) == "" {
			content = item.Description
		}
		cleanContent := stripHTML(content)

		var pubTime *time.Time
		if item.PublishedParsed != nil {
			pubTime = item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			pubTime = item.UpdatedParsed
		}

		items = append(items, &service.RSSItem{
			Title:       strings.TrimSpace(item.Title),
			URL:         strings.TrimSpace(item.Link),
			Content:     cleanContent,
			PublishedAt: pubTime,
		})
	}

	return items, nil
}

// ValidateFeed checks connectivity to the feed and returns its title.
func (p *rssParser) ValidateFeed(ctx context.Context, feedURL string) (string, error) {
	feed, err := p.fp.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return "", fmt.Errorf("invalid feed URL: %w", err)
	}
	return feed.Title, nil
}

func stripHTML(input string) string {
	cleaned := htmlTagRegex.ReplaceAllString(input, " ")
	return strings.TrimSpace(strings.Join(strings.Fields(cleaned), " "))
}
