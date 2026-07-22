package registry

import "time"

const (
	DefaultMaxIndexSize  int64 = 10 << 20
	DefaultMaxRecipeSize int64 = 5 << 20
)

type Source struct {
	Name     string `yaml:"name" json:"name"`
	URL      string `yaml:"url" json:"url"`
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Priority int    `yaml:"priority" json:"priority"`
}

type SourcesFile struct {
	Registries []Source `yaml:"registries" json:"registries"`
}

type RemoteRecipe struct {
	Name          string   `yaml:"name" json:"name"`
	Version       string   `yaml:"version" json:"version"`
	Description   string   `yaml:"description" json:"description"`
	DownloadURL   string   `yaml:"download_url" json:"download_url"`
	SHA256        string   `yaml:"sha256" json:"sha256"`
	Tags          []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Architectures []string `yaml:"architectures,omitempty" json:"architectures,omitempty"`
	Distributions []string `yaml:"distributions,omitempty" json:"distributions,omitempty"`
}

type RemoteIndexMetadata struct {
	Name      string    `yaml:"name" json:"name"`
	UpdatedAt time.Time `yaml:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type RemoteIndex struct {
	APIVersion string              `yaml:"apiVersion" json:"apiVersion"`
	Kind       string              `yaml:"kind" json:"kind"`
	Metadata   RemoteIndexMetadata `yaml:"metadata" json:"metadata"`
	Recipes    []RemoteRecipe      `yaml:"recipes" json:"recipes"`
}

type CacheMetadata struct {
	ETag         string    `yaml:"etag,omitempty" json:"etag,omitempty"`
	LastModified string    `yaml:"last_modified,omitempty" json:"last_modified,omitempty"`
	FetchedAt    time.Time `yaml:"fetched_at" json:"fetched_at"`
}
