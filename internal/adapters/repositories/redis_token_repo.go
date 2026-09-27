package repositories

import (
	"context"
	"kda-auth-service/internal/core/ports"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type redisTokenRepo struct {
	rdb redisCommands
}

type redisCommands interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
}

func NewRedisTokenRepository(rdb *redis.Client) ports.TokenCacheRepository {
	return &redisTokenRepo{rdb: rdb}
}

func (r *redisTokenRepo) Store(ctx context.Context, userID uuid.UUID, token string, ttlSeconds int) error {
	return r.rdb.Set(ctx, "session:"+token, userID, time.Duration(ttlSeconds)*time.Second).Err()
}

func (r *redisTokenRepo) Revoke(ctx context.Context, token string) error {
	return r.rdb.Del(ctx, "session:"+token).Err()
}

func (r *redisTokenRepo) Validate(ctx context.Context, token string) (bool, error) {
	val, err := r.rdb.Get(ctx, "session:"+token).Result()
	if err == redis.Nil {
		return false, nil // Not found means invalid/revoked/expired
	}
	if err != nil {
		return false, err
	}
	return val != "", nil
}
