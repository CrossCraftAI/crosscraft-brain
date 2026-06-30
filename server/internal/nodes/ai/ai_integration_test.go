package ai

import (
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/llm"
	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func collectOps(def schema.NodeDefinition) map[string]bool {
	ops := map[string]bool{}
	for _, p := range def.Params {
		if p.Name == "operation" {
			for _, opt := range p.Options {
				ops[opt.Value] = true
			}
		}
	}
	return ops
}

func TestAINodeCount(t *testing.T) {
	c := &llm.Client{}
	nodes := Nodes(c)
	if len(nodes) != 12 {
		t.Fatalf("expected 12 AI nodes (3 LLM + 9 integrations), got %d", len(nodes))
	}
}

// ---- Hugging Face ----

func TestHuggingFaceNode(t *testing.T) {
	def := HuggingFace()
	if def.Type != "ai.huggingface" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if def.Execute == nil {
		t.Fatal("expected execute function")
	}
	ops := collectOps(def)
	for _, want := range []string{"inference:run", "model:list", "model:get", "embedding:create"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

// ---- Cohere ----

func TestCohereNode(t *testing.T) {
	def := Cohere()
	if def.Type != "ai.cohere" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	for _, want := range []string{"generate", "embed", "classify", "summarize", "chat"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

// ---- Mistral ----

func TestMistralNode(t *testing.T) {
	def := Mistral()
	if def.Type != "ai.mistral" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	for _, want := range []string{"chat:completion", "embed:create", "model:list"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

// ---- Pinecone ----

func TestPineconeNode(t *testing.T) {
	def := Pinecone()
	if def.Type != "ai.pinecone" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if def.Execute == nil {
		t.Fatal("expected execute function")
	}
	ops := collectOps(def)
	for _, want := range []string{"vector:upsert", "vector:query", "vector:delete", "index:list", "index:create", "index:delete"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

// ---- Qdrant ----

func TestQdrantNode(t *testing.T) {
	def := Qdrant()
	if def.Type != "ai.qdrant" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	for _, want := range []string{"vector:upsert", "vector:query", "vector:delete", "collection:list", "collection:create", "collection:delete"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

// ---- ElevenLabs ----

func TestElevenLabsNode(t *testing.T) {
	def := ElevenLabs()
	if def.Type != "ai.elevenlabs" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if len(def.Outputs) != 2 {
		t.Fatalf("expected 2 outputs (main + audio), got %d", len(def.Outputs))
	}
	ops := collectOps(def)
	for _, want := range []string{"tts:generate", "voice:list", "voice:get"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

// ---- Stability AI ----

func TestStabilityAINode(t *testing.T) {
	def := StabilityAI()
	if def.Type != "ai.stability" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	for _, want := range []string{"image:generate", "image:variation", "model:list"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

// ---- Perplexity ----

func TestPerplexityNode(t *testing.T) {
	def := Perplexity()
	if def.Type != "ai.perplexity" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	ops := collectOps(def)
	for _, want := range []string{"chat:completion", "search:run", "model:list"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

// ---- OpenAI ----

func TestOpenAINode(t *testing.T) {
	def := OpenAI()
	if def.Type != "ai.openai" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if len(def.Params) < 8 {
		t.Fatalf("expected at least 8 params, got %d", len(def.Params))
	}
	ops := collectOps(def)
	for _, want := range []string{"chat:completion", "completion:create", "embedding:create", "image:generate", "audio:transcribe", "audio:translate", "moderation:check", "model:list"} {
		if !ops[want] {
			t.Fatalf("expected operation %q", want)
		}
	}
}

// ---- Helper tests ----

func TestStr(t *testing.T) {
	if got := str(map[string]any{"a": "hello"}, "a", ""); got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
	if got := str(map[string]any{}, "missing", "default"); got != "default" {
		t.Fatalf("expected 'default', got %q", got)
	}
}

func TestNum(t *testing.T) {
	if got := num(map[string]any{"x": float64(42)}, "x", 0); got != 42 {
		t.Fatalf("expected 42, got %v", got)
	}
	if got := num(map[string]any{}, "missing", 99); got != 99 {
		t.Fatalf("expected 99, got %v", got)
	}
}

func TestTrunc(t *testing.T) {
	if got := trunc("hello", 3); got != "hel…" {
		t.Fatalf("expected 'hel…', got %q", got)
	}
	if got := trunc("hi", 10); got != "hi" {
		t.Fatalf("expected 'hi', got %q", got)
	}
}

func TestBytesToB64(t *testing.T) {
	result := bytesToB64([]byte("test"))
	if len(result) == 0 {
		t.Fatal("expected non-empty base64 output")
	}
}

func TestItem(t *testing.T) {
	m := map[string]any{"key": "val"}
	it := item(m)
	if it.JSON["key"] != "val" {
		t.Fatal("expected item to preserve map")
	}
}

func TestToItems(t *testing.T) {
	result := map[string]any{
		"data": []any{
			map[string]any{"id": "1"},
			map[string]any{"id": "2"},
		},
	}
	items := toItems(result, "data")
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestGetPath(t *testing.T) {
	root := map[string]any{"a": map[string]any{"b": "c"}}
	if got := getPath(root, "a.b"); got != "c" {
		t.Fatalf("expected 'c', got %v", got)
	}
}
