package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
	"github.com/ai-tech-pulse/digest/internal/domain/repository"
	"github.com/ai-tech-pulse/digest/internal/domain/service"
)

// PipelineUseCase coordinates RSS fetching, AI processing, storage, and Telegram delivery.
type PipelineUseCase struct {
	feedRepo     repository.FeedRepository
	vectorRepo   repository.VectorRepository
	aiService    service.AIService
	notifier     service.NotificationService
	rssParser    service.RSSParser
	jobQueue     service.JobQueue
	topLimit     int
}

// NewPipelineUseCase creates a new PipelineUseCase instance.
func NewPipelineUseCase(
	feedRepo repository.FeedRepository,
	vectorRepo repository.VectorRepository,
	aiService service.AIService,
	notifier service.NotificationService,
	rssParser service.RSSParser,
	jobQueue service.JobQueue,
	topLimit int,
) *PipelineUseCase {
	if topLimit <= 0 {
		topLimit = 5
	}
	return &PipelineUseCase{
		feedRepo:   feedRepo,
		vectorRepo: vectorRepo,
		aiService:  aiService,
		notifier:   notifier,
		rssParser:  rssParser,
		jobQueue:   jobQueue,
		topLimit:   topLimit,
	}
}

// RegisterQueueHandlers attaches background workers to handle queued pipeline jobs.
func (uc *PipelineUseCase) RegisterQueueHandlers() {
	if uc.jobQueue == nil {
		return
	}

	uc.jobQueue.RegisterHandler(service.JobTypeDailyPipeline, func(ctx context.Context, job *service.Job) error {
		log.Printf("[Pipeline] Starting background daily pipeline execution (Job ID: %s)", job.ID)
		digest, err := uc.ExecuteDailyPipeline(ctx)
		if err != nil {
			log.Printf("[Pipeline] Daily pipeline job %s failed: %v", job.ID, err)
			return err
		}
		log.Printf("[Pipeline] Daily pipeline job %s completed successfully (%d top articles)", job.ID, len(digest.Articles))
		return nil
	})
}

// TriggerPipelineAsync enqueues the daily pipeline job into the JobQueue.
func (uc *PipelineUseCase) TriggerPipelineAsync(ctx context.Context) error {
	if uc.jobQueue == nil {
		// Fallback to synchronous if no queue configured
		_, err := uc.ExecuteDailyPipeline(ctx)
		return err
	}

	job := &service.Job{
		ID:        fmt.Sprintf("pipeline-%d", time.Now().UnixNano()),
		Type:      service.JobTypeDailyPipeline,
		Payload:   []byte(`{}`),
		CreatedAt: time.Now().UTC(),
	}

	return uc.jobQueue.Enqueue(ctx, job)
}

// ExecuteDailyPipeline runs the complete end-to-end ingestion, AI enrichment, and digest delivery.
func (uc *PipelineUseCase) ExecuteDailyPipeline(ctx context.Context) (*entity.Digest, error) {
	log.Println("[Pipeline] Fetching active RSS feeds...")
	feeds, err := uc.feedRepo.GetActiveFeeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active feeds: %w", err)
	}

	if len(feeds) == 0 {
		log.Println("[Pipeline] No active feeds found. Pipeline execution skipped.")
		return entity.NewDigest([]*entity.Article{}), nil
	}

	var processedCount int
	var newArticlesCount int

	for _, feed := range feeds {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		log.Printf("[Pipeline] Parsing feed: %s (%s)", feed.Name, feed.URL)
		items, err := uc.rssParser.ParseURL(ctx, feed.URL)
		if err != nil {
			log.Printf("[Pipeline] Error parsing feed %s: %v. Continuing...", feed.URL, err)
			continue
		}

		for _, item := range items {
			processedCount++

			// 1. Deduplication check
			exists, err := uc.feedRepo.ArticleExistsByURL(ctx, item.URL)
			if err != nil {
				log.Printf("[Pipeline] Error checking article URL %s: %v", item.URL, err)
				continue
			}
			if exists {
				continue
			}

			// 2. Process new article with AI
			log.Printf("[Pipeline] Processing new article: %s", item.Title)
			if err := uc.processArticle(ctx, feed.ID, item); err != nil {
				log.Printf("[Pipeline] Error processing article '%s': %v", item.Title, err)
				continue
			}
			newArticlesCount++
		}
	}

	log.Printf("[Pipeline] Ingestion completed. Checked %d items, processed %d new articles.", processedCount, newArticlesCount)

	// 3. Select Top N highest-scoring articles of the day
	topArticles, err := uc.feedRepo.GetTopArticlesToday(ctx, uc.topLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve top articles: %w", err)
	}

	digest := entity.NewDigest(topArticles)

	// 4. Send Daily Digest via Telegram NotificationService
	if uc.notifier != nil && len(topArticles) > 0 {
		log.Printf("[Pipeline] Delivering daily digest (%d articles) to Telegram...", len(topArticles))
		if err := uc.notifier.SendDigest(ctx, digest); err != nil {
			log.Printf("[Pipeline] Warning: failed to send Telegram digest: %v", err)
			// Return digest anyway so caller gets the result
		} else {
			log.Println("[Pipeline] Telegram digest sent successfully!")
		}
	}

	return digest, nil
}

// processArticle enriches a single article via Gemini and persists Article + Vector Embedding.
func (uc *PipelineUseCase) processArticle(ctx context.Context, feedID string, item *service.RSSItem) error {
	// A. AI Summarization & Scoring (3 lines + 2 takeaways + score 1-10)
	aiRes, err := uc.aiService.SummarizeAndScore(ctx, item.Title, item.Content)
	if err != nil {
		return fmt.Errorf("AI summarization failed: %w", err)
	}

	summaryText := aiRes.FormattedSummary()

	// B. Create domain Article entity
	article, err := entity.NewArticle(
		feedID,
		item.Title,
		item.URL,
		item.Content,
		summaryText,
		aiRes.QualityScore,
		item.PublishedAt,
	)
	if err != nil {
		return fmt.Errorf("invalid article entity: %w", err)
	}

	// C. Persist Article into Database
	if err := uc.feedRepo.CreateArticle(ctx, article); err != nil {
		return fmt.Errorf("failed to store article: %w", err)
	}

	// D. Generate 768-dim Vector Embedding for the summary
	embedding, err := uc.aiService.GenerateEmbedding(ctx, summaryText)
	if err != nil {
		log.Printf("[Pipeline] Warning: failed to generate embedding for article %s: %v", article.ID, err)
		return nil // Non-fatal: article is saved, embedding can be backfilled
	}

	// E. Persist Vector Embedding
	if err := uc.vectorRepo.SaveEmbedding(ctx, article.ID, embedding); err != nil {
		log.Printf("[Pipeline] Warning: failed to save vector embedding for article %s: %v", article.ID, err)
	}

	return nil
}
