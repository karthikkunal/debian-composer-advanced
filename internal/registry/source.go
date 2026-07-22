package registry

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadSources(path string) ([]Source, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Source{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read registry sources: %w", err)
	}
	var file SourcesFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse registry sources: %w", err)
	}
	sortSources(file.Registries)
	return file.Registries, nil
}

func SaveSources(path string, sources []Source) error {
	for _, source := range sources {
		if err := ValidateSource(source, false); err != nil {
			return err
		}
	}
	sortSources(sources)
	data, err := yaml.Marshal(SourcesFile{Registries: sources})
	if err != nil {
		return fmt.Errorf("marshal registry sources: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create registry config directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".registries-*.yaml")
	if err != nil {
		return fmt.Errorf("create temporary registry config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write registry config: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fmt.Errorf("set registry config permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close registry config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace registry config: %w", err)
	}
	return nil
}

func ValidateSource(source Source, allowHTTP bool) error {
	if strings.TrimSpace(source.Name) == "" {
		return errors.New("registry source name is required")
	}
	parsed, err := url.ParseRequestURI(source.URL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid registry URL %q", source.URL)
	}
	if parsed.Scheme != "https" && !(allowHTTP && parsed.Scheme == "http") {
		return fmt.Errorf("%w: %s", ErrInsecureURL, source.URL)
	}
	return nil
}

func AddSource(path string, source Source, allowHTTP bool) error {
	if err := ValidateSource(source, allowHTTP); err != nil {
		return err
	}
	sources, err := LoadSources(path)
	if err != nil {
		return err
	}
	for _, existing := range sources {
		if strings.EqualFold(existing.Name, source.Name) {
			return fmt.Errorf("%w: %s", ErrDuplicateSource, source.Name)
		}
	}
	sources = append(sources, source)
	return SaveSourcesAllowHTTP(path, sources, allowHTTP)
}

func SaveSourcesAllowHTTP(path string, sources []Source, allowHTTP bool) error {
	for _, source := range sources {
		if err := ValidateSource(source, allowHTTP); err != nil {
			return err
		}
	}
	sortSources(sources)
	data, err := yaml.Marshal(SourcesFile{Registries: sources})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func RemoveSource(path, name string) error {
	sources, err := LoadSources(path)
	if err != nil {
		return err
	}
	filtered := sources[:0]
	found := false
	for _, source := range sources {
		if strings.EqualFold(source.Name, name) {
			found = true
			continue
		}
		filtered = append(filtered, source)
	}
	if !found {
		return fmt.Errorf("%w: %s", ErrSourceNotFound, name)
	}
	return SaveSources(path, filtered)
}

func SetSourceEnabled(path, name string, enabled bool) error {
	sources, err := LoadSources(path)
	if err != nil {
		return err
	}
	found := false
	for i := range sources {
		if strings.EqualFold(sources[i].Name, name) {
			sources[i].Enabled = enabled
			found = true
		}
	}
	if !found {
		return fmt.Errorf("%w: %s", ErrSourceNotFound, name)
	}
	return SaveSources(path, sources)
}

func FindSource(sources []Source, name string) (Source, error) {
	for _, source := range sources {
		if strings.EqualFold(source.Name, name) {
			return source, nil
		}
	}
	return Source{}, fmt.Errorf("%w: %s", ErrSourceNotFound, name)
}

func sortSources(sources []Source) {
	sort.SliceStable(sources, func(i, j int) bool {
		if sources[i].Priority == sources[j].Priority {
			return sources[i].Name < sources[j].Name
		}
		return sources[i].Priority > sources[j].Priority
	})
}
