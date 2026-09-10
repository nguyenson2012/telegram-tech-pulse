package entity_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
)

func TestDigest_FormatHTML(t *testing.T) {
	now := time.Now().UTC()
	art, err := entity.NewArticle(
		"feed-1",
		"Google Unveils Gemini 2.0 & Next-Gen Architecture",
		"https://example.com/gemini-2",
		"Raw content about Gemini 2.0",
		"1. Revolutionary reasoning capabilities.\n2. Ultra-fast latency.\n3. Native tool integration.",
		9,
		&now,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	digest := entity.NewDigest([]*entity.Article{art})
	html := digest.FormatHTML()

	if !strings.Contains(html, "AI TECH PULSE — DAILY DIGEST") {
		t.Errorf("expected HTML to contain header")
	}
	if !strings.Contains(html, "Google Unveils Gemini 2.0 &amp; Next-Gen Architecture") {
		t.Errorf("expected HTML escaping of & to &amp;")
	}
	if !strings.Contains(html, "Quality Score:</b> 9/10") {
		t.Errorf("expected quality score 9/10")
	}
	if !strings.Contains(html, "href=\"https://example.com/gemini-2\"") {
		t.Errorf("expected article URL link")
	}
}

func TestDigest_EmptyArticles(t *testing.T) {
	digest := entity.NewDigest([]*entity.Article{})
	html := digest.FormatHTML()

	if !strings.Contains(html, "No new high-quality articles processed today") {
		t.Errorf("expected empty message")
	}
}
