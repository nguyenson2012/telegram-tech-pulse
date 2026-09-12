package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-tech-pulse/digest/internal/adapter/ai/gemini"
	queueMemory "github.com/ai-tech-pulse/digest/internal/adapter/queue/memory"
	"github.com/ai-tech-pulse/digest/internal/adapter/repository/postgres"
	"github.com/ai-tech-pulse/digest/internal/adapter/rss"
	"github.com/ai-tech-pulse/digest/internal/adapter/telegram"
	"github.com/ai-tech-pulse/digest/internal/config"
	cronDelivery "github.com/ai-tech-pulse/digest/internal/delivery/cron"
	httpDelivery "github.com/ai-tech-pulse/digest/internal/delivery/http"
	"github.com/ai-tech-pulse/digest/internal/usecase"
)

func main() {
	log.Println("==================================================")
	log.Println("🚀 AI Tech Pulse Digest — Clean Architecture MVP")
	log.Println("==================================================")

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Fatal: failed to load configuration: %v", err)
	}

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 2. Database Connection (PostgreSQL + pgvector)
	log.Printf("[Init] Connecting to PostgreSQL at %s...", redactDSN(cfg.DatabaseURL))
	db, err := postgres.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Printf("[Init] Warning: Database connection failed (%v). Retrying in background...", err)
		// For MVP testing without active DB, continue gracefully or handle retry
	} else {
		log.Println("[Init] PostgreSQL connected with pgvector support enabled.")
	}

	// 3. Initialize Adapters (Infrastructure Layer)
	feedRepo := postgres.NewFeedRepository(db)
	vectorRepo := postgres.NewVectorRepository(db)

	aiService, err := gemini.NewGeminiService(rootCtx, cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiEmbeddingModel)
	if err != nil {
		log.Fatalf("Fatal: failed to initialize Gemini service: %v", err)
	}

	notifier := telegram.NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
	rssParser := rss.NewRSSParser()
	jobQueue := queueMemory.NewChannelJobQueue(cfg.WorkerCount, cfg.QueueBufferSize)

	// 4. Initialize Use Cases (Application Layer)
	feedUseCase := usecase.NewFeedUseCase(feedRepo, rssParser)
	searchUseCase := usecase.NewSearchUseCase(vectorRepo, aiService)
	pipelineUseCase := usecase.NewPipelineUseCase(
		feedRepo,
		vectorRepo,
		aiService,
		notifier,
		rssParser,
		jobQueue,
		cfg.TopArticlesLimit,
	)

	// 5. Start Background Job Queue & Register Handlers
	pipelineUseCase.RegisterQueueHandlers()
	if err := jobQueue.Start(rootCtx); err != nil {
		log.Fatalf("Fatal: failed to start JobQueue: %v", err)
	}

	// 6. Initialize Cron Scheduler
	cronScheduler := cronDelivery.NewScheduler(cfg.CronSchedule, pipelineUseCase)
	if err := cronScheduler.Start(rootCtx); err != nil {
		log.Printf("[Init] Warning: could not start cron scheduler: %v", err)
	}

	// Run pipeline on startup if enabled via env
	if cfg.RunPipelineOnStartup {
		log.Println("[Init] RUN_PIPELINE_ON_STARTUP is enabled. Enqueuing pipeline run...")
		_ = pipelineUseCase.TriggerPipelineAsync(rootCtx)
	}

	// 7. Initialize HTTP Delivery Layer
	feedHandler := httpDelivery.NewFeedHandler(feedUseCase)
	searchHandler := httpDelivery.NewSearchHandler(searchUseCase)
	pipelineHandler := httpDelivery.NewPipelineHandler(pipelineUseCase)

	router := httpDelivery.NewRouter(httpDelivery.RouterConfig{
		DB:              db,
		FeedHandler:     feedHandler,
		SearchHandler:   searchHandler,
		PipelineHandler: pipelineHandler,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. Start HTTP Server in background
	go func() {
		log.Printf("🌐 HTTP Server listening on port :%s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Fatal: HTTP server failed: %v", err)
		}
	}()

	// 9. Graceful Shutdown listener
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	sig := <-shutdownSignal
	log.Printf("\n🛑 Received shutdown signal (%s). Initiating graceful shutdown...", sig.String())

	// Shutdown context with 30s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// A. Stop HTTP Server
	log.Println("[Shutdown] Stopping HTTP server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Shutdown] HTTP server forced shutdown error: %v", err)
	}

	// B. Stop Cron
	log.Println("[Shutdown] Stopping Cron scheduler...")
	cronScheduler.Stop()

	// C. Stop Job Queue (drain workers)
	log.Println("[Shutdown] Draining Job Queue...")
	if err := jobQueue.Stop(shutdownCtx); err != nil {
		log.Printf("[Shutdown] Job Queue drain error: %v", err)
	}

	// D. Close Database connection pool
	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			log.Println("[Shutdown] Closing database connection pool...")
			_ = sqlDB.Close()
		}
	}

	rootCancel()
	log.Println("✅ Shutdown complete. Goodbye!")
}

func redactDSN(dsn string) string {
	if len(dsn) < 15 {
		return "***"
	}
	return dsn[:15] + "...[redacted]"
}
