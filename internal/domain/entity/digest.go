package entity

import (
	"fmt"
	"strings"
	"time"
)

// SearchResult wraps an Article with its vector cosine similarity score.
type SearchResult struct {
	Article         *Article `json:"article"`
	SimilarityScore float64  `json:"similarity_score"`
}

// Digest represents a curated daily digest of top articles ready for delivery.
type Digest struct {
	Date      time.Time  `json:"date"`
	Articles  []*Article `json:"articles"`
	CreatedAt time.Time  `json:"created_at"`
}

// NewDigest creates a new Digest entity.
func NewDigest(articles []*Article) *Digest {
	return &Digest{
		Date:      time.Now().UTC(),
		Articles:  articles,
		CreatedAt: time.Now().UTC(),
	}
}

// FormatHTMLChunks splits the digest into multiple safe HTML chunks,
// ensuring each message does not exceed Telegram's length limits and HTML tags
// are never broken across chunk boundaries.
func (d *Digest) FormatHTMLChunks() []string {
	dateStr := d.Date.Format("Monday, 02 Jan 2006")
	header := fmt.Sprintf("🚀 <b>AI TECH PULSE — DAILY DIGEST</b>\n<i>%s</i>\n\n", dateStr)

	if len(d.Articles) == 0 {
		return []string{header + "No new high-quality articles processed today. Check back tomorrow!\n"}
	}

	var chunks []string
	var current strings.Builder
	current.WriteString(header)

	for i, a := range d.Articles {
		var block strings.Builder
		block.WriteString(fmt.Sprintf("<b>#%d: <a href=\"%s\">%s</a></b>\n", i+1, escapeHTML(a.URL), escapeHTML(a.Title)))
		block.WriteString(fmt.Sprintf("⭐ <b>Quality Score:</b> %d/10\n", a.QualityScore))
		block.WriteString(fmt.Sprintf("📝 <b>Summary:</b>\n%s\n\n", escapeHTML(a.Summary)))
		block.WriteString("─────────────────────\n\n")

		// If adding this block exceeds 3500 chars, flush current chunk and start a new one
		if current.Len()+block.Len() > 3500 && current.Len() > len(header) {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		current.WriteString(block.String())
	}

	footer := "<i>Powered by Clean Architecture & Google Gemini</i> 🤖"
	if current.Len()+len(footer) > 3900 {
		chunks = append(chunks, current.String())
		current.Reset()
	}
	current.WriteString(footer)

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

// FormatHTML formats the digest into a clean, rich HTML message for Telegram.
func (d *Digest) FormatHTML() string {
	chunks := d.FormatHTMLChunks()
	return strings.Join(chunks, "\n\n")
}

// FormatMarkdown formats the digest as standard Markdown.
func (d *Digest) FormatMarkdown() string {
	var sb strings.Builder

	dateStr := d.Date.Format("Monday, 02 Jan 2006")
	sb.WriteString(fmt.Sprintf("# 🚀 AI Tech Pulse — Daily Digest\n*%s*\n\n", dateStr))

	if len(d.Articles) == 0 {
		sb.WriteString("No new high-quality articles processed today.\n")
		return sb.String()
	}

	for i, a := range d.Articles {
		sb.WriteString(fmt.Sprintf("### %d. [%s](%s)\n", i+1, a.Title, a.URL))
		sb.WriteString(fmt.Sprintf("**Quality Score:** %d/10\n\n", a.QualityScore))
		sb.WriteString(fmt.Sprintf("%s\n\n", a.Summary))
		sb.WriteString("---\n\n")
	}

	return sb.String()
}

func escapeHTML(text string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return r.Replace(text)
}
