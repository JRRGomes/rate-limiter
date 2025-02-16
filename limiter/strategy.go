package limiter

import "context"

// LimiterStorage defines the interface for persistence mechanisms.
type LimiterStorage interface {
	Increment(ctx context.Context, key string, expirationSeconds int) (int64, error)
	IsBlocked(ctx context.Context, key string) (bool, error)
	Block(ctx context.Context, key string, durationSeconds int) error
	Reset(ctx context.Context, key string) error
}
