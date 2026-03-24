package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/debian-composer/debian-composer-go/internal/types"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Manager handles state persistence
type Manager struct {
	db     *sqlx.DB
	dbPath string
}

// New creates a new state manager
func New(stateDir string) (*Manager, error) {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(stateDir, "state.db")
	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	m := &Manager{db: db, dbPath: dbPath}
	if err := m.init(); err != nil {
		db.Close()
		return nil, err
	}

	return m, nil
}

// init creates database tables
func (m *Manager) init() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS recipes (
			name TEXT PRIMARY KEY,
			version TEXT,
			installed_at DATETIME,
			packages TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS blends (
			name TEXT PRIMARY KEY,
			task_package TEXT,
			installed_at DATETIME,
			packages TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS packages (
			name TEXT PRIMARY KEY,
			recipe_name TEXT,
			installed_at DATETIME
		)`,
	}

	for _, q := range queries {
		if _, err := m.db.Exec(q); err != nil {
			return err
		}
	}

	return nil
}

// recipeRow is the raw DB representation of an installed recipe.
type recipeRow struct {
	Name        string    `db:"name"`
	Version     string    `db:"version"`
	InstalledAt time.Time `db:"installed_at"`
	Packages    string    `db:"packages"` // JSON-encoded []string
}

func (r recipeRow) toInstalled() (types.InstalledRecipe, error) {
	out := types.InstalledRecipe{
		Name:      r.Name,
		Version:   r.Version,
		Installed: r.InstalledAt,
	}
	if err := json.Unmarshal([]byte(r.Packages), &out.Packages); err != nil {
		return out, err
	}
	return out, nil
}

// SaveRecipe saves an installed recipe
func (m *Manager) SaveRecipe(recipe *types.ResolvedRecipe) error {
	pkgsJSON, err := json.Marshal(recipe.AllPackages)
	if err != nil {
		return err
	}

	_, err = m.db.Exec(
		`INSERT OR REPLACE INTO recipes (name, version, installed_at, packages) VALUES (?, ?, ?, ?)`,
		recipe.Name, recipe.Version, time.Now(), string(pkgsJSON),
	)
	return err
}

// GetRecipe returns an installed recipe
func (m *Manager) GetRecipe(name string) (*types.InstalledRecipe, error) {
	var row recipeRow
	if err := m.db.Get(&row, `SELECT name, version, installed_at, packages FROM recipes WHERE name = ?`, name); err != nil {
		return nil, err
	}
	r, err := row.toInstalled()
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListRecipes returns all installed recipes
func (m *Manager) ListRecipes() ([]types.InstalledRecipe, error) {
	var rows []recipeRow
	if err := m.db.Select(&rows, `SELECT name, version, installed_at, packages FROM recipes ORDER BY installed_at DESC`); err != nil {
		return nil, err
	}

	recipes := make([]types.InstalledRecipe, 0, len(rows))
	for _, row := range rows {
		r, err := row.toInstalled()
		if err != nil {
			continue
		}
		recipes = append(recipes, r)
	}
	return recipes, nil
}

// DeleteRecipe removes a recipe from state
func (m *Manager) DeleteRecipe(name string) error {
	_, err := m.db.Exec(`DELETE FROM recipes WHERE name = ?`, name)
	return err
}

// SaveBlend saves an installed blend
func (m *Manager) SaveBlend(name, taskPkg string, packages []string) error {
	pkgsJSON, _ := json.Marshal(packages)
	_, err := m.db.Exec(
		`INSERT OR REPLACE INTO blends (name, task_package, installed_at, packages) VALUES (?, ?, ?, ?)`,
		name, taskPkg, time.Now(), string(pkgsJSON),
	)
	return err
}

// IsRecipeInstalled checks if a recipe is installed
func (m *Manager) IsRecipeInstalled(name string) bool {
	var count int
	m.db.Get(&count, `SELECT COUNT(*) FROM recipes WHERE name = ?`, name)
	return count > 0
}

// Close closes the database
func (m *Manager) Close() error {
	return m.db.Close()
}
