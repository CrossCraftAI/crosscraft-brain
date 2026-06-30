package database

import (
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/credtype"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
	"github.com/redis/go-redis/v9"
)

func TestRedisNodeDefinition(t *testing.T) {
	def := RedisNode()
	if def.Type != "database.redis" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if len(def.Params) == 0 {
		t.Fatal("expected params to be defined")
	}
	if def.Execute == nil {
		t.Fatal("expected execute function to be set")
	}
	if len(def.Outputs) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(def.Outputs))
	}

	opCount := countOperations(def)
	if opCount != 16 {
		t.Fatalf("expected 16 operations, got %d", opCount)
	}
}

func TestRedisNodeUsesRegisteredCredentialType(t *testing.T) {
	reg := credtype.Default()
	credType, ok := reg.Get("redisApi")
	if !ok {
		t.Fatal("expected redisApi credential type to be registered")
	}
	if credType.DisplayName == "" {
		t.Fatal("expected redisApi credential type to have a display name")
	}
	if len(credType.Fields) == 0 {
		t.Fatal("expected redisApi credential fields to be defined")
	}
}

func TestResolveRedisOptsRequiresCredential(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, err := resolveRedisOpts(ctx)
	if err == nil {
		t.Fatal("expected error when credential is missing")
	}
}

func TestResolveRedisOptsDefaults(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return map[string]any{
				"host": "redis.example.com",
			}, nil
		},
	}
	opts, err := resolveRedisOpts(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Addr != "redis.example.com:6379" {
		t.Fatalf("expected default port, got %s", opts.Addr)
	}
	if opts.DB != 0 {
		t.Fatalf("expected default DB 0, got %d", opts.DB)
	}
}

func TestResolveRedisOptsFullConfig(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return map[string]any{
				"host":     "redis.internal",
				"port":     "6380",
				"password": "secret123",
				"db":       "5",
			}, nil
		},
	}
	opts, err := resolveRedisOpts(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Addr != "redis.internal:6380" {
		t.Fatalf("unexpected addr: %s", opts.Addr)
	}
	if opts.Password != "secret123" {
		t.Fatalf("unexpected password: %s", opts.Password)
	}
	if opts.DB != 5 {
		t.Fatalf("unexpected DB: %d", opts.DB)
	}
}

func TestRedisExecuteRequiresCredential(t *testing.T) {
	ctx := &schema.ExecContext{
		Params: map[string]any{
			"operation": "get",
			"key":       "testkey",
		},
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, err := executeRedis(ctx)
	if err == nil {
		t.Fatal("expected error when credential is missing")
	}
}

func TestRedisClientCached(t *testing.T) {
	originalFactory := newRedisClient
	redisClientMu.Lock()
	redisClientCache = make(map[string]*redis.Client)
	redisClientMu.Unlock()
	t.Cleanup(func() {
		newRedisClient = originalFactory
		redisClientMu.Lock()
		redisClientCache = make(map[string]*redis.Client)
		redisClientMu.Unlock()
	})

	var createCalls int
	newRedisClient = func(opts *redis.Options) (*redis.Client, error) {
		createCalls++
		return &redis.Client{}, nil
	}

	opts := &redis.Options{Addr: "localhost:6379", DB: 0}
	client1, err := getOrCreateRedisClient(opts)
	if err != nil {
		t.Fatalf("expected first client creation to succeed: %v", err)
	}
	client2, err := getOrCreateRedisClient(opts)
	if err != nil {
		t.Fatalf("expected cached client reuse to succeed: %v", err)
	}

	if createCalls != 1 {
		t.Fatalf("expected one client creation, got %d", createCalls)
	}
	if client1 != client2 {
		t.Fatal("expected the same client instance to be reused for the same opts")
	}
}

func TestRedisGetRequiresKey(t *testing.T) {
	ctx := &schema.ExecContext{
		Params: map[string]any{
			"operation": "get",
			// key is missing
		},
		Credential: func(paramName string) (map[string]any, error) {
			return map[string]any{"host": "localhost", "port": "6379"}, nil
		},
	}
	_, err := executeRedis(ctx)
	if err == nil {
		t.Fatal("expected error when key is missing")
	}
}
