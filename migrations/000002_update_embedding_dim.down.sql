-- Rollback: Restore embedding dimensions to 768 (text-embedding-004)

DROP INDEX IF EXISTS article_embeddings_embedding_hnsw_idx;
DROP TABLE IF EXISTS article_embeddings;

CREATE TABLE article_embeddings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id  UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    embedding   vector(768) NOT NULL,
    model       VARCHAR(100) NOT NULL DEFAULT 'text-embedding-004',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (article_id)
);

CREATE INDEX article_embeddings_embedding_hnsw_idx
    ON article_embeddings
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
