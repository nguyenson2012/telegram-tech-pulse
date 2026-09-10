package entity

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidFeedURL  = errors.New("invalid feed url")
	ErrEmptyFeedName   = errors.New("feed name cannot be empty")
	ErrFeedNotFound    = errors.New("feed not found")
	ErrFeedURLConflict = errors.New("feed url already exists")
)

// Feed represents an RSS/Atom source tracked by the system.
type Feed struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewFeed creates a validated Feed entity.
func NewFeed(feedURL, name string) (*Feed, error) {
	cleanURL := strings.TrimSpace(feedURL)
	cleanName := strings.TrimSpace(name)

	if cleanName == "" {
		return nil, ErrEmptyFeedName
	}

	parsedURL, err := url.ParseRequestURI(cleanURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, ErrInvalidFeedURL
	}

	now := time.Now().UTC()
	return &Feed{
		URL:       cleanURL,
		Name:      cleanName,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
