package triggers

import (
	"testing"
	"time"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestPollerShouldFire(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}
	p := New(ctx, Opts{Scope: "test:fire:check", Interval: 100 * time.Millisecond})

	// First call: should fire
	if !p.ShouldFire(ctx) {
		t.Fatal("expected first poll to fire")
	}

	// Immediate second call: should NOT fire (rate-limited)
	if p.ShouldFire(ctx) {
		t.Fatal("expected second poll to be rate-limited")
	}
}

func TestPollerEmitDeduplication(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}
	p := New(ctx, Opts{Scope: "test:dedup:items", Interval: time.Second})

	items := []schema.Item{
		{JSON: map[string]any{"id": "a", "value": 1}},
		{JSON: map[string]any{"id": "b", "value": 2}},
	}

	// First emit: both items are new
	emitted := p.Emit(ctx, items, WithIDFunc(func(item schema.Item) string {
		return item.JSON["id"].(string)
	}))
	if len(emitted) != 2 {
		t.Fatalf("expected 2 new items, got %d", len(emitted))
	}

	// Second emit with same items plus one new
	items2 := []schema.Item{
		{JSON: map[string]any{"id": "a", "value": 1}}, // already seen
		{JSON: map[string]any{"id": "c", "value": 3}}, // new
	}
	emitted2 := p.Emit(ctx, items2, WithIDFunc(func(item schema.Item) string {
		return item.JSON["id"].(string)
	}))
	if len(emitted2) != 1 {
		t.Fatalf("expected 1 new item, got %d", len(emitted2))
	}
	if emitted2[0].JSON["id"] != "c" {
		t.Fatalf("expected item c, got %v", emitted2[0].JSON["id"])
	}
}

func TestPollerIdleBackoff(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}
	p := New(ctx, Opts{
		Scope:        "test:idle:backoff",
		Interval:     100 * time.Millisecond,
		MaxIdlePolls: 3,
	})

	// Fire first poll
	p.ShouldFire(ctx)

	// Emit empty items several times
	for i := 0; i < 8; i++ {
		p.Emit(ctx, []schema.Item{}, WithIDFunc(func(item schema.Item) string { return "x" }))
	}

	// After many idle polls, the interval should be longer
	eff := p.effectiveInterval()
	if eff <= 100*time.Millisecond {
		t.Fatalf("expected backoff to increase interval beyond 100ms, got %v", eff)
	}
	// Should be capped at 30 minutes
	if eff > 30*time.Minute {
		t.Fatalf("expected interval cap at 30m, got %v", eff)
	}
}

func TestPollerCursor(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}
	p := New(ctx, Opts{Scope: "test:cursor:pos", Interval: time.Second})

	p.ShouldFire(ctx)
	p.Emit(ctx, []schema.Item{
		{JSON: map[string]any{"id": "item1"}},
	}, WithIDFunc(func(item schema.Item) string {
		return item.JSON["id"].(string)
	}), WithCursor("cursor-5"))

	// Check cursor was stored
	ps := loadState(ctx.State, "test:cursor:pos")
	if ps.Cursor != "cursor-5" {
		t.Fatalf("expected cursor 'cursor-5', got %q", ps.Cursor)
	}
}

func TestPollerStatePersistence(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}
	scope := "test:persist:state"

	// First session
	p1 := New(ctx, Opts{Scope: scope, Interval: time.Second})
	p1.ShouldFire(ctx)
	p1.Emit(ctx, []schema.Item{
		{JSON: map[string]any{"id": "x"}},
		{JSON: map[string]any{"id": "y"}},
	}, WithIDFunc(func(item schema.Item) string { return item.JSON["id"].(string) }))

	// Second session (simulates restart / re-execution)
	p2 := New(ctx, Opts{Scope: scope, Interval: time.Second})

	// Emit same items again — should be deduplicated from persisted state
	emitted := p2.Emit(ctx, []schema.Item{
		{JSON: map[string]any{"id": "x"}},
		{JSON: map[string]any{"id": "z"}},
	}, WithIDFunc(func(item schema.Item) string { return item.JSON["id"].(string) }))

	if len(emitted) != 1 {
		t.Fatalf("expected 1 new item across sessions, got %d", len(emitted))
	}
	if emitted[0].JSON["id"] != "z" {
		t.Fatalf("expected item z, got %v", emitted[0].JSON["id"])
	}
}

func TestPollerStats(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}
	p := New(ctx, Opts{Scope: "test:stats", Interval: time.Second})

	p.ShouldFire(ctx)
	stats := p.Stats()

	if stats["scope"] != "test:stats" {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats["pollCount"] != 1 {
		t.Fatalf("expected pollCount 1, got %v", stats["pollCount"])
	}
}

func TestPollerReset(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}
	p := New(ctx, Opts{Scope: "test:reset", Interval: time.Second})

	p.ShouldFire(ctx)
	p.Emit(ctx, []schema.Item{
		{JSON: map[string]any{"id": "old"}},
	}, WithIDFunc(func(item schema.Item) string { return item.JSON["id"].(string) }))

	// Reset
	p.Reset(ctx)

	// Should fire again
	if !p.ShouldFire(ctx) {
		t.Fatal("expected to fire after reset")
	}

	// Same item should be new again
	emitted := p.Emit(ctx, []schema.Item{
		{JSON: map[string]any{"id": "old"}},
	}, WithIDFunc(func(item schema.Item) string { return item.JSON["id"].(string) }))

	if len(emitted) != 1 {
		t.Fatalf("expected item to be new after reset, got %d", len(emitted))
	}
}

func TestPollerPollConvenience(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}
	p := New(ctx, Opts{Scope: "test:poll:conv", Interval: time.Second})

	called := false
	items, err := p.Poll(ctx, func(ctx *schema.ExecContext, cursor string) ([]schema.Item, string, error) {
		called = true
		if cursor != "" {
			t.Fatalf("expected empty cursor on first call, got %q", cursor)
		}
		return []schema.Item{
			{JSON: map[string]any{"id": "fetched"}},
		}, "cursor-1", nil
	}, WithIDFunc(func(item schema.Item) string {
		return item.JSON["id"].(string)
	}))

	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected fetch function to be called")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 fetched item, got %d", len(items))
	}

	// Second call within interval should not fire
	called = false
	items, err = p.Poll(ctx, func(ctx *schema.ExecContext, cursor string) ([]schema.Item, string, error) {
		called = true
		return nil, "", nil
	}, WithIDFunc(func(item schema.Item) string { return "x" }))

	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("expected fetch NOT to be called (rate-limited)")
	}
	if items != nil {
		t.Fatalf("expected nil items when rate-limited, got %+v", items)
	}
}

func TestConvenienceConstructors(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}

	p1 := NewSheetsTrigger(ctx, "SHEET1", "A1:B10", "rowAdded", 30)
	if p1 == nil {
		t.Fatal("NewSheetsTrigger returned nil")
	}

	p2 := NewGmailTrigger(ctx, "from:me", "is:unread", 60)
	if p2 == nil {
		t.Fatal("NewGmailTrigger returned nil")
	}

	p3 := NewDriveTrigger(ctx, "mimeType='text/plain'", "folder1", 30)
	if p3 == nil {
		t.Fatal("NewDriveTrigger returned nil")
	}

	p4 := NewCalendarTrigger(ctx, "primary", "2025-01-01T00:00:00Z", "2025-01-08T00:00:00Z", 60)
	if p4 == nil {
		t.Fatal("NewCalendarTrigger returned nil")
	}

	p5 := NewFormsTrigger(ctx, "form1", 30)
	if p5 == nil {
		t.Fatal("NewFormsTrigger returned nil")
	}

	p6 := NewGenericTrigger(ctx, "customService", "myEntity", 10)
	if p6 == nil {
		t.Fatal("NewGenericTrigger returned nil")
	}
}

func TestIntervalSeconds(t *testing.T) {
	if d := IntervalSeconds(0); d < DefaultMinInterval {
		t.Fatalf("expected zero seconds to clamp to min, got %v", d)
	}
	if d := IntervalSeconds(5); d < DefaultMinInterval {
		t.Fatalf("expected 5s to clamp to min, got %v", d)
	}
	if d := IntervalSeconds(60); d != 60*time.Second {
		t.Fatalf("expected 60s, got %v", d)
	}
}

func TestHashScope(t *testing.T) {
	h1 := hashScope("sheets:rowAdded:SHEET1!A1:B10")
	h2 := hashScope("sheets:rowAdded:SHEET1!A1:B10")
	h3 := hashScope("sheets:rowAdded:SHEET2!A1:B10")

	if h1 != h2 {
		t.Fatal("same input should produce same hash")
	}
	if h1 == h3 {
		t.Fatal("different inputs should produce different hashes")
	}
}

func TestPollNoIDFunc(t *testing.T) {
	ctx := &schema.ExecContext{State: map[string]any{}}
	p := New(ctx, Opts{Scope: "test:noid", Interval: time.Second})

	p.ShouldFire(ctx)

	// Without IDFunc, all items pass through
	items := []schema.Item{
		{JSON: map[string]any{"id": "a"}},
		{JSON: map[string]any{"id": "a"}},
	}
	emitted := p.Emit(ctx, items) // no WithIDFunc
	if len(emitted) != 2 {
		t.Fatalf("expected both items without dedup, got %d", len(emitted))
	}
}

var _ = schema.ExecContext{}
