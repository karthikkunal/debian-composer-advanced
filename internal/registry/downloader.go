package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func (c *Client) DownloadRecipe(ctx context.Context, recipe RemoteRecipe, destination string, force bool) error {
	parsed, err := url.Parse(recipe.DownloadURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid recipe download URL %q", recipe.DownloadURL)
	}
	if parsed.Scheme != "https" && !(c.AllowHTTP && parsed.Scheme == "http") {
		return fmt.Errorf("%w: %s", ErrInsecureURL, recipe.DownloadURL)
	}
	cleanDestination := filepath.Clean(destination)
	if strings.Contains(filepath.Base(cleanDestination), "..") {
		return errors.New("invalid destination path")
	}
	if !force {
		if _, err := os.Stat(cleanDestination); err == nil {
			return fmt.Errorf("destination already exists: %s", cleanDestination)
		}
	}
	if err := os.MkdirAll(filepath.Dir(cleanDestination), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, recipe.DownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("download recipe: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("recipe download returned HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp(filepath.Dir(cleanDestination), ".recipe-*.yaml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	maxSize := DefaultMaxRecipeSize
	written, err := io.Copy(tmp, io.LimitReader(resp.Body, maxSize+1))
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if written > maxSize {
		return ErrResponseTooLarge
	}
	if err := VerifySHA256(tmpName, recipe.SHA256); err != nil {
		return err
	}
	if force {
		_ = os.Remove(cleanDestination)
	}
	return os.Rename(tmpName, cleanDestination)
}

func SearchRemote(indexes map[string]*RemoteIndex, term string) []RemoteRecipe {
	term = strings.ToLower(strings.TrimSpace(term))
	var results []RemoteRecipe
	for _, index := range indexes {
		for _, recipe := range index.Recipes {
			haystack := strings.ToLower(recipe.Name + " " + recipe.Description + " " + strings.Join(recipe.Tags, " "))
			if term == "" || strings.Contains(haystack, term) {
				results = append(results, recipe)
			}
		}
	}
	return results
}
