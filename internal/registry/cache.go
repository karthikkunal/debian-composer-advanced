package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

var safeCacheName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type Cache struct{ Dir string }

func (c Cache) paths(sourceName string) (string, string) {
	name := safeCacheName.ReplaceAllString(sourceName, "_")
	return filepath.Join(c.Dir, name+".index.yaml"), filepath.Join(c.Dir, name+".meta.yaml")
}

func (c Cache) Write(sourceName string, index *RemoteIndex, metadata CacheMetadata) error {
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return fmt.Errorf("create registry cache: %w", err)
	}
	indexData, err := yaml.Marshal(index)
	if err != nil {
		return err
	}
	metaData, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}
	indexPath, metaPath := c.paths(sourceName)
	if err := atomicWrite(indexPath, indexData, 0o644); err != nil {
		return err
	}
	return atomicWrite(metaPath, metaData, 0o644)
}

func (c Cache) Read(sourceName string) (*RemoteIndex, CacheMetadata, error) {
	indexPath, metaPath := c.paths(sourceName)
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, CacheMetadata{}, err
	}
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, CacheMetadata{}, err
	}
	var index RemoteIndex
	var metadata CacheMetadata
	if err := yaml.Unmarshal(indexData, &index); err != nil {
		return nil, CacheMetadata{}, err
	}
	if err := yaml.Unmarshal(metaData, &metadata); err != nil {
		return nil, CacheMetadata{}, err
	}
	return &index, metadata, nil
}

func (c Cache) Metadata(sourceName string) CacheMetadata {
	_, metadata, err := c.Read(sourceName)
	if err != nil {
		return CacheMetadata{}
	}
	return metadata
}

func (c Cache) IsFresh(sourceName string, ttl time.Duration) bool {
	metadata := c.Metadata(sourceName)
	return !metadata.FetchedAt.IsZero() && time.Since(metadata.FetchedAt) <= ttl
}

func (c Cache) Clear(sourceName string) error {
	indexPath, metaPath := c.paths(sourceName)
	for _, path := range []string{indexPath, metaPath} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (c Cache) ClearAll() error {
	if err := os.RemoveAll(c.Dir); err != nil {
		return err
	}
	return os.MkdirAll(c.Dir, 0o755)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
