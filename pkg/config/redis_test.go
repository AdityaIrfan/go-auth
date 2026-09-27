package config

import "testing"

func TestNewRedisConn(t *testing.T) {
	t.Setenv("REDIS_HOST", "redis.example:6380")
	t.Setenv("REDIS_PASSWORD", "secret")
	client := NewRedisConn()
	defer client.Close()
	options := client.Options()
	if options.Addr != "redis.example:6380" || options.Password != "secret" || options.DB != 0 {
		t.Fatalf("unexpected options: %#v", options)
	}
}
