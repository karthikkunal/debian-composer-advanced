package types

import "time"

// Recipe represents a YAML recipe file
type Recipe struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version"`
	Description string   `yaml:"description" json:"description"`
	Author      string   `yaml:"author" json:"author"`
	License     string   `yaml:"license" json:"license"`
	Tags        []string `yaml:"tags" json:"tags"`
	Kind        string   `yaml:"kind" json:"kind"` // recipe, distro, blend, persona

	// Composition
	Includes []string `yaml:"includes" json:"includes"`
	Extends  string   `yaml:"extends" json:"extends"`
	Layers   []string `yaml:"layers" json:"layers"`

	// Variables
	Variables map[string]Variable `yaml:"variables" json:"variables"`

	// Packages
	Packages []string `yaml:"packages" json:"packages"`

	// Categories with conditional installation
	Categories map[string]Category `yaml:"categories" json:"categories"`

	// Stacks (pre-configured combinations)
	Stacks map[string]Stack `yaml:"stacks" json:"stacks"`

	// Requirements
	Requirements Requirements `yaml:"requirements" json:"requirements"`

	// Dependencies and Conflicts
	Dependencies []string `yaml:"dependencies" json:"dependencies"` // other recipes this depends on
	Conflicts    []string `yaml:"conflicts" json:"conflicts"`       // recipes that cannot be installed together

	// Hooks
	Install   InstallHook   `yaml:"install" json:"install"`
	Configure ConfigureHook `yaml:"configure" json:"configure"`
	Verify    VerifyHook    `yaml:"verify" json:"verify"`

	// Post-install messages
	PostInstall []string `yaml:"post_install" json:"post_install"`

	// Metadata
	FilePath string    `yaml:"-" json:"file_path"`
	LoadedAt time.Time `yaml:"-" json:"loaded_at"`
}

// Variable defines a template variable
type Variable struct {
	Type        string   `yaml:"type" json:"type"`
	Default     string   `yaml:"default" json:"default"`
	Choices     []string `yaml:"choices" json:"choices"`
	Description string   `yaml:"description" json:"description"`
	Required    bool     `yaml:"required" json:"required"`
}

// Category represents an installable group of packages
type Category struct {
	Condition   string   `yaml:"condition" json:"condition"`
	Description string   `yaml:"description" json:"description"`
	Packages    []string `yaml:"install.packages" json:"packages"`
	Scripts     []string `yaml:"install.scripts" json:"scripts"`
	Verify      []string `yaml:"verify.commands" json:"verify_commands"`
}

// Stack is a pre-configured combination of categories
type Stack struct {
	Description string   `yaml:"description" json:"description"`
	Categories  []string `yaml:"categories" json:"categories"`
}

// Requirements defines system requirements
type Requirements struct {
	MinRAM    int    `yaml:"min_ram" json:"min_ram"`   // MB
	MinDisk   int    `yaml:"min_disk" json:"min_disk"` // GB
	MinCPU    int    `yaml:"min_cpu" json:"min_cpu"`   // cores
	GPU       bool   `yaml:"gpu" json:"gpu"`
	DebianVer string `yaml:"debian_version" json:"debian_version"`
}

// InstallHook contains pre/post install commands
type InstallHook struct {
	Pre  []string `yaml:"pre" json:"pre"`
	Post []string `yaml:"post" json:"post"`
}

// ConfigureHook contains configuration commands
type ConfigureHook struct {
	UserGroups []string `yaml:"user_groups" json:"user_groups"`
	Commands   []string `yaml:"commands" json:"commands"`
}

// VerifyHook contains verification commands
type VerifyHook struct {
	Commands []string `yaml:"commands" json:"commands"`
}

// Package represents a Debian package with metadata
type Package struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Size         int64    `json:"size"`
	Section      string   `json:"section"`
}

// InstalledRecipe tracks an installed recipe
type InstalledRecipe struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Installed time.Time `json:"installed_at"`
	Packages  []string  `json:"packages"`
}

// Blend represents a Debian Pure Blend
type Blend struct {
	Name        string   `yaml:"name" json:"name"`
	Kind        string   `yaml:"kind" json:"kind"` // "pure blend"
	TaskPackage string   `json:"task_package"`
	Packages    []string `yaml:"packages" json:"packages"`
}

// ResolvedRecipe is a recipe with all includes/extends resolved
type ResolvedRecipe struct {
	Recipe
	AllPackages  []string          `json:"all_packages"`
	ResolvedVars map[string]string `json:"resolved_vars"`
	ActiveCats   []string          `json:"active_categories"`
}

// GetAllPackages returns the resolved package list
func (r *ResolvedRecipe) GetAllPackages() []string {
	return r.AllPackages
}
