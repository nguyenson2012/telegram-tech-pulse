-- =============================================================================
-- Migration: 000001_init_schema.down.sql
-- Description: AI Tech Pulse Digest - Teardown
-- =============================================================================

DROP INDEX IF EXISTS idx_article_embeddings_cosine;
DROP TABLE IF EXISTS article_embeddings;
DROP TABLE IF EXISTS articles;
DROP TABLE IF EXISTS feeds;
