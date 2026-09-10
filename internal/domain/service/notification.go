package service

import (
	"context"

	"github.com/ai-tech-pulse/digest/internal/domain/entity"
)

// NotificationService defines the port for broadcasting alerts and digests (e.g. Telegram).
type NotificationService interface {
	// Send delivers a raw formatted message string.
	Send(ctx context.Context, message string) error

	// SendDigest delivers a structured Digest entity.
	SendDigest(ctx context.Context, digest *entity.Digest) error
}
