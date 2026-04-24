package component

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Component struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Kind        string   `yaml:"kind" json:"kind"` // "component"
	Tags        []string `yaml:"tags" json:"tags"`
	Packages    []string `yaml:"packages" json:"packages"`
	DependsOn   []string `yaml:"depends_on" json:"depends_on"`
	Conflicts   []string `yaml:"conflicts" json:"conflicts"`
}

type Registry struct {
	componentsPath string
}

func New(componentsPath string) *Registry {
	return &Registry{componentsPath: componentsPath}
}

func (r *Registry) List() ([]Component, error) {
	entries, err := os.ReadDir(r.componentsPath)
	if err != nil {
		return nil, err
	}

	var components []Component
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		comp, err := r.Load(entry.Name())
		if err != nil {
			continue
		}
		components = append(components, comp)
	}

	return components, nil
}

func (r *Registry) Load(name string) (Component, error) {
	path := filepath.Join(r.componentsPath, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return Component{}, err
	}

	var comp Component
	if err := yaml.Unmarshal(data, &comp); err != nil {
		return Component{}, fmt.Errorf("parse component %s: %w", name, err)
	}

	comp.Name = strings.TrimSuffix(name, ".yaml")
	return comp, nil
}

func (r *Registry) Get(name string) (Component, error) {
	return r.Load(name + ".yaml")
}

func (r *Registry) Search(query string) []Component {
	all, _ := r.List()
	var results []Component
	lowerQuery := strings.ToLower(query)

	for _, comp := range all {
		if strings.Contains(strings.ToLower(comp.Name), lowerQuery) ||
			strings.Contains(strings.ToLower(comp.Description), lowerQuery) {
			results = append(results, comp)
		}
	}

	return results
}

func (r *Registry) ListByTag(tag string) []Component {
	all, _ := r.List()
	var results []Component

	for _, comp := range all {
		for _, t := range comp.Tags {
			if strings.EqualFold(t, tag) {
				results = append(results, comp)
				break
			}
		}
	}

	return results
}

func (r *Registry) GetDependencies(name string) ([]Component, error) {
	comp, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	var deps []Component
	visited := make(map[string]bool)

	var resolve func(string) error
	resolve = func(c string) error {
		if visited[c] {
			return nil
		}
		visited[c] = true

		cmp, err := r.Get(c)
		if err != nil {
			return err
		}

		for _, dep := range cmp.DependsOn {
			if err := resolve(dep); err != nil {
				return err
			}
		}

		deps = append(deps, cmp)
		return nil
	}

	for _, dep := range comp.DependsOn {
		if err := resolve(dep); err != nil {
			return nil, err
		}
	}

	return deps, nil
}

var DefaultComponents = []Component{
	{
		Name:        "desktop-base",
		Description: "Base desktop environment packages",
		Kind:        "component",
		Tags:        []string{"base", "desktop"},
		Packages:    []string{"xorg", "x11-utils", "x11-xserver-utils"},
	},
	{
		Name:        "media-player",
		Description: "Media playback packages",
		Kind:        "component",
		Tags:        []string{"media", "multimedia"},
		Packages:    []string{"vlc", "ffmpeg", "gstreamer1.0-plugins-base"},
	},
	{
		Name:        "office",
		Description: "Office productivity suite",
		Kind:        "component",
		Tags:        []string{"office", "productivity"},
		Packages:    []string{"libreoffice", "libreoffice-gnome"},
		DependsOn:   []string{"desktop-base"},
	},
	{
		Name:        "development",
		Description: "Development tools and compilers",
		Kind:        "component",
		Tags:        []string{"development", "tools"},
		Packages:    []string{"build-essential", "git", "vim", "curl", "wget"},
	},
	{
		Name:        "docker",
		Description: "Docker container runtime",
		Kind:        "component",
		Tags:        []string{"container", "devops"},
		Packages:    []string{"docker.io", "docker-compose"},
	},
	{
		Name:        "kubernetes",
		Description: "Kubernetes cluster tools",
		Kind:        "component",
		Tags:        []string{"container", "orchestration"},
		Packages:    []string{"kubectl", "kubeadm", "kubelet"},
		DependsOn:   []string{"docker"},
	},
	{
		Name:        "golang",
		Description: "Go programming language",
		Kind:        "component",
		Tags:        []string{"language", "development"},
		Packages:    []string{"golang-go"},
	},
	{
		Name:        "nodejs",
		Description: "Node.js runtime and npm",
		Kind:        "component",
		Tags:        []string{"language", "development", "javascript"},
		Packages:    []string{"nodejs", "npm"},
	},
	{
		Name:        "python",
		Description: "Python programming language",
		Kind:        "component",
		Tags:        []string{"language", "development", "python"},
		Packages:    []string{"python3", "python3-pip", "python3-venv"},
	},
	{
		Name:        "rust",
		Description: "Rust programming language",
		Kind:        "component",
		Tags:        []string{"language", "development"},
		Packages:    []string{"cargo", "rustc"},
	},
	{
		Name:        "fonts",
		Description: "Common fonts for desktop",
		Kind:        "component",
		Tags:        []string{"fonts", "desktop"},
		Packages:    []string{"fonts-dejavu", "fonts-liberation", "fonts-noto"},
	},
	{
		Name:        "printer",
		Description: "Printer support and CUPS",
		Kind:        "component",
		Tags:        []string{"hardware", "printing"},
		Packages:    []string{"cups", "cups-client", "printer-driver-brlaser"},
	},
}
