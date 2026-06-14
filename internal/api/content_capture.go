package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/agentsh/agentsh/internal/platform"
)

type ContentStore struct {
	dir     string
	maxSize int64

	mu    sync.Mutex
	sizes map[string]int64 // path -> total captured bytes
}

func NewContentStore(dir string, maxSize int64) *ContentStore {
	return &ContentStore{
		dir:     dir,
		maxSize: maxSize,
		sizes:   make(map[string]int64),
	}
}

func (s *ContentStore) Init() error {
	return os.MkdirAll(s.dir, 0700)
}

func (s *ContentStore) Close() error {
	return os.RemoveAll(s.dir)
}

func (s *ContentStore) Capture(path string, op platform.FileOperation, content []byte, offset int64) (contentID string, truncated bool, err error) {
	if len(content) == 0 {
		return "", false, nil
	}

	s.mu.Lock()
	total := s.sizes[path]
	if total >= s.maxSize {
		s.mu.Unlock()
		return "", true, nil
	}
	s.mu.Unlock()

	hash := sha256.Sum256(content)
	id := hex.EncodeToString(hash[:]) + "_" + string(op)

	dst := filepath.Join(s.dir, id)
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return "", false, fmt.Errorf("content store: %w", err)
	}
	defer f.Close()

	written, err := f.Write(content)
	if err != nil {
		return "", false, fmt.Errorf("content store write: %w", err)
	}

	s.mu.Lock()
	s.sizes[path] = total + int64(written)
	overLimit := s.sizes[path] > s.maxSize
	s.mu.Unlock()

	return id, overLimit, nil
}
