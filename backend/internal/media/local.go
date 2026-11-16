package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type LocalStore struct {
	root string
}

func NewLocalStore(root string) (*LocalStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("media: upload directory is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("media: resolve upload directory: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("media: create upload directory: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("media: stat upload directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("media: upload path is not a directory")
	}
	probe, err := os.CreateTemp(abs, ".wms-write-test-*")
	if err != nil {
		return nil, fmt.Errorf("media: upload directory is not writable: %w", err)
	}
	probeName := probe.Name()
	_ = probe.Close()
	_ = os.Remove(probeName)
	return &LocalStore{root: abs}, nil
}

func (s *LocalStore) Put(ctx context.Context, key string, r io.Reader) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	dst, err := s.resolve(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".wms-upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()
	if err = tmp.Chmod(0o640); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err = io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpName, dst); err != nil {
		return err
	}
	return nil
}

func (s *LocalStore) Open(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dst, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(dst)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *LocalStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dst, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *LocalStore) resolve(key string) (string, error) {
	if !validKey(key) {
		return "", ErrInvalidKey
	}
	dst := filepath.Join(s.root, filepath.FromSlash(key))
	rel, err := filepath.Rel(s.root, dst)
	if err != nil {
		return "", ErrInvalidKey
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", ErrInvalidKey
	}
	return dst, nil
}

func validKey(key string) bool {
	if key == "" || strings.ContainsRune(key, '\x00') || strings.Contains(key, `\`) || strings.HasPrefix(key, "/") {
		return false
	}
	clean := path.Clean(key)
	if clean != key || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	return true
}
