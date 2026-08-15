package sqlite

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type ObjectStore struct{ root string }

func NewObjectStore(root string) (*ObjectStore, error) {
	absolute, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil || strings.TrimSpace(root) == "" {
		return nil, errors.New("Space attachment root is required")
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("create Space attachment root: %w", err)
	}
	return &ObjectStore{root: filepath.Clean(absolute)}, nil
}

func (s *ObjectStore) Put(ctx context.Context, key string, content []byte) error {
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary := path + ".tmp-" + uuid.NewString()
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(temporary)
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if _, err := file.Write(content); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	remove = false
	return nil
}

func (s *ObjectStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.pathFor(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (s *ObjectStore) Quarantine(_ context.Context, key string) (string, error) {
	path, err := s.pathFor(key)
	if err != nil {
		return "", err
	}
	tombstone := key + ".deleting-" + uuid.NewString()
	tombstonePath, err := s.pathFor(tombstone)
	if err != nil {
		return "", err
	}
	if err := os.Rename(path, tombstonePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.Join(err, errors.New("attachment object not found"))
		}
		return "", err
	}
	return tombstone, nil
}

func (s *ObjectStore) Restore(_ context.Context, tombstone, key string) error {
	from, err := s.pathFor(tombstone)
	if err != nil {
		return err
	}
	to, err := s.pathFor(key)
	if err != nil {
		return err
	}
	return os.Rename(from, to)
}

func (s *ObjectStore) Remove(_ context.Context, key string) error {
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *ObjectStore) Reconcile(_ context.Context, referenced []string) error {
	expected := make(map[string]struct{}, len(referenced))
	for _, key := range referenced {
		path, err := s.pathFor(key)
		if err != nil {
			return err
		}
		expected[filepath.Clean(path)] = struct{}{}
	}
	var files []string
	if err := filepath.WalkDir(s.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, filepath.Clean(path))
		}
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		base := filepath.Base(file)
		marker := strings.LastIndex(base, ".deleting-")
		if marker < 0 {
			continue
		}
		original := filepath.Join(filepath.Dir(file), base[:marker])
		if _, retained := expected[filepath.Clean(original)]; !retained {
			continue
		}
		if _, err := os.Stat(original); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect retained Space attachment object: %w", err)
		}
		if err := os.Rename(file, original); err != nil {
			return fmt.Errorf("restore retained Space attachment object: %w", err)
		}
	}
	for path := range expected {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("Space attachment object %q is unavailable: %w", path, err)
		}
	}
	for _, path := range files {
		if _, ok := expected[path]; ok {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove orphan Space attachment object: %w", err)
		}
	}
	return nil
}

func (s *ObjectStore) pathFor(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(key)))
	if clean == "." || filepath.IsAbs(clean) || filepath.VolumeName(clean) != "" || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid Space attachment object key")
	}
	path := filepath.Join(s.root, clean)
	relative, err := filepath.Rel(s.root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("Space attachment object escaped its root")
	}
	return path, nil
}
