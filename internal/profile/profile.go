package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Kind        string            `json:"kind"`
	Variables   map[string]string `json:"variables"`
	Mixins      []string          `json:"mixins"`
	Stack       string            `json:"stack"`
}

type Manager struct {
	profilesPath string
}

func New(profilesPath string) *Manager {
	return &Manager{profilesPath: profilesPath}
}

func (m *Manager) List() ([]Profile, error) {
	entries, err := os.ReadDir(m.profilesPath)
	if err != nil {
		return nil, err
	}

	var profiles []Profile
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		profile, err := m.Load(entry.Name())
		if err != nil {
			continue
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func (m *Manager) Load(name string) (Profile, error) {
	path := filepath.Join(m.profilesPath, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}

	var profile Profile
	if err := parseProfile(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("parse profile %s: %w", name, err)
	}

	profile.Name = name
	return profile, nil
}

func parseProfile(data []byte, p *Profile) error {
	if err := yaml.Unmarshal(data, p); err != nil {
		return err
	}
	if p.Variables == nil {
		p.Variables = make(map[string]string)
	}
	return nil
}

func (m *Manager) Save(profile *Profile) error {
	path := filepath.Join(m.profilesPath, profile.Name+".yaml")
	data, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *Manager) Delete(name string) error {
	path := filepath.Join(m.profilesPath, name+".yaml")
	return os.Remove(path)
}

func (m *Manager) Exists(name string) bool {
	path := filepath.Join(m.profilesPath, name+".yaml")
	_, err := os.Stat(path)
	return err == nil
}

var DefaultProfiles = []Profile{
	{
		Name:        "laptop",
		Description: "Laptop optimized profile with power management",
		Kind:        "profile",
		Variables: map[string]string{
			"power_management":  "true",
			"screen_brightness": "auto",
			"touchpad":          "enabled",
		},
	},
	{
		Name:        "desktop",
		Description: "Desktop optimized profile with full performance",
		Kind:        "profile",
		Variables: map[string]string{
			"power_management":  "performance",
			"screen_brightness": "100",
			"touchpad":          "disabled",
		},
	},
	{
		Name:        "server",
		Description: "Server profile without GUI",
		Kind:        "profile",
		Variables: map[string]string{
			"install_gui": "false",
			"ssh_server":  "true",
			"docker":      "true",
		},
	},
	{
		Name:        "development",
		Description: "Development environment with full tooling",
		Kind:        "profile",
		Variables: map[string]string{
			"install_gui": "true",
			"dev_tools":   "true",
			"docker":      "true",
			"vscode":      "true",
		},
		Mixins: []string{"go", "nodejs", "python"},
	},
}
