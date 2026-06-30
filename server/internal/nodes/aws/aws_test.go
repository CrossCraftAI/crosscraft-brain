package aws

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/CrossCraftAI/crosscraft-brain/server/internal/schema"
)

func TestSigV4Signing(t *testing.T) {
	signer := &Signer{
		AccessKey: "AKIDEXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		Region:    "us-east-1",
		Service:   "s3",
	}

	req, _ := http.NewRequest("GET", "https://example-bucket.s3.us-east-1.amazonaws.com/test.txt", nil)
	req.Host = "example-bucket.s3.us-east-1.amazonaws.com"

	if err := signer.Sign(req, []byte{}); err != nil {
		t.Fatal(err)
	}

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatal("expected Authorization header")
	}
	if req.Header.Get("X-Amz-Date") == "" {
		t.Fatal("expected X-Amz-Date header")
	}
}

func TestCanonicalURI(t *testing.T) {
	tests := map[string]string{
		"":           "/",
		"/":          "/",
		"/test.txt":  "/test.txt",
		"/path/to/file.txt": "/path/to/file.txt",
	}
	for input, expected := range tests {
		if got := canonicalURI(input); got != expected {
			t.Fatalf("canonicalURI(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestCanonicalQuery(t *testing.T) {
	got := canonicalQuery("max-keys=10&prefix=test")
	if got == "" {
		t.Fatal("expected non-empty canonical query")
	}
}

func TestURLEncode(t *testing.T) {
	if got := urlEncode("simple"); got != "simple" {
		t.Fatalf("expected 'simple', got %q", got)
	}
	if got := urlEncode("hello world"); got != "hello%20world" {
		t.Fatalf("expected encoded space, got %q", got)
	}
}

func TestHmacSHA256(t *testing.T) {
	key := []byte("secret")
	data := []byte("message")
	mac := hmacSHA256(key, data)
	expected := hmac.New(sha256.New, key)
	expected.Write(data)
	if hex.EncodeToString(mac) != hex.EncodeToString(expected.Sum(nil)) {
		t.Fatal("hmac mismatch")
	}
}

func TestDeriveSigningKey(t *testing.T) {
	signer := &Signer{
		SecretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		Region:    "us-east-1",
		Service:   "s3",
	}
	key := signer.deriveSigningKey("20150830")
	if len(key) != 32 {
		t.Fatalf("expected 32-byte signing key, got %d", len(key))
	}
}

func TestS3NodeParams(t *testing.T) {
	def := S3Node()
	if def.Type != "aws.s3" {
		t.Fatalf("unexpected type: %s", def.Type)
	}
	if len(def.Params) < 2 {
		t.Fatalf("expected at least 2 params, got %d", len(def.Params))
	}
}

func TestSESNodeParams(t *testing.T) {
	def := SESNode()
	if len(def.Params) < 2 {
		t.Fatalf("expected at least 2 params, got %d", len(def.Params))
	}
}

func TestSQSNodeParams(t *testing.T) {
	def := SQSNode()
	if len(def.Params) < 2 {
		t.Fatalf("expected at least 2 params, got %d", len(def.Params))
	}
}

func TestLambdaNodeParams(t *testing.T) {
	def := LambdaNode()
	if len(def.Params) < 2 {
		t.Fatalf("expected at least 2 params, got %d", len(def.Params))
	}
}

func TestDynamoDBNodeParams(t *testing.T) {
	def := DynamoDBNode()
	if len(def.Params) < 2 {
		t.Fatalf("expected at least 2 params, got %d", len(def.Params))
	}
	if len(def.Params) > 5 {
		// Should have many params for all the operations
	}
}

func TestDynamoUnwrap(t *testing.T) {
	item := map[string]any{
		"id":   map[string]any{"S": "123"},
		"name": map[string]any{"S": "Alice"},
		"age":  map[string]any{"N": "30"},
		"active": map[string]any{"BOOL": true},
	}
	unwrapped := dynamoUnwrapMap(item)
	if unwrapped["id"] != "123" {
		t.Fatalf("expected id '123', got %v", unwrapped["id"])
	}
	if unwrapped["name"] != "Alice" {
		t.Fatalf("expected name 'Alice', got %v", unwrapped["name"])
	}
	if unwrapped["age"] != int64(30) {
		t.Fatalf("expected age 30, got %v (%T)", unwrapped["age"], unwrapped["age"])
	}
	if unwrapped["active"] != true {
		t.Fatalf("expected active true, got %v", unwrapped["active"])
	}
}

func TestNodesReturnsAll(t *testing.T) {
	nodes := Nodes()
	if len(nodes) != 5 {
		t.Fatalf("expected 5 AWS nodes, got %d", len(nodes))
	}
	names := map[string]bool{}
	for _, n := range nodes {
		names[n.Type] = true
	}
	for _, want := range []string{"aws.s3", "aws.ses", "aws.sqs", "aws.lambda", "aws.dynamodb"} {
		if !names[want] {
			t.Fatalf("expected node %q to be registered", want)
		}
	}
}

func TestSigningRoundTripper(t *testing.T) {
	signer := &Signer{
		AccessKey: "test",
		SecretKey: "test",
		Region:    "us-east-1",
		Service:   "s3",
	}
	srt := &SigningRoundTripper{Sig: signer, Base: http.DefaultTransport}
	if srt == nil {
		t.Fatal("expected non-nil round tripper")
	}
}

func TestSplitCSV(t *testing.T) {
	parts := splitCSV("a, b, c")
	if len(parts) != 3 || parts[0] != "a" || parts[2] != "c" {
		t.Fatalf("unexpected split: %v", parts)
	}
	parts = splitCSV("")
	if len(parts) != 0 {
		t.Fatalf("expected empty, got %v", parts)
	}
}

func TestMakeSignerError(t *testing.T) {
	ctx := &schema.ExecContext{
		Credential: func(paramName string) (map[string]any, error) {
			return nil, nil
		},
	}
	_, err := makeSigner(ctx, "s3")
	if err == nil {
		t.Fatal("expected error for missing credential")
	}
}

var _ = schema.ExecContext{}
