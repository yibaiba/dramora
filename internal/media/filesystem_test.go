package media

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestFilesystemStorageRoundTrip(t *testing.T) {
	t.Parallel()
	store, err := NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	obj, err := store.Put(context.Background(), "audio/job-1.mp3", strings.NewReader("hello"), "audio/mpeg")
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if obj.URI != "file://audio/job-1.mp3" {
		t.Fatalf("uri=%q", obj.URI)
	}
	if obj.SizeBytes != 5 {
		t.Fatalf("size=%d", obj.SizeBytes)
	}
	rc, err := store.Get(context.Background(), obj.URI)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer rc.Close()
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("body=%q", string(body))
	}
}

func TestFilesystemStorageRejectsTraversal(t *testing.T) {
	t.Parallel()
	store, err := NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := store.Put(context.Background(), "../../../etc/passwd", strings.NewReader("x"), ""); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if _, err := store.Get(context.Background(), "file://../../etc/passwd"); err == nil {
		t.Fatal("expected get traversal rejection")
	}
}

func TestFilesystemStorageRejectsWrongScheme(t *testing.T) {
	t.Parallel()
	store, err := NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := store.Get(context.Background(), "mem://foo"); err == nil {
		t.Fatal("expected scheme rejection")
	}
}

func TestFilesystemStorageEmptyRootRejected(t *testing.T) {
	t.Parallel()
	if _, err := NewFilesystemStorage(""); err == nil {
		t.Fatal("expected empty root rejection")
	}
}
