package telegram

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
)

func TestStripHTMLTags(t *testing.T) {
	input := "<b>#1: <a href=\"https://example.com?a=1&amp;b=2\">Title &amp; Test</a></b>\n<i>Summary</i>"
	expected := "#1: Title & Test\nSummary"
	actual := stripHTMLTags(input)
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestTelegramNotifier_MockConsole(t *testing.T) {
	notifier := NewTelegramNotifier("", "")
	ctx := context.Background()

	now := time.Now().UTC()
	art, _ := entity.NewArticle("f1", "Test Title", "https://example.com", "Content", "Summary", 8, &now)
	digest := entity.NewDigest([]*entity.Article{art})

	err := notifier.SendDigest(ctx, digest)
	if err != nil {
		t.Fatalf("expected nil error for console fallback, got %v", err)
	}
}

func TestFormatHTMLChunks_Boundaries(t *testing.T) {
	now := time.Now().UTC()
	var articles []*entity.Article
	for i := 0; i < 15; i++ {
		longSummary := strings.Repeat("This is a detailed summary of a tech article. ", 15)
		art, _ := entity.NewArticle("f1", "Sample Article Title", "https://example.com/test", "Content", longSummary, 7, &now)
		articles = append(articles, art)
	}

	digest := entity.NewDigest(articles)
	chunks := digest.FormatHTMLChunks()

	if len(chunks) <= 1 {
		t.Errorf("expected multiple chunks for 15 long articles, got %d", len(chunks))
	}

	for idx, chunk := range chunks {
		if len(chunk) > 4000 {
			t.Errorf("chunk %d exceeded 4000 chars: %d", idx, len(chunk))
		}
		// Verify no unclosed <a> or <b> tags
		openA := strings.Count(chunk, "<a ")
		closeA := strings.Count(chunk, "</a>")
		if openA != closeA {
			t.Errorf("chunk %d has mismatched <a> tags: %d open vs %d close", idx, openA, closeA)
		}
		openB := strings.Count(chunk, "<b>")
		closeB := strings.Count(chunk, "</b>")
		if openB != closeB {
			t.Errorf("chunk %d has mismatched <b> tags: %d open vs %d close", idx, openB, closeB)
		}
	}
}
