package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/domain/service"
	"github.com/ai-tech-pulse/digest/internal/usecase"
)

type mockFeedRepo struct {
	feeds []*entity.Feed
}

func (m *mockFeedRepo) CreateFeed(ctx context.Context, feed *entity.Feed) error {
	if feed.ID == "" {
		feed.ID = "feed-uuid-1"
	}
	m.feeds = append(m.feeds, feed)
	return nil
}

func (m *mockFeedRepo) GetFeedByID(ctx context.Context, id string) (*entity.Feed, error) {
	for _, f := range m.feeds {
		if f.ID == id {
			return f, nil
		}
	}
	return nil, entity.ErrFeedNotFound
}

func (m *mockFeedRepo) GetFeedByURL(ctx context.Context, url string) (*entity.Feed, error) {
	for _, f := range m.feeds {
		if f.URL == url {
			return f, nil
		}
	}
	return nil, nil
}

func (m *mockFeedRepo) GetActiveFeeds(ctx context.Context) ([]*entity.Feed, error) {
	return m.feeds, nil
}

func (m *mockFeedRepo) GetAllFeeds(ctx context.Context) ([]*entity.Feed, error) {
	return m.feeds, nil
}

func (m *mockFeedRepo) UpdateFeedStatus(ctx context.Context, id string, isActive bool) error {
	for _, f := range m.feeds {
		if f.ID == id {
			f.IsActive = isActive
			return nil
		}
	}
	return entity.ErrFeedNotFound
}

func (m *mockFeedRepo) DeleteFeed(ctx context.Context, id string) error {
	for i, f := range m.feeds {
		if f.ID == id {
			m.feeds = append(m.feeds[:i], m.feeds[i+1:]...)
			return nil
		}
	}
	return entity.ErrFeedNotFound
}

func (m *mockFeedRepo) CreateArticle(ctx context.Context, article *entity.Article) error {
	return nil
}

func (m *mockFeedRepo) ArticleExistsByURL(ctx context.Context, url string) (bool, error) {
	return false, nil
}

func (m *mockFeedRepo) GetArticleByID(ctx context.Context, id string) (*entity.Article, error) {
	return nil, entity.ErrArticleNotFound
}

func (m *mockFeedRepo) GetTopArticlesToday(ctx context.Context, limit int) ([]*entity.Article, error) {
	return nil, nil
}

type mockRSSParser struct{}

func (m *mockRSSParser) ParseURL(ctx context.Context, feedURL string) ([]*service.RSSItem, error) {
	return nil, nil
}

func (m *mockRSSParser) ValidateFeed(ctx context.Context, feedURL string) (string, error) {
	return "Mock Feed", nil
}

func TestListener_HandleCommands(t *testing.T) {
	repo := &mockFeedRepo{}
	parser := &mockRSSParser{}
	feedUC := usecase.NewFeedUseCase(repo, parser)

	var lastSentText string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req tgSendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		lastSentText = req.Text

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer ts.Close()

	listener := NewListener("dummy-token", "12345", feedUC, nil)
	listener.baseURL = ts.URL
	listener.httpClient = ts.Client()
	// Test escapeHTML
	if escapeHTML("<b>test & ok</b>") != "&lt;b&gt;test &amp; ok&lt;/b&gt;" {
		t.Errorf("escapeHTML failed")
	}

	ctx := context.Background()

	// 1. Unauthorized check
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 99999},
		Text: "/feeds",
	})
	// In production, sends unauthorized to non-matching chat ID

	// 2. Help command
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/help",
	})

	// 3. Add feed command
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/addfeed https://example.com/rss Example Feed",
	})

	feeds, _ := feedUC.ListFeeds(ctx)
	if len(feeds) != 1 {
		t.Fatalf("expected 1 feed, got %d", len(feeds))
	}
	if feeds[0].Name != "Example Feed" {
		t.Errorf("expected Example Feed, got %s", feeds[0].Name)
	}

	// 4. List feeds command
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/feeds",
	})

	// 5. Toggle feed command
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/toggle " + feeds[0].ID,
	})
	if feeds[0].IsActive {
		t.Errorf("expected feed to be deactivated")
	}

	// 6. Delete feed command
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/delfeed " + feeds[0].ID,
	})
	feedsAfter, _ := feedUC.ListFeeds(ctx)
	if len(feedsAfter) != 0 {
		t.Errorf("expected feed to be deleted, got %d", len(feedsAfter))
	}

	// 7. Unknown command
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/unknown",
	})

	// 8. Run pipeline command (without pipeline use case)
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/run",
	})

	// 9. Edge cases: empty args
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/addfeed",
	})
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/toggle",
	})
	listener.handleMessage(ctx, &tgMessage{
		Chat: &tgChat{ID: 12345},
		Text: "/delfeed",
	})

	_ = lastSentText
}

func TestListener_Lifecycle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tgGetUpdatesResponse{
			OK:     true,
			Result: []*tgUpdate{},
		})
	}))
	defer ts.Close()

	repo := &mockFeedRepo{}
	parser := &mockRSSParser{}
	feedUC := usecase.NewFeedUseCase(repo, parser)

	listener := NewListener("dummy-token", "12345", feedUC, nil)
	listener.baseURL = ts.URL
	listener.httpClient = ts.Client()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener.Start(ctx)
	// Stop should gracefully return
	listener.Stop()
}

