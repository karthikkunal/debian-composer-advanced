package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Client struct {
	HTTPClient *http.Client
	UserAgent  string
	MaxSize    int64
	AllowHTTP  bool
}

func NewClient(timeout time.Duration) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	return &Client{
		HTTPClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					return errors.New("cross-host redirect rejected")
				}
				return nil
			},
		},
		UserAgent: "debian-composer/remote-registry",
		MaxSize:   DefaultMaxIndexSize,
	}
}

func (c *Client) FetchIndex(ctx context.Context, source Source, cached CacheMetadata) (*RemoteIndex, CacheMetadata, error) {
	if err := ValidateSource(source, c.AllowHTTP); err != nil {
		return nil, CacheMetadata{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, CacheMetadata{}, fmt.Errorf("create registry request: %w", err)
	}
	req.Header.Set("Accept", "application/yaml, application/x-yaml, text/yaml, application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	if cached.ETag != "" {
		req.Header.Set("If-None-Match", cached.ETag)
	}
	if cached.LastModified != "" {
		req.Header.Set("If-Modified-Since", cached.LastModified)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, CacheMetadata{}, fmt.Errorf("fetch registry index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, cached, ErrNotModified
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, CacheMetadata{}, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}
	maxSize := c.MaxSize
	if maxSize <= 0 {
		maxSize = DefaultMaxIndexSize
	}
	limited := io.LimitReader(resp.Body, maxSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, CacheMetadata{}, fmt.Errorf("read registry response: %w", err)
	}
	if int64(len(data)) > maxSize {
		return nil, CacheMetadata{}, ErrResponseTooLarge
	}
	var index RemoteIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, CacheMetadata{}, fmt.Errorf("parse registry index: %w", err)
	}
	if index.Kind != "RegistryIndex" || len(index.Recipes) == 0 {
		return nil, CacheMetadata{}, errors.New("invalid or empty remote registry index")
	}
	baseURL, _ := url.Parse(source.URL)
	for i := range index.Recipes {
		ref, err := url.Parse(strings.TrimSpace(index.Recipes[i].DownloadURL))
		if err != nil {
			return nil, CacheMetadata{}, fmt.Errorf("invalid recipe URL for %s: %w", index.Recipes[i].Name, err)
		}
		index.Recipes[i].DownloadURL = baseURL.ResolveReference(ref).String()
	}
	metadata := CacheMetadata{
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		FetchedAt:    time.Now().UTC(),
	}
	return &index, metadata, nil
}
