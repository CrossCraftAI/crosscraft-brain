package core

import (
	"testing"
)

func TestLoopForEach(t *testing.T) {
	res := mustExec(t, loopNode, ctxFor(
		items(map[string]any{"x": 1}, map[string]any{"x": 2}, map[string]any{"x": 3}),
		map[string]any{"mode": "forEach"}))
	out := res.Outputs["main"]
	if len(out) != 3 {
		t.Fatalf("expected 3 items, got %d", len(out))
	}
	if out[0].JSON["item"] != 0 || out[0].JSON["x"] != 1 {
		t.Fatalf("unexpected item 0: %+v", out[0].JSON)
	}
	if out[2].JSON["item"] != 2 {
		t.Fatalf("expected index 2, got %v", out[2].JSON["item"])
	}
}

func TestLoopSplitBatches(t *testing.T) {
	res := mustExec(t, loopNode, ctxFor(
		items(
			map[string]any{"x": 1}, map[string]any{"x": 2}, map[string]any{"x": 3},
			map[string]any{"x": 4}, map[string]any{"x": 5},
		),
		map[string]any{"mode": "splitBatches", "batchSize": float64(2)}))
	out := res.Outputs["main"]
	if len(out) != 3 {
		t.Fatalf("expected 3 batches (2+2+1), got %d", len(out))
	}
	b1 := out[0].JSON
	if b1["batchNum"] != 1 || b1["batchSize"] != 2 || b1["totalItems"] != 5 {
		t.Fatalf("unexpected batch 1: %+v", b1)
	}
	b3 := out[2].JSON
	if b3["batchNum"] != 3 || b3["batchSize"] != 1 {
		t.Fatalf("unexpected batch 3: %+v", b3)
	}
}

func TestLoopEmpty(t *testing.T) {
	// Empty input should produce a single empty item via itemsOrEmpty
	res := mustExec(t, loopNode, ctxFor(nil, map[string]any{"mode": "forEach"}))
	if len(res.Outputs["main"]) != 1 {
		t.Fatalf("expected 1 empty item, got %d", len(res.Outputs["main"]))
	}
}
