package registry

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSourceLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registries.yaml")
	source := Source{Name: "test", URL: "https://example.com/index.yaml", Enabled: true, Priority: 10}
	if err := AddSource(path, source, false); err != nil {
		t.Fatal(err)
	}
	if err := AddSource(path, source, false); err == nil {
		t.Fatal("expected duplicate error")
	}
	if err := SetSourceEnabled(path, "test", false); err != nil {
		t.Fatal(err)
	}
	sources, err := LoadSources(path)
	if err != nil || len(sources) != 1 || sources[0].Enabled {
		t.Fatalf("unexpected sources: %#v, %v", sources, err)
	}
	if err := RemoveSource(path, "test"); err != nil {
		t.Fatal(err)
	}
}

func TestFetchIndexAndDownload(t *testing.T) {
	recipeBody := []byte("name: demo\ndescription: test\n")
	hash := fmt.Sprintf("%x", sha256.Sum256(recipeBody))
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/index.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		fmt.Fprintf(w, "apiVersion: composer.debian.org/v1alpha1\nkind: RegistryIndex\nmetadata:\n  name: test\nrecipes:\n  - name: demo\n    version: 1.0.0\n    description: Demo\n    download_url: /demo.yaml\n    sha256: %s\n", hash)
	})
	mux.HandleFunc("/demo.yaml", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(recipeBody) })
	client := NewClient(2 * time.Second)
	client.AllowHTTP = true
	index, metadata, err := client.FetchIndex(context.Background(), Source{Name: "test", URL: server.URL + "/index.yaml"}, CacheMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ETag == "" || len(index.Recipes) != 1 {
		t.Fatalf("bad result: %#v %#v", index, metadata)
	}
	destination := filepath.Join(t.TempDir(), "demo.yaml")
	if err := client.DownloadRecipe(context.Background(), index.Recipes[0], destination, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != string(recipeBody) {
		t.Fatalf("bad download: %q %v", data, err)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	cache := Cache{Dir: t.TempDir()}
	index := &RemoteIndex{APIVersion: "v1", Kind: "RegistryIndex", Recipes: []RemoteRecipe{{Name: "demo"}}}
	metadata := CacheMetadata{ETag: "v1", FetchedAt: time.Now().UTC()}
	if err := cache.Write("test", index, metadata); err != nil {
		t.Fatal(err)
	}
	loaded, loadedMeta, err := cache.Read("test")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Recipes[0].Name != "demo" || loadedMeta.ETag != "v1" {
		t.Fatal("cache mismatch")
	}
}
