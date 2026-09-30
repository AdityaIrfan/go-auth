package repositories

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type fakeRedisCommands struct {
	values map[string]string
	ttls   map[string]time.Duration
	setErr error
	delErr error
	getErr error
}

func (f *fakeRedisCommands) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if f.setErr != nil {
		cmd.SetErr(f.setErr)
		return cmd
	}
	f.values[key] = fmt.Sprint(value)
	f.ttls[key] = expiration
	cmd.SetVal("OK")
	return cmd
}

func (f *fakeRedisCommands) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	if f.delErr != nil {
		cmd.SetErr(f.delErr)
		return cmd
	}
	var deleted int64
	for _, key := range keys {
		if _, ok := f.values[key]; ok {
			delete(f.values, key)
			delete(f.ttls, key)
			deleted++
		}
	}
	cmd.SetVal(deleted)
	return cmd
}

func (f *fakeRedisCommands) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	if f.getErr != nil {
		cmd.SetErr(f.getErr)
		return cmd
	}
	value, ok := f.values[key]
	if !ok {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(value)
	return cmd
}

func TestRedisTokenRepositoryLifecycle(t *testing.T) {
	client := &fakeRedisCommands{values: map[string]string{}, ttls: map[string]time.Duration{}}
	repo := &redisTokenRepo{rdb: client}
	ctx := context.Background()
	userID := uuid.New()

	if err := repo.Store(ctx, userID, "token-1", 120); err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	if got := client.values["session:token-1"]; got != userID.String() {
		t.Fatalf("stored value = %q", got)
	}
	if ttl := client.ttls["session:token-1"]; ttl != 120*time.Second {
		t.Fatalf("TTL = %v", ttl)
	}

	valid, err := repo.Validate(ctx, "token-1")
	if err != nil || !valid {
		t.Fatalf("Validate() = %v, %v", valid, err)
	}
	if err := repo.Revoke(ctx, "token-1"); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	valid, err = repo.Validate(ctx, "token-1")
	if err != nil || valid {
		t.Fatalf("Validate() after revoke = %v, %v", valid, err)
	}
}

func TestRedisTokenRepositoryErrors(t *testing.T) {
	want := errors.New("redis unavailable")
	ctx := context.Background()

	repo := &redisTokenRepo{rdb: &fakeRedisCommands{values: map[string]string{}, ttls: map[string]time.Duration{}, setErr: want}}
	if err := repo.Store(ctx, uuid.New(), "token", 10); !errors.Is(err, want) {
		t.Fatalf("Store() error = %v", err)
	}

	repo.rdb = &fakeRedisCommands{values: map[string]string{}, ttls: map[string]time.Duration{}, delErr: want}
	if err := repo.Revoke(ctx, "token"); !errors.Is(err, want) {
		t.Fatalf("Revoke() error = %v", err)
	}

	repo.rdb = &fakeRedisCommands{values: map[string]string{}, ttls: map[string]time.Duration{}, getErr: want}
	if valid, err := repo.Validate(ctx, "token"); !errors.Is(err, want) || valid {
		t.Fatalf("Validate() = %v, %v", valid, err)
	}
}

func TestNewRedisTokenRepository(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	defer client.Close()
	if NewRedisTokenRepository(client) == nil {
		t.Fatal("constructor returned nil")
	}
}
