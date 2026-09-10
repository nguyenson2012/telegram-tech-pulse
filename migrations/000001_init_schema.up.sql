-- =============================================================================
-- Migration: 000001_init_schema.up.sql
-- Description: AI Tech Pulse Digest - Schema Initialization with pgvector
-- =============================================================================

-- 1. Enable required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. Create feeds table
CREATE TABLE IF NOT EXISTS feeds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    url VARCHAR(2048) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feeds_is_active ON feeds(is_active);

-- 3. Create articles table
CREATE TABLE IF NOT EXISTS articles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    title VARCHAR(1024) NOT NULL,
    url VARCHAR(2048) NOT NULL UNIQUE,
    raw_content TEXT NOT NULL,
    summary TEXT NOT NULL,
    quality_score INTEGER NOT NULL CHECK (quality_score BETWEEN 1 AND 10),
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_articles_url ON articles(url);
CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);
CREATE INDEX IF NOT EXISTS idx_articles_created_score ON articles(created_at DESC, quality_score DESC);

-- 4. Create article_embeddings table (768 dimensions for Gemini text-embedding-004)
CREATE TABLE IF NOT EXISTS article_embeddings (
    article_id UUID PRIMARY KEY REFERENCES articles(id) ON DELETE CASCADE,
    embedding vector(768) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 5. Create HNSW Cosine Index for ultra-fast vector similarity search
CREATE INDEX IF NOT EXISTS idx_article_embeddings_cosine 
ON article_embeddings USING hnsw (embedding vector_cosine_ops);
