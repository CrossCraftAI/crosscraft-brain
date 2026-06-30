package database

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
	"github.com/redis/go-redis/v9"
)

var (
	redisClientMu    sync.Mutex
	redisClientCache = make(map[string]*redis.Client)
	newRedisClient   = func(opts *redis.Options) (*redis.Client, error) {
		client := redis.NewClient(opts)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("redis: ping failed: %w", err)
		}
		return client, nil
	}
)

// RedisNode returns the definition for the Redis node.
func RedisNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Type:        "database.redis",
		Label:       "Redis",
		Description: "Manage keys, hashes, lists, sets, and pub/sub on a Redis server.",
		Group:       "storage",
		Icon:        "Database",
		Inputs:      []schema.Port{{ID: "main"}},
		Outputs:     []schema.Port{{ID: "main", Label: "Results"}, {ID: "error", Label: "Error"}},
		Credentials: []string{"redisApi"},
		Params: []schema.ParamSchema{
			{Name: "credential", Label: "Credential", Type: "credential", Required: true, CredentialType: "redisApi"},
			{Name: "operation", Label: "Operation", Type: "select", Required: true, Default: "get", Options: []schema.ParamOption{
				{Label: "Get (key value)", Value: "get"},
				{Label: "Set (key value)", Value: "set"},
				{Label: "Delete (key)", Value: "delete"},
				{Label: "Expire (set TTL)", Value: "expire"},
				{Label: "Increment", Value: "incr"},
				{Label: "Decrement", Value: "decr"},
				{Label: "Hash: Get Field", Value: "hget"},
				{Label: "Hash: Set Field", Value: "hset"},
				{Label: "Hash: Get All", Value: "hgetall"},
				{Label: "List: Push (left)", Value: "lpush"},
				{Label: "List: Pop (right)", Value: "rpop"},
				{Label: "Set: Add", Value: "sadd"},
				{Label: "Set: Members", Value: "smembers"},
				{Label: "Publish", Value: "publish"},
				{Label: "Subscribe", Value: "subscribe"},
				{Label: "Trigger: New Message", Value: "trigger:newMessage"},
			}},
			{Name: "key", Label: "Key", Type: "string", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"get", "delete", "expire", "incr", "decr", "hget", "hgetall", "lpush", "rpop", "sadd", "smembers"}}},
			{Name: "value", Label: "Value", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"set", "lpush", "sadd"}}},
			{Name: "field", Label: "Field", Type: "string",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"hget", "hset"}}},
			{Name: "hashData", Label: "Hash Data (JSON object)", Type: "json",
				Description: "JSON object of field:value pairs for hset",
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"hset"}}},
			{Name: "ttl", Label: "TTL (seconds)", Type: "number", Default: 3600,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"set", "expire"}}},
			{Name: "channel", Label: "Channel", Type: "string", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"publish", "subscribe", "trigger:newMessage"}}},
			{Name: "message", Label: "Message", Type: "string", Required: true,
				ShowWhen: &schema.ShowWhen{Param: "operation", Equals: []any{"publish"}}},
		},
		Execute: executeRedis,
	}
}

// resolveRedisOpts builds Redis options from the credential.
func resolveRedisOpts(ctx *schema.ExecContext) (*redis.Options, error) {
	if ctx.Credential != nil {
		cred, err := ctx.Credential("credential")
		if err != nil {
			return nil, fmt.Errorf("redis: failed to get credentials: %w", err)
		}
		if len(cred) > 0 {
			host, _ := cred["host"].(string)
			port, _ := cred["port"].(string)
			password, _ := cred["password"].(string)
			dbStr, _ := cred["db"].(string)
			if host == "" {
				host = "localhost"
			}
			if port == "" {
				port = "6379"
			}
			db := 0
			if dbStr != "" {
				if n, err := strconv.Atoi(dbStr); err == nil {
					db = n
				}
			}
			return &redis.Options{
				Addr:     fmt.Sprintf("%s:%s", host, port),
				Password: password,
				DB:       db,
			}, nil
		}
	}
	return nil, fmt.Errorf("redis: no credential configured")
}

// getOrCreateRedisClient returns a cached Redis client for the given options.
func getOrCreateRedisClient(opts *redis.Options) (*redis.Client, error) {
	key := fmt.Sprintf("%s/%d", opts.Addr, opts.DB)
	redisClientMu.Lock()
	if client, ok := redisClientCache[key]; ok {
		redisClientMu.Unlock()
		return client, nil
	}
	redisClientMu.Unlock()

	client, err := newRedisClient(opts)
	if err != nil {
		return nil, fmt.Errorf("redis: failed to create client: %w", err)
	}

	redisClientMu.Lock()
	defer redisClientMu.Unlock()
	if existing, ok := redisClientCache[key]; ok {
		client.Close()
		return existing, nil
	}
	redisClientCache[key] = client
	return client, nil
}

// executeRedis is the execution function for the Redis node.
func executeRedis(ctx *schema.ExecContext) (schema.NodeResult, error) {
	opts, err := resolveRedisOpts(ctx)
	if err != nil {
		return schema.NodeResult{}, err
	}

	client, err := getOrCreateRedisClient(opts)
	if err != nil {
		return schema.NodeResult{}, err
	}

	operation, _ := ctx.Params["operation"].(string)
	execCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch operation {
	case "get":
		return redisGet(execCtx, client, ctx)
	case "set":
		return redisSet(execCtx, client, ctx)
	case "delete":
		return redisDelete(execCtx, client, ctx)
	case "expire":
		return redisExpire(execCtx, client, ctx)
	case "incr":
		return redisIncr(execCtx, client, ctx)
	case "decr":
		return redisDecr(execCtx, client, ctx)
	case "hget":
		return redisHGet(execCtx, client, ctx)
	case "hset":
		return redisHSet(execCtx, client, ctx)
	case "hgetall":
		return redisHGetAll(execCtx, client, ctx)
	case "lpush":
		return redisLPush(execCtx, client, ctx)
	case "rpop":
		return redisRPop(execCtx, client, ctx)
	case "sadd":
		return redisSAdd(execCtx, client, ctx)
	case "smembers":
		return redisSMembers(execCtx, client, ctx)
	case "publish":
		return redisPublish(execCtx, client, ctx)
	case "subscribe":
		return redisSubscribe(execCtx, client, ctx)
	case "trigger:newMessage":
		return redisTriggerNewMessage(execCtx, client, ctx)
	default:
		return schema.NodeResult{}, fmt.Errorf("redis: unknown operation %q", operation)
	}
}

// --- operation helpers ---

func itemFromAny(v any) schema.Item {
	if m, ok := v.(map[string]any); ok {
		return schema.Item{JSON: m}
	}
	return schema.Item{JSON: map[string]any{"value": fmt.Sprintf("%v", v)}}
}

func redisGet(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}
	result, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {}}}, nil
	}
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis get failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "value": result}}}}}, nil
}

func redisSet(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	value, _ := execCtx.Params["value"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}
	ttl := int64(0)
	if v, ok := execCtx.Params["ttl"].(int); ok && v > 0 {
		ttl = int64(v)
	} else if v, ok := execCtx.Params["ttl"].(float64); ok && v > 0 {
		ttl = int64(v)
	}
	dur := time.Duration(ttl) * time.Second
	if err := client.Set(ctx, key, value, dur).Err(); err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis set failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "status": "ok"}}}}}, nil
}

func redisDelete(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}
	deleted, err := client.Del(ctx, key).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis delete failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "deleted": deleted}}}}}, nil
}

func redisExpire(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}
	ttl := int64(3600)
	if v, ok := execCtx.Params["ttl"].(int); ok && v > 0 {
		ttl = int64(v)
	} else if v, ok := execCtx.Params["ttl"].(float64); ok && v > 0 {
		ttl = int64(v)
	}
	dur := time.Duration(ttl) * time.Second
	ok, err := client.Expire(ctx, key, dur).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis expire failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "expireSet": ok}}}}}, nil
}

func redisIncr(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}
	result, err := client.Incr(ctx, key).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis incr failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "value": result}}}}}, nil
}

func redisDecr(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}
	result, err := client.Decr(ctx, key).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis decr failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "value": result}}}}}, nil
}

func redisHGet(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	field, _ := execCtx.Params["field"].(string)
	if key == "" || field == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key and field are required")
	}
	result, err := client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {}}}, nil
	}
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis hget failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"field": field, "value": result}}}}}, nil
}

func redisHSet(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}

	// Support both single field:value and JSON object of multiple fields
	field, _ := execCtx.Params["field"].(string)
	hashData := asObjectValue(execCtx.RawParam("hashData"))

	var pairs []any
	if hashData != nil && len(hashData) > 0 {
		for k, v := range hashData {
			pairs = append(pairs, k, fmt.Sprintf("%v", v))
		}
	} else if field != "" {
		value, _ := execCtx.Params["value"].(string)
		pairs = append(pairs, field, value)
	}

	if len(pairs) == 0 {
		return schema.NodeResult{}, fmt.Errorf("redis: field+value or hashData is required for hset")
	}

	result, err := client.HSet(ctx, key, pairs...).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis hset failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "fieldsSet": result}}}}}, nil
}

func redisHGetAll(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}
	result, err := client.HGetAll(ctx, key).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis hgetall failed: %w", err)
	}
	// Convert map[string]string to map[string]any
	m := make(map[string]any, len(result))
	for k, v := range result {
		m[k] = v
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: m}}}}, nil
}

func redisLPush(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	value, _ := execCtx.Params["value"].(string)
	if key == "" || value == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key and value are required")
	}
	result, err := client.LPush(ctx, key, value).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis lpush failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "length": result}}}}}, nil
}

func redisRPop(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}
	result, err := client.RPop(ctx, key).Result()
	if err == redis.Nil {
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {}}}, nil
	}
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis rpop failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "value": result}}}}}, nil
}

func redisSAdd(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	value, _ := execCtx.Params["value"].(string)
	if key == "" || value == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key and value are required")
	}
	result, err := client.SAdd(ctx, key, value).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis sadd failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"key": key, "added": result}}}}}, nil
}

func redisSMembers(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	key, _ := execCtx.Params["key"].(string)
	if key == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: key is required")
	}
	members, err := client.SMembers(ctx, key).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis smembers failed: %w", err)
	}
	var out []schema.Item
	for _, m := range members {
		out = append(out, schema.Item{JSON: map[string]any{"member": m}})
	}
	if out == nil {
		out = []schema.Item{}
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": out}}, nil
}

func redisPublish(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	channel, _ := execCtx.Params["channel"].(string)
	message, _ := execCtx.Params["message"].(string)
	if channel == "" || message == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: channel and message are required")
	}
	received, err := client.Publish(ctx, channel, message).Result()
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis publish failed: %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{"channel": channel, "receivedBy": received}}}}}, nil
}

func redisSubscribe(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	channel, _ := execCtx.Params["channel"].(string)
	if channel == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: channel is required")
	}
	sub := client.Subscribe(ctx, channel)
	defer sub.Close()

	// Wait for a single message with a timeout
	msgCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	msg, err := sub.ReceiveMessage(msgCtx)
	if err != nil {
		return schema.NodeResult{}, fmt.Errorf("redis subscribe failed (no message received): %w", err)
	}
	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{
		"channel": msg.Channel,
		"pattern": msg.Pattern,
		"payload": msg.Payload,
	}}}}}, nil
}

func redisTriggerNewMessage(ctx context.Context, client *redis.Client, execCtx *schema.ExecContext) (schema.NodeResult, error) {
	channel, _ := execCtx.Params["channel"].(string)
	if channel == "" {
		return schema.NodeResult{}, fmt.Errorf("redis: channel is required")
	}

	// Subscribe with a short timeout for polling-based trigger
	sub := client.Subscribe(ctx, channel)
	defer sub.Close()

	msgCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msg, err := sub.ReceiveMessage(msgCtx)
	if err != nil {
		// No message within timeout — return empty (caller will re-poll)
		return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {}}}, nil
	}

	// Store the last seen message timestamp in state for dedup
	execCtx.State["lastMessageTs"] = time.Now().UnixMilli()

	return schema.NodeResult{Outputs: map[string][]schema.Item{"main": {{JSON: map[string]any{
		"channel": msg.Channel,
		"payload": msg.Payload,
	}}}}}, nil
}
