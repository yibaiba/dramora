package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FilesystemStorage persists media bytes onto the local disk under `root`.
// 适合本地开发态需要把生成产物落到磁盘 / 共享卷的场景，对外仍以 Storage 接口
// 暴露 Put/Get 语义。
//
// URI 形如 `file://<rel-key>`。read 时把 scheme 去掉再 join 到 root 上读取，
// 对 ../ 路径穿越做了 cleaning + 边界校验，避免越出 root。
type FilesystemStorage struct {
	root string

	mu sync.RWMutex
}

func NewFilesystemStorage(root string) (*FilesystemStorage, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("media: filesystem root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("media: resolve root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("media: prepare root: %w", err)
	}
	return &FilesystemStorage{root: abs}, nil
}

func (f *FilesystemStorage) Root() string { return f.root }

func (f *FilesystemStorage) Put(_ context.Context, key string, body io.Reader, contentType string) (Object, error) {
	cleanKey, err := f.safeKey(key)
	if err != nil {
		return Object{}, err
	}
	if body == nil {
		return Object{}, fmt.Errorf("media: body is required")
	}
	target := filepath.Join(f.root, cleanKey)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Object{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	tmp, err := os.CreateTemp(filepath.Dir(target), ".upload-*")
	if err != nil {
		return Object{}, err
	}
	written, err := io.Copy(tmp, body)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return Object{}, err
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		_ = os.Remove(tmp.Name())
		return Object{}, err
	}
	return Object{
		URI:         "file://" + filepath.ToSlash(cleanKey),
		ContentType: contentType,
		SizeBytes:   written,
	}, nil
}

func (f *FilesystemStorage) Get(_ context.Context, uri string) (io.ReadCloser, error) {
	const prefix = "file://"
	if !strings.HasPrefix(uri, prefix) {
		return nil, fmt.Errorf("media: uri %q does not match scheme %q", uri, "file")
	}
	cleanKey, err := f.safeKey(strings.TrimPrefix(uri, prefix))
	if err != nil {
		return nil, err
	}
	return os.Open(filepath.Join(f.root, cleanKey))
}

func (f *FilesystemStorage) safeKey(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("media: key is required")
	}
	if filepath.IsAbs(raw) || strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("media: key %q must be relative", raw)
	}
	for _, seg := range strings.Split(filepath.ToSlash(raw), "/") {
		if seg == ".." {
			return "", fmt.Errorf("media: key %q escapes root", raw)
		}
	}
	cleaned := filepath.Clean(raw)
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("media: key %q invalid", raw)
	}
	return cleaned, nil
}
