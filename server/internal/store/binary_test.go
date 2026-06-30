package store

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiskBinaryStorePutGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewDiskBinaryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	key := BinaryKey("exec1", "node1", "file.txt")
	content := []byte("Hello, streaming world!")

	k, err := store.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	if k != key {
		t.Fatalf("expected key %q, got %q", key, k)
	}

	rc, err := store.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("expected %q, got %q", content, got)
	}
}

func TestDiskBinaryStoreExists(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskBinaryStore(dir)
	ctx := context.Background()
	key := BinaryKey("exec1", "node2", "data.bin")

	exists, err := store.Exists(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("expected key to not exist")
	}

	_, _ = store.Put(ctx, key, strings.NewReader("data"), 4)

	exists, err = store.Exists(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected key to exist")
	}
}

func TestDiskBinaryStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskBinaryStore(dir)
	ctx := context.Background()
	key := BinaryKey("exec1", "node3", "delete-me.txt")

	store.Put(ctx, key, strings.NewReader("data"), 4)

	if err := store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}

	exists, _ := store.Exists(ctx, key)
	if exists {
		t.Fatal("expected key to be deleted")
	}

	// Delete non-existent key should not error
	if err := store.Delete(ctx, "nonexistent/key"); err != nil {
		t.Fatal(err)
	}
}

func TestDiskBinaryStoreGetURL(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskBinaryStore(dir)
	ctx := context.Background()
	key := BinaryKey("exec1", "node4", "image.png")

	store.Put(ctx, key, strings.NewReader("pngdata"), 7)

	url, err := store.GetURL(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "file://") {
		t.Fatalf("expected file:// URL, got %s", url)
	}
}

func TestDiskBinaryStoreCleanup(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskBinaryStore(dir)
	ctx := context.Background()

	// Put files for two runs
	store.Put(ctx, BinaryKey("exec1", "n1", "a.txt"), strings.NewReader("a"), 1)
	store.Put(ctx, BinaryKey("exec1", "n2", "b.txt"), strings.NewReader("b"), 1)
	store.Put(ctx, BinaryKey("exec2", "n1", "c.txt"), strings.NewReader("c"), 1)

	// Cleanup exec1
	if err := store.Cleanup(ctx, BinaryPrefix("exec1")); err != nil {
		t.Fatal(err)
	}

	// exec1 files should be gone
	if exists, _ := store.Exists(ctx, BinaryKey("exec1", "n1", "a.txt")); exists {
		t.Fatal("exec1 files should be cleaned up")
	}
	// exec2 files should remain
	if exists, _ := store.Exists(ctx, BinaryKey("exec2", "n1", "c.txt")); !exists {
		t.Fatal("exec2 files should remain")
	}
}

func TestMemoryBinaryStore(t *testing.T) {
	store := NewMemoryBinaryStore()
	ctx := context.Background()
	key := "test/file.txt"

	store.Put(ctx, key, strings.NewReader("mem data"), 8)
	rc, err := store.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	data, _ := io.ReadAll(rc)
	if string(data) != "mem data" {
		t.Fatalf("unexpected: %s", data)
	}
}

func TestMemoryBinaryStoreCleanup(t *testing.T) {
	store := NewMemoryBinaryStore()
	ctx := context.Background()

	store.Put(ctx, "run/e1/n1/a.txt", strings.NewReader("a"), 1)
	store.Put(ctx, "run/e1/n2/b.txt", strings.NewReader("b"), 1)
	store.Put(ctx, "run/e2/n1/c.txt", strings.NewReader("c"), 1)

	store.Cleanup(ctx, "run/e1/")

	exists, _ := store.Exists(ctx, "run/e1/n1/a.txt")
	if exists {
		t.Fatal("e1 files should be cleaned")
	}
	exists, _ = store.Exists(ctx, "run/e2/n1/c.txt")
	if !exists {
		t.Fatal("e2 files should remain")
	}
}

func TestSizeLimitedReader(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 1000)
	lr := NewSizeLimitedReader(bytes.NewReader(data), 10)

	buf := make([]byte, 5)
	n, err := lr.Read(buf)
	if err != nil || n != 5 {
		t.Fatalf("first read: %d, %v", n, err)
	}

	n, err = lr.Read(buf)
	if err != nil || n != 5 {
		t.Fatalf("second read: %d, %v", n, err)
	}

	// Third read should error
	_, err = lr.Read(buf)
	if err == nil {
		t.Fatal("expected size limit error")
	}
	if lr.TotalRead() != 10 {
		t.Fatalf("expected 10 total, got %d", lr.TotalRead())
	}
}

func TestLimitedPut(t *testing.T) {
	store := NewMemoryBinaryStore()
	ctx := context.Background()

	// Known size, within limit
	_, err := LimitedPut(ctx, store, "k1", strings.NewReader("hello"), 5, 10)
	if err != nil {
		t.Fatal(err)
	}

	// Known size, exceeds limit
	_, err = LimitedPut(ctx, store, "k2", strings.NewReader("hello world"), 11, 10)
	if err != ErrTooLarge {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}

	// Unknown size, within limit
	_, err = LimitedPut(ctx, store, "k3", strings.NewReader("data"), -1, 10)
	if err != nil {
		t.Fatal(err)
	}

	// Unknown size, exceeds limit
	_, err = LimitedPut(ctx, store, "k4", strings.NewReader(strings.Repeat("x", 100)), -1, 10)
	if err == nil {
		t.Fatal("expected size limit error for oversized stream")
	}
}

func TestBinaryKey(t *testing.T) {
	k := BinaryKey("exec-123", "node-abc", "report.pdf")
	if !strings.HasPrefix(k, "run/exec-123/node-abc/") {
		t.Fatalf("unexpected key format: %s", k)
	}
	if !strings.HasSuffix(k, "report.pdf") {
		t.Fatalf("expected filename in key, got %s", k)
	}
}

func TestBinaryPrefix(t *testing.T) {
	p := BinaryPrefix("exec-456")
	if p != "run/exec-456/" {
		t.Fatalf("unexpected prefix: %s", p)
	}
}

func TestDiskBinaryStoreSafePath(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskBinaryStore(dir)

	// Attempt to escape via "../" should be caught
	badKey := "run/../../../etc/passwd"
	safe := store.safePath(badKey)
	if strings.Contains(safe, "etc") {
		t.Fatalf("path traversal not caught: %s", safe)
	}
	// Should be under rootDir
	if !strings.HasPrefix(filepath.Clean(safe), filepath.Clean(dir)) {
		t.Fatalf("safe path %s not under rootDir %s", safe, dir)
	}
}

func TestDiskBinaryStoreCreateDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested", "store")
	store, err := NewDiskBinaryStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("store directory was not created")
	}
	_ = store
}
