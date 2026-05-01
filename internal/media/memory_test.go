package media

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestMemoryStoragePutAndGetRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewMemoryStorage()
	obj, err := store.Put(context.Background(), "audio/job-1.mp3", strings.NewReader("hello-bytes"), "audio/mpeg")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if obj.URI != "mem://audio/job-1.mp3" {
		t.Fatalf("URI=%q want mem://audio/job-1.mp3", obj.URI)
	}
	if obj.SizeBytes != int64(len("hello-bytes")) {
		t.Fatalf("SizeBytes=%d want %d", obj.SizeBytes, len("hello-bytes"))
	}
	rc, err := store.Get(context.Background(), obj.URI)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "hello-bytes" {
		t.Fatalf("round-trip data=%q want hello-bytes", string(data))
	}
}

func TestMemoryStorageGetUnknownURI(t *testing.T) {
	t.Parallel()

	store := NewMemoryStorage()
	if _, err := store.Get(context.Background(), "mem://does/not/exist"); err == nil {
		t.Fatal("expected error for unknown key")
	}
	if _, err := store.Get(context.Background(), "https://other.example.com/x"); err == nil {
		t.Fatal("expected error for foreign scheme URI")
	}
}

func TestMemoryStoragePutValidatesInputs(t *testing.T) {
	t.Parallel()

	store := NewMemoryStorage()
	if _, err := store.Put(context.Background(), "  ", strings.NewReader("x"), "text/plain"); err == nil {
		t.Fatal("expected error for empty key")
	}
	if _, err := store.Put(context.Background(), "k", nil, "text/plain"); err == nil {
		t.Fatal("expected error for nil body")
	}
}
