package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/domain/service"
)

type telegramNotifier struct {
	botToken   string
	chatID     string
	httpClient *http.Client
}

type sendMessageRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// NewTelegramNotifier creates a new Telegram NotificationService adapter.
func NewTelegramNotifier(botToken, chatID string) service.NotificationService {
	if botToken == "" || chatID == "" {
		log.Println("[Telegram] Warning: TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is empty. Messages will be logged to console.")
	}

	return &telegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Send broadcasts a text message via Telegram Bot API.
func (t *telegramNotifier) Send(ctx context.Context, message string) error {
	if t.botToken == "" || t.chatID == "" {
		log.Printf("[Telegram:Console] Target Chat: %s\n%s", t.chatID, message)
		return nil
	}

	// Telegram max message length is 4096 characters; split if necessary
	chunks := splitMessage(message, 3800)
	for _, chunk := range chunks {
		if err := t.sendChunkWithFallback(ctx, chunk); err != nil {
			return err
		}
	}

	return nil
}

// SendDigest formats and sends the daily digest entity using safe HTML chunking.
func (t *telegramNotifier) SendDigest(ctx context.Context, digest *entity.Digest) error {
	if t.botToken == "" || t.chatID == "" {
		log.Printf("[Telegram:Console] Target Chat: %s\n%s", t.chatID, digest.FormatHTML())
		return nil
	}

	chunks := digest.FormatHTMLChunks()
	for _, chunk := range chunks {
		if err := t.sendChunkWithFallback(ctx, chunk); err != nil {
			return err
		}
	}
	return nil
}

func (t *telegramNotifier) sendChunkWithFallback(ctx context.Context, text string) error {
	err := t.sendChunk(ctx, text, "HTML")
	if err != nil && (strings.Contains(err.Error(), "can't parse entities") || strings.Contains(err.Error(), "Bad Request")) {
		log.Printf("[Telegram] Warning: HTML parse error (%v), retrying chunk as plain text...", err)
		plainText := stripHTMLTags(text)
		return t.sendChunk(ctx, plainText, "")
	}
	return err
}

func (t *telegramNotifier) sendChunk(ctx context.Context, text, parseMode string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	reqBody := sendMessageRequest{
		ChatID:                t.chatID,
		Text:                  text,
		ParseMode:             parseMode,
		DisableWebPagePreview: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram API request error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var apiResp apiResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return fmt.Errorf("unexpected response from Telegram: %s", string(bodyBytes))
	}

	if !apiResp.OK {
		return fmt.Errorf("telegram API returned error: %s", apiResp.Description)
	}

	return nil
}

var tagRegex = regexp.MustCompile(`<[^>]*>`)

func stripHTMLTags(s string) string {
	r := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", "\"",
	)
	clean := tagRegex.ReplaceAllString(s, "")
	return r.Replace(clean)
}

func splitMessage(msg string, chunkSize int) []string {
	if len(msg) <= chunkSize {
		return []string{msg}
	}

	var chunks []string
	runes := []rune(msg)
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}
