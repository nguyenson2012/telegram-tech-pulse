-- Migration: Update embedding dimensions from 768 (text-embedding-004)
-- to 3072 (gemini-embedding-2)
-- This migration drops and recreates the article_embeddings table
-- since pgvector does not support ALTER COLUMN for vector dimensions.

-- Drop existing HNSW index
DROP INDEX IF EXISTS article_embeddings_embedding_hnsw_idx;

-- Drop existing table
DROP TABLE IF EXISTS article_embeddings;

-- Recreate with 3072 dimensions for gemini-embedding-2
CREATE TABLE article_embeddings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id  UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    embedding   vector(3072) NOT NULL,
    model       VARCHAR(100) NOT NULL DEFAULT 'gemini-embedding-2',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (article_id)
);

-- Recreate HNSW index for cosine similarity search
-- Note: Standard vector(3072) exceeds pgvector's HNSW 2000-dim limit, so we cast to halfvec(3072)
CREATE INDEX article_embeddings_embedding_hnsw_idx
    ON article_embeddings
    USING hnsw ((embedding::halfvec(3072)) halfvec_cosine_ops)
    WITH (m = 16, ef_construction = 64);
