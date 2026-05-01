package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
)

// MemoryStorage 是 Storage 的本地内存实现，主要用于开发态 / 测试 /
// inline worker。生产上应当替换为 S3 / OSS / 本地文件系统等带持久层的实现，
// 共享相同的 URI -> bytes 语义即可。
//
// 写入返回的 URI 形如 `mem://<key>`，便于在前端 / 资产 DB 中识别归属的存储后端。
type MemoryStorage struct {
	scheme string

	mu      sync.RWMutex
	objects map[string]memoryObject
}

type memoryObject struct {
	contentType string
	data        []byte
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{scheme: "mem", objects: make(map[string]memoryObject)}
}

func (m *MemoryStorage) Put(_ context.Context, key string, body io.Reader, contentType string) (Object, error) {
	if strings.TrimSpace(key) == "" {
		return Object{}, fmt.Errorf("media: key is required")
	}
	if body == nil {
		return Object{}, fmt.Errorf("media: body is required")
	}
	buf, err := io.ReadAll(body)
	if err != nil {
		return Object{}, err
	}
	m.mu.Lock()
	m.objects[key] = memoryObject{contentType: contentType, data: append([]byte(nil), buf...)}
	m.mu.Unlock()
	return Object{
		URI:         fmt.Sprintf("%s://%s", m.scheme, strings.TrimPrefix(key, "/")),
		ContentType: contentType,
		SizeBytes:   int64(len(buf)),
	}, nil
}

func (m *MemoryStorage) Get(_ context.Context, uri string) (io.ReadCloser, error) {
	prefix := m.scheme + "://"
	if !strings.HasPrefix(uri, prefix) {
		return nil, fmt.Errorf("media: uri %q does not match scheme %q", uri, m.scheme)
	}
	key := strings.TrimPrefix(uri, prefix)
	m.mu.RLock()
	obj, ok := m.objects[key]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("media: object %q not found", key)
	}
	return io.NopCloser(bytes.NewReader(obj.data)), nil
}
