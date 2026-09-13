package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/usecase"
)

// Listener manages incoming Telegram commands via long polling.
type Listener struct {
	botToken        string
	baseURL         string
	allowedChatID   string
	feedUseCase     *usecase.FeedUseCase
	pipelineUseCase *usecase.PipelineUseCase
	httpClient      *http.Client
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// NewListener creates a new Telegram command Listener.
func NewListener(
	botToken string,
	allowedChatID string,
	feedUseCase *usecase.FeedUseCase,
	pipelineUseCase *usecase.PipelineUseCase,
) *Listener {
	return &Listener{
		botToken:        botToken,
		baseURL:         "https://api.telegram.org",
		allowedChatID:   strings.TrimSpace(allowedChatID),
		feedUseCase:     feedUseCase,
		pipelineUseCase: pipelineUseCase,
		httpClient: &http.Client{
			Timeout: 40 * time.Second, // Long-polling timeout + buffer
		},
		stopChan: make(chan struct{}),
	}
}

// Start launches the polling loop in a background goroutine.
func (l *Listener) Start(ctx context.Context) {
	if l.botToken == "" {
		log.Println("[Telegram:Listener] Bot token is empty. Listener not started.")
		return
	}

	l.wg.Add(1)
	go l.pollLoop(ctx)
}

// Stop terminates the polling loop gracefully.
func (l *Listener) Stop() {
	select {
	case <-l.stopChan:
		return // already closed
	default:
		close(l.stopChan)
	}
	l.wg.Wait()
}

type tgUpdate struct {
	UpdateID int        `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	MessageID int     `json:"message_id"`
	Chat      *tgChat `json:"chat"`
	Text      string  `json:"text"`
	Date      int64   `json:"date"`
}

type tgChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type tgGetUpdatesResponse struct {
	OK          bool        `json:"ok"`
	Result      []*tgUpdate `json:"result"`
	Description string      `json:"description,omitempty"`
}

type tgSendMessageRequest struct {
	ChatID                int64  `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

func (l *Listener) pollLoop(ctx context.Context) {
	defer l.wg.Done()
	log.Println("[Telegram:Listener] Long polling started...")

	var lastUpdateID int

	for {
		select {
		case <-ctx.Done():
			log.Println("[Telegram:Listener] Context cancelled, stopping poll loop.")
			return
		case <-l.stopChan:
			log.Println("[Telegram:Listener] Stop signal received, terminating.")
			return
		default:
		}

		updates, err := l.fetchUpdates(ctx, lastUpdateID+1)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// Transient network error, wait a bit before retrying
			select {
			case <-time.After(3 * time.Second):
			case <-l.stopChan:
				return
			case <-ctx.Done():
				return
			}
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= lastUpdateID {
				lastUpdateID = update.UpdateID
			}

			if update.Message != nil && update.Message.Text != "" {
				l.handleMessage(ctx, update.Message)
			}
		}
	}
}

func (l *Listener) fetchUpdates(ctx context.Context, offset int) ([]*tgUpdate, error) {
	url := fmt.Sprintf("%s/bot%s/getUpdates?offset=%d&timeout=25", l.baseURL, l.botToken, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res tgGetUpdatesResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	if !res.OK {
		return nil, fmt.Errorf("telegram API error: %s", res.Description)
	}

	return res.Result, nil
}

func (l *Listener) handleMessage(ctx context.Context, msg *tgMessage) {
	chatIDStr := fmt.Sprintf("%d", msg.Chat.ID)

	// Authorization check
	if l.allowedChatID != "" && chatIDStr != l.allowedChatID {
		log.Printf("[Telegram:Listener] Unauthorized command attempt from Chat ID: %s", chatIDStr)
		_ = l.reply(ctx, msg.Chat.ID, "⛔ <b>Từ chối truy cập:</b> Bạn không có quyền quản trị bot này.")
		return
	}

	text := strings.TrimSpace(msg.Text)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	// Handle commands with @botusername suffix (e.g. /feeds@MyBot)
	if idx := strings.Index(cmd, "@"); idx != -1 {
		cmd = cmd[:idx]
	}

	switch strings.ToLower(cmd) {
	case "/start", "/help":
		l.handleHelp(ctx, msg.Chat.ID)
	case "/feeds", "/listfeeds":
		l.handleListFeeds(ctx, msg.Chat.ID)
	case "/addfeed":
		l.handleAddFeed(ctx, msg.Chat.ID, parts[1:])
	case "/toggle", "/togglefeed":
		l.handleToggleFeed(ctx, msg.Chat.ID, parts[1:])
	case "/delfeed", "/deletefeed", "/rmfeed":
		l.handleDeleteFeed(ctx, msg.Chat.ID, parts[1:])
	case "/run", "/pipeline":
		l.handleRunPipeline(ctx, msg.Chat.ID)
	default:
		// Do not spam on unrecognized plain text messages
		if strings.HasPrefix(cmd, "/") {
			_ = l.reply(ctx, msg.Chat.ID, fmt.Sprintf("❓ Lệnh không xác định: <code>%s</code>\nGõ <code>/help</code> để xem hướng dẫn.", cmd))
		}
	}
}

func (l *Listener) handleHelp(ctx context.Context, chatID int64) {
	helpText := `🤖 <b>AI Tech Pulse — Trình Quản Lý Nguồn Tin RSS</b>

Các lệnh khả dụng:
📋 <code>/feeds</code> — Xem danh sách các nguồn RSS đang theo dõi
➕ <code>/addfeed &lt;url&gt; [tên]</code> — Thêm một nguồn RSS mới
🔄 <code>/toggle &lt;id&gt;</code> — Bật / Tắt trạng thái một nguồn RSS
🗑️ <code>/delfeed &lt;id&gt;</code> — Xóa hoàn toàn nguồn RSS
🚀 <code>/run</code> — Kích hoạt chạy pipeline tổng hợp tin ngay lập tức
❓ <code>/help</code> — Hiển thị hướng dẫn này

<i>Ví dụ thêm nguồn:</i>
<code>/addfeed https://vnexpress.net/rss/khoa-hoc-cong-nghe.rss VnExpress</code>`

	_ = l.reply(ctx, chatID, helpText)
}

func (l *Listener) handleListFeeds(ctx context.Context, chatID int64) {
	feeds, err := l.feedUseCase.ListFeeds(ctx)
	if err != nil {
		_ = l.reply(ctx, chatID, fmt.Sprintf("❌ Lỗi khi lấy danh sách feed: %v", err))
		return
	}

	if len(feeds) == 0 {
		_ = l.reply(ctx, chatID, "ℹ️ Hiện chưa có nguồn RSS nào trong hệ thống.\nDùng <code>/addfeed &lt;url&gt; [tên]</code> để thêm nguồn mới.")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>Danh sách nguồn tin RSS (%d nguồn):</b>\n\n", len(feeds)))

	for i, f := range feeds {
		statusIcon := "🟢"
		statusText := "Hoạt động"
		if !f.IsActive {
			statusIcon = "🔴"
			statusText = "Đang tắt"
		}

		sb.WriteString(fmt.Sprintf(
			"%d. %s <b>%s</b> (%s)\n   🔗 <i>%s</i>\n   🆔 <code>%s</code>\n\n",
			i+1,
			statusIcon,
			escapeHTML(f.Name),
			statusText,
			f.URL,
			f.ID,
		))
	}

	sb.WriteString("👉 <i>Gợi ý:</i>\n• Dùng <code>/toggle &lt;id&gt;</code> để Bật/Tắt\n• Dùng <code>/delfeed &lt;id&gt;</code> để Xóa")

	_ = l.reply(ctx, chatID, sb.String())
}

func (l *Listener) handleAddFeed(ctx context.Context, chatID int64, args []string) {
	if len(args) < 1 {
		_ = l.reply(ctx, chatID, "⚠️ <b>Thiếu thông tin!</b>\nCú pháp: <code>/addfeed &lt;url&gt; [tên nguồn]</code>\n\n<i>Ví dụ:</i>\n<code>/addfeed https://vnexpress.net/rss/khoa-hoc-cong-nghe.rss VnExpress</code>")
		return
	}

	rawURL := args[0]
	name := ""
	if len(args) > 1 {
		name = strings.Join(args[1:], " ")
	}

	_ = l.reply(ctx, chatID, "⏳ Đang kiểm tra và thêm nguồn RSS...")

	feed, err := l.feedUseCase.AddFeed(ctx, usecase.AddFeedRequest{
		URL:  rawURL,
		Name: name,
	})
	if err != nil {
		_ = l.reply(ctx, chatID, fmt.Sprintf("❌ <b>Không thể thêm nguồn:</b> %v", err))
		return
	}

	msg := fmt.Sprintf(
		"✅ <b>Đã thêm nguồn RSS thành công!</b>\n\n📌 <b>%s</b>\n🔗 URL: %s\n🆔 ID: <code>%s</code>\nTrạng thái: 🟢 Hoạt động",
		escapeHTML(feed.Name),
		feed.URL,
		feed.ID,
	)
	_ = l.reply(ctx, chatID, msg)
}

func (l *Listener) handleToggleFeed(ctx context.Context, chatID int64, args []string) {
	if len(args) < 1 {
		_ = l.reply(ctx, chatID, "⚠️ <b>Thiếu ID nguồn!</b>\nCú pháp: <code>/toggle &lt;id&gt;</code>\n(Dùng <code>/feeds</code> để lấy ID)")
		return
	}

	targetID := strings.TrimSpace(args[0])

	// Find the feed to inspect current state
	feeds, err := l.feedUseCase.ListFeeds(ctx)
	if err != nil {
		_ = l.reply(ctx, chatID, fmt.Sprintf("❌ Lỗi: %v", err))
		return
	}

	var targetFeed *entity.Feed
	for _, f := range feeds {
		if f.ID == targetID {
			targetFeed = f
			break
		}
	}

	if targetFeed == nil {
		_ = l.reply(ctx, chatID, fmt.Sprintf("❌ Không tìm thấy nguồn RSS với ID: <code>%s</code>", targetID))
		return
	}

	newStatus := !targetFeed.IsActive
	if err := l.feedUseCase.ToggleFeedStatus(ctx, targetFeed.ID, newStatus); err != nil {
		_ = l.reply(ctx, chatID, fmt.Sprintf("❌ Lỗi khi đổi trạng thái: %v", err))
		return
	}

	statusIcon := "🟢"
	statusAction := "BẬT"
	if !newStatus {
		statusIcon = "🔴"
		statusAction = "TẮT"
	}

	msg := fmt.Sprintf(
		"%s Đã <b>%s</b> thu thập tin từ nguồn: <b>%s</b>",
		statusIcon,
		statusAction,
		escapeHTML(targetFeed.Name),
	)
	_ = l.reply(ctx, chatID, msg)
}

func (l *Listener) handleDeleteFeed(ctx context.Context, chatID int64, args []string) {
	if len(args) < 1 {
		_ = l.reply(ctx, chatID, "⚠️ <b>Thiếu ID nguồn!</b>\nCú pháp: <code>/delfeed &lt;id&gt;</code>\n(Dùng <code>/feeds</code> để lấy ID)")
		return
	}

	targetID := strings.TrimSpace(args[0])

	if err := l.feedUseCase.DeleteFeed(ctx, targetID); err != nil {
		_ = l.reply(ctx, chatID, fmt.Sprintf("❌ Lỗi khi xóa nguồn RSS: %v", err))
		return
	}

	_ = l.reply(ctx, chatID, fmt.Sprintf("🗑️ <b>Đã xóa thành công nguồn RSS</b> (ID: <code>%s</code>)", targetID))
}

func (l *Listener) handleRunPipeline(ctx context.Context, chatID int64) {
	if l.pipelineUseCase == nil {
		_ = l.reply(ctx, chatID, "⚠️ Pipeline use case chưa được khởi tạo.")
		return
	}

	if err := l.pipelineUseCase.TriggerPipelineAsync(ctx); err != nil {
		_ = l.reply(ctx, chatID, fmt.Sprintf("❌ Không thể khởi chạy pipeline: %v", err))
		return
	}

	_ = l.reply(ctx, chatID, "🚀 <b>Đã kích hoạt pipeline tổng hợp tin!</b>\nTiến trình đang cào tin, phân tích bằng Gemini và sẽ gửi bản tin tới bạn ngay khi hoàn tất.")
}

func (l *Listener) reply(ctx context.Context, chatID int64, text string) error {
	apiURL := fmt.Sprintf("%s/bot%s/sendMessage", l.baseURL, l.botToken)

	reqBody := tgSendMessageRequest{
		ChatID:                chatID,
		Text:                  text,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return r.Replace(s)
}
