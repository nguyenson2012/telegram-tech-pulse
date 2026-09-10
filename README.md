# 🚀 AI Tech Pulse Digest — Clean Architecture MVP

An automated, production-ready system that aggregates tech articles from RSS feeds, summarizes and quality-scores them using Google Gemini (`gemini-3.5-flash` / `gemini-3.6-flash`), generates 768-dimensional vector embeddings (`gemini-embedding-2` via Matryoshka MRL reduction), stores them in PostgreSQL with `pgvector`, and broadcasts daily digests via Telegram Bot.

Built following **Hexagonal / Clean Architecture** principles in **Go 1.24+**, with high maintainability and a clean path to Event-Driven / Microservices.

---

## 🏛️ Architecture Overview

```
                                  +----------------------------+
                                  |     Delivery / Ports       |
                                  |   - REST API (Chi)         |
                                  |   - Cron Scheduler (robfig)|
                                  +--------------+-------------+
                                                 |
                                                 v
                                  +----------------------------+
                                  |     Application Layer      |
                                  |   - FeedUseCase            |
                                  |   - PipelineUseCase        |
                                  |   - SearchUseCase          |
                                  +--------------+-------------+
                                                 |
                                                 v
                                  +----------------------------+
                                  |     Domain Core (Zero Ext) |
                                  |   - Entities (Article, Feed)|
                                  |   - Ports (Interfaces)     |
                                  +--------------+-------------+
                                                 ^
                                                 |
                 +---------------+---------------+---------------+---------------+
                 |                               |                               |
                 v                               v                               v
    +-------------------------+     +-------------------------+     +-------------------------+
    |       PostgreSQL        |     |      Google Gemini      |     |      Telegram Bot       |
    |   pgvector (768-dim)    |     |  gemini-3.5/3.6-flash   |     |   HTML formatted digest |
    |   HNSW Cosine Index     |     |   gemini-embedding-2    |     |   Auto-chunking         |
    +-------------------------+     +-------------------------+     +-------------------------+
                 |                               |
                 v                               v
    +-------------------------+     +-------------------------+
    |   GoFeed (RSS/Atom)     |     |   In-Memory JobQueue    |
    |   HTML tag sanitizer    |     |  Go channels & workers  |
    +-------------------------+     +-------------------------+
```

### Architectural Boundaries
1. **Domain Layer (`internal/domain/`)**: Pure business logic, entities, and port definitions. **Zero external dependencies**.
2. **Use Case Layer (`internal/usecase/`)**: Orchestrates business rules (`ExecuteDailyPipeline`, `Search`, `AddFeed`). Interacts solely with Domain Interfaces.
3. **Adapter Layer (`internal/adapter/`)**: Concrete implementations for PostgreSQL (`gorm.io` + `pgvector-go`), Gemini API (`genai`), Telegram Bot (`net/http`), RSS Parser (`gofeed`), and JobQueue (`memory`).
4. **Delivery Layer (`internal/delivery/`)**: HTTP endpoints (Chi router) and Cron task runner (`robfig/cron/v3`).

---

## 📁 Directory Tree

```
ai-tech-pulse/
├── cmd/
│   └── server/
│       └── main.go                 # Dependency injection & graceful shutdown entrypoint
├── internal/
│   ├── config/
│   │   └── config.go               # Environment variables configuration loader
│   ├── domain/                     # CORE DOMAIN LAYER (Zero external dependencies)
│   │   ├── entity/
│   │   │   ├── article.go          # Article & ArticleEmbedding entities
│   │   │   ├── feed.go             # Feed entity with URL validation
│   │   │   ├── digest.go           # Digest value object with HTML/Markdown rendering
│   │   │   ├── feed_test.go        # Unit tests for Feed entity
│   │   │   └── digest_test.go      # Unit tests for Digest entity & HTML escaping
│   │   ├── repository/
│   │   │   ├── feed_repository.go  # Port: Feed & Article persistence
│   │   │   └── vector_repository.go# Port: Vector similarity search
│   │   └── service/
│   │       ├── ai_service.go       # Port: Gemini summarization & embeddings
│   │       ├── notification.go     # Port: Notification broadcasting
│   │       ├── job_queue.go        # Port: Background Job Queue abstraction
│   │       └── rss_parser.go       # Port: RSS/Atom parser
│   ├── usecase/                    # USE CASE / APPLICATION LAYER
│   │   ├── feed_usecase.go         # Add, list, toggle feeds
│   │   ├── feed_usecase_test.go    # Unit tests for FeedUseCase
│   │   ├── pipeline_usecase.go     # End-to-end ingestion, AI enrichment & digest
│   │   ├── search_usecase.go       # Semantic search using cosine similarity
│   │   └── search_usecase_test.go  # Unit tests for SearchUseCase
│   ├── adapter/                    # ADAPTER / INFRASTRUCTURE LAYER
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── client.go       # GORM connection & pgvector extension check
│   │   │       ├── feed_repo.go    # PostgreSQL FeedRepository implementation
│   │   │       └── vector_repo.go  # PostgreSQL pgvector VectorRepository (<=> cosine)
│   │   ├── ai/
│   │   │   └── gemini/
│   │   │       ├── client.go       # Official Gemini SDK: 3.5/3.6 Flash & gemini-embedding-2 (MRL 768)
│   │   │       └── client_test.go  # Unit tests for Gemini client & MRL reduction
│   │   ├── telegram/
│   │   │   └── bot.go              # Telegram Bot API notification client
│   │   ├── rss/
│   │   │   └── parser.go           # GoFeed RSS/Atom parser with HTML stripping
│   │   └── queue/
│   │       └── memory/
│   │           ├── channel_queue.go     # Concurrency-safe channel worker pool
│   │           └── channel_queue_test.go# Unit tests for worker pool & graceful drain
│   └── delivery/                   # DELIVERY / ENTRYPOINT LAYER
│       ├── http/
│       │   ├── router.go           # Chi router, middlewares, CORS, health check
│       │   ├── response.go         # Standardized JSON response helpers
│       │   ├── feed_handler.go     # Feed endpoints
│       │   ├── search_handler.go   # Semantic search endpoint
│       │   └── pipeline_handler.go # Pipeline manual trigger endpoint
│       └── cron/
│           └── scheduler.go        # Cron runner for scheduled daily digest
├── migrations/
│   ├── 000001_init_schema.up.sql   # PostgreSQL + pgvector schema & HNSW index (768 dims)
│   ├── 000001_init_schema.down.sql # Drop tables
│   ├── 000002_update_embedding_dim.up.sql   # Optional 3072-dim migration with halfvec HNSW index
│   └── 000002_update_embedding_dim.down.sql # Rollback to 768 dims
├── .env.example                    # Sample environment variables
├── Dockerfile                      # Multi-stage production container build (<35MB)
├── docker-compose.yml              # Local dev stack (App + PostgreSQL with pgvector)
├── Makefile                        # Dev commands
├── go.mod
├── go.sum
└── README.md
```

---

## ⚙️ Environment Variables (`.env.example`)

Copy `.env.example` to `.env` and fill in your credentials:

```bash
cp .env.example .env
```

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | HTTP Server Port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/aitechpulse?sslmode=disable` |
| `GEMINI_API_KEY` | Google Gemini API Key | *(Required for live AI calls)* |
| `GEMINI_MODEL` | Gemini text model | `gemini-3.6-flash` (auto-fallbacks to `gemini-3.5-flash` on 429) |
| `GEMINI_EMBEDDING_MODEL` | Embedding model (768 dimensions via MRL) | `gemini-embedding-2` |
| `TELEGRAM_BOT_TOKEN` | Telegram Bot Token from @BotFather | *(Optional for dev, logs to console)* |
| `TELEGRAM_CHAT_ID` | Telegram Chat ID from @userinfobot | *(Optional for dev)* |
| `CRON_SCHEDULE` | Cron expression for daily pipeline | `0 8 * * *` (8:00 AM UTC daily) |
| `RUN_PIPELINE_ON_STARTUP`| Trigger pipeline run immediately on start | `false` |
| `WORKER_COUNT` | Number of background worker goroutines | `3` |
| `QUEUE_BUFFER_SIZE` | In-memory job queue channel buffer | `100` |
| `TOP_ARTICLES_LIMIT` | Number of top articles in daily digest | `5` |

---

## 🗄️ Database Schema & Vector Indexing

The migration script [`migrations/000001_init_schema.up.sql`](migrations/000001_init_schema.up.sql) sets up:

1. **`feeds` table**: Tracks RSS source URLs, names, and active status.
2. **`articles` table**: Ingested articles with `raw_content`, AI `summary`, `quality_score` (1–10), and timestamps.
3. **`article_embeddings` table**: Stores `vector(768)` embeddings for each article summary.
   - `gemini-embedding-2` natively generates 3072-dimensional embeddings. The adapter applies **Matryoshka Representation Learning (MRL)** to truncate the vector to the first 768 dimensions and L2-normalizes it. This maintains high semantic fidelity while remaining fully within pgvector's HNSW index limitation (max 2000 dimensions for standard float vectors).
4. **HNSW Cosine Index**:
   ```sql
   CREATE INDEX idx_article_embeddings_cosine 
   ON article_embeddings USING hnsw (embedding vector_cosine_ops);
   ```

---

## 🚀 Quick Start

### 1. Run with Docker Compose (Recommended)

Spins up PostgreSQL with `pgvector` pre-configured and the application in isolated containers:

```bash
docker compose up -d --build
```

View logs:
```bash
docker compose logs -f app
```

### 2. Run Locally

Ensure PostgreSQL with `pgvector` is running on `localhost:5432`:

```bash
# 1. Download dependencies
make tidy

# 2. Run automated test suite
make test

# 3. Compile binary
make build

# 4. Run application
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/aitechpulse?sslmode=disable"
export GEMINI_API_KEY="your-gemini-key"
export TELEGRAM_BOT_TOKEN="your-bot-token"
export TELEGRAM_CHAT_ID="your-chat-id"
./bin/server
```

---

## 📡 REST API Documentation

### 1. Health Check
```http
GET /health
```
**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "database": "connected",
    "status": "ok",
    "timestamp": "2026-09-09T14:15:00Z"
  }
}
```

---

### 2. Add an RSS Feed
```http
POST /api/v1/feeds
Content-Type: application/json

{
  "url": "https://techcrunch.com/category/artificial-intelligence/feed/",
  "name": "TechCrunch AI"
}
```
*(Note: If `name` is omitted, the title will be automatically discovered from the RSS feed metadata).*

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": "e8d910a2-a9b0-4ecb-99d8-8fcda9505c21",
    "url": "https://techcrunch.com/category/artificial-intelligence/feed/",
    "name": "TechCrunch AI",
    "is_active": true,
    "created_at": "2026-09-09T14:15:00Z",
    "updated_at": "2026-09-09T14:15:00Z"
  }
}
```

---

### 3. List Tracked Feeds
```http
GET /api/v1/feeds
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "e8d910a2-a9b0-4ecb-99d8-8fcda9505c21",
      "url": "https://techcrunch.com/category/artificial-intelligence/feed/",
      "name": "TechCrunch AI",
      "is_active": true,
      "created_at": "2026-09-09T14:15:00Z",
      "updated_at": "2026-09-09T14:15:00Z"
    }
  ]
}
```

---

### 4. Trigger Daily Pipeline On-Demand
Execute or queue the end-to-end ingestion, AI scoring, and Telegram delivery manually:

```http
POST /api/v1/pipeline/run?async=true
```

**Response (202 Accepted):**
```json
{
  "success": true,
  "data": {
    "message": "Daily pipeline job enqueued successfully into JobQueue",
    "status": "queued"
  }
}
```

Or execute synchronously:
```http
POST /api/v1/pipeline/run?async=false
```

---

### 5. Semantic Vector Search
Perform Cosine Similarity search over ingested article embeddings using natural language queries:

```http
GET /api/v1/search?q=open+source+reasoning+models&limit=5
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "count": 1,
    "query": "open source reasoning models",
    "results": [
      {
        "article": {
          "id": "3b1859c2-b34e-4e4b-97e3-0570b20cb3a0",
          "feed_id": "e8d910a2-a9b0-4ecb-99d8-8fcda9505c21",
          "title": "DeepSeek Unveils Next-Gen Reasoning Architecture",
          "url": "https://techcrunch.com/deepseek-reasoning",
          "summary": "1. DeepSeek open-sources new reasoning-focused weights.\n2. Employs reinforcement learning without supervised warm-up.\n3. Matches top proprietary models on competitive benchmarks.\n\n💡 Key Takeaways:\n• Dramatically lowers training cost for mathematical reasoning.\n• Open weights accelerate agentic workflows.",
          "quality_score": 9,
          "published_at": "2026-09-09T10:00:00Z",
          "created_at": "2026-09-09T14:15:00Z"
        },
        "similarity_score": 0.8932
      }
    ]
  }
}
```

---

## 🔄 Transitioning to Event-Driven / Microservices

The codebase was architected from Day 1 to make transition to microservices effortless:

1. **Pluggable Job Queue (`internal/domain/service/job_queue.go`)**:
   - The current MVP uses `internal/adapter/queue/memory` (Go channels + worker pool).
   - To transition to **Redis (Asynq / Redis Streams)**, simply create `internal/adapter/queue/redis` implementing `service.JobQueue`. No use cases or business logic need to change.
2. **Decomposing into Microservices**:
   - **Ingestion Service**: Handles `FeedUseCase` and RSS polling. Publishes `ArticleIngestedEvent` to Kafka / NATS / RabbitMQ.
   - **AI Enrichment Worker**: Consumes `ArticleIngestedEvent`, calls Gemini for structured summary & embeddings, persists to `pgvector`, and emits `ArticleEnrichedEvent`.
   - **Digest & Notification Service**: Consumes top scored articles, builds digests, and dispatches to Telegram, Slack, or Email.
   - **Search Service**: Dedicated read-optimized semantic search API querying `pgvector`.
