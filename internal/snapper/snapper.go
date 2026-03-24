package snapper

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type SnapshotType string

const (
	SnapshotTypePre    SnapshotType = "pre"
	SnapshotTypePost   SnapshotType = "post"
	SnapshotTypeSingle SnapshotType = "single"
	SnapshotTypeEmpty  SnapshotType = "empty"
)

type CleanupAlgorithm string

const (
	CleanupNumber   CleanupAlgorithm = "number"
	CleanupTimeline CleanupAlgorithm = "timeline"
	CleanupEmpty    CleanupAlgorithm = "empty-pre-post"
)

type Snapshot struct {
	ID        int          `json:"num"`
	Name      string       `json:"description"`
	Type      SnapshotType `json:"type"`
	Timestamp time.Time    `json:"date"`
	Subvolume string       `json:"subvolume"`
	UsedSpace int64        `json:"used_space"`
	Immutable bool         `json:"immutable"`
	UserData  string       `json:"userdata"`
	ReadOnly  bool         `json:"read_only"`
}

type Config struct {
	Name                   string `json:"config"`
	Subvolume              string `json:"subvolume"`
	FSType                 string `json:"fstype"`
	ALLOW_USERS            string `json:"ALLOW_USERS"`
	ALLOW_GROUPS           string `json:"ALLOW_GROUPS"`
	BACKGROUND_COMPARISON  string `json:"BACKGROUND_COMPARISON"`
	EMPTY_PRE_POST_CLEANUP string `json:"EMPTY_PRE_POST_CLEANUP"`
	EMPTY_PRE_POST_MIN_AGE string `json:"EMPTY_PRE_POST_MIN_AGE"`
	NUMBER_CLEANUP         string `json:"NUMBER_CLEANUP"`
	NUMBER_MIN_AGE         string `json:"NUMBER_MIN_AGE"`
	NUMBER_LIMIT           string `json:"NUMBER_LIMIT"`
	TIMELINE_CLEANUP       string `json:"TIMELINE_CLEANUP"`
	TIMELINE_MIN_AGE       string `json:"TIMELINE_MIN_AGE"`
	TIMELINE_LIMIT         string `json:"TIMELINE_LIMIT"`
	TIMELINE_LIMIT_DAILY   string `json:"TIMELINE_LIMIT_DAILY"`
	TIMELINE_LIMIT_WEEKLY  string `json:"TIMELINE_LIMIT_WEEKLY"`
	TIMELINE_LIMIT_MONTHLY string `json:"TIMELINE_LIMIT_MONTHLY"`
	TIMELINE_LIMIT_YEARLY  string `json:"TIMELINE_LIMIT_YEARLY"`
	SYNC_ACL               string `json:"SYNC_ACL"`
	FREE_LIMIT             string `json:"FREE_LIMIT"`
	SPACE_LIMIT            string `json:"SPACE_LIMIT"`
}

type DiffEntry struct {
	Path   string `json:"path"`
	Status string `json:"status"` // c=created, d=deleted, m=modified
	Perms  string `json:"perms"`
	Owner  string `json:"owner"`
	Group  string `json:"group"`
}

type Manager struct {
	dryRun bool
	config string
}

func New(dryRun bool, config ...string) *Manager {
	cfg := "root"
	if len(config) > 0 {
		cfg = config[0]
	}
	return &Manager{dryRun: dryRun, config: cfg}
}

func IsAvailable() bool {
	_, err := exec.LookPath("snapper")
	return err == nil
}

func (m *Manager) CreateSnapshot(snapType, description string) (*Snapshot, error) {
	return m.CreateSnapshotWithCleanup(snapType, description, "")
}

func (m *Manager) CreateSnapshotWithCleanup(snapType, description string, cleanup CleanupAlgorithm) (*Snapshot, error) {
	if m.dryRun {
		return &Snapshot{
			ID:   -1,
			Name: "dry-run-snapshot",
			Type: SnapshotType(snapType),
		}, nil
	}

	args := []string{"-c", m.config, "create"}
	if snapType != "" {
		args = append(args, "--type", snapType)
	}
	if description != "" {
		args = append(args, "--description", description)
	}
	if cleanup != "" {
		args = append(args, "--cleanup", string(cleanup))
	}
	args = append(args, "--print-number")

	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w\nOutput: %s", err, out)
	}

	id, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return &Snapshot{
		ID:   id,
		Name: description,
		Type: SnapshotType(snapType),
	}, nil
}

func (m *Manager) CreatePreSnapshot(description string) (int, error) {
	if m.dryRun {
		return -1, nil
	}

	args := []string{"-c", m.config, "create", "--type", "pre", "--description", description, "--print-number"}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to create pre snapshot: %w\nOutput: %s", err, out)
	}

	id, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return id, nil
}

func (m *Manager) CreatePostSnapshot(preID int, description string) (*Snapshot, error) {
	if m.dryRun {
		return &Snapshot{
			ID:   -1,
			Name: "dry-run-post-snapshot",
			Type: SnapshotTypePost,
		}, nil
	}

	args := []string{"-c", m.config, "create", "--type", "post", "--description", description, "--pre-number", strconv.Itoa(preID)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create post snapshot: %w\nOutput: %s", err, out)
	}

	id, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return &Snapshot{
		ID:   id,
		Name: description,
		Type: SnapshotTypePost,
	}, nil
}

func (m *Manager) CreateCommandSnapshot(description, command string) (*Snapshot, error) {
	if m.dryRun {
		return &Snapshot{
			ID:   -1,
			Name: "dry-run-cmd-snapshot",
			Type: SnapshotTypeSingle,
		}, nil
	}

	args := []string{"-c", m.config, "create", "--description", description, "--command", command}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create command snapshot: %w\nOutput: %s", err, out)
	}

	id, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return &Snapshot{
		ID:   id,
		Name: description,
		Type: SnapshotTypeSingle,
	}, nil
}

func (m *Manager) RestoreSnapshot(id int) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "rollback", strconv.Itoa(id)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restore snapshot %d: %w\nOutput: %s", id, err, out)
	}

	return nil
}

func (m *Manager) DeleteSnapshot(id int) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "delete", strconv.Itoa(id)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete snapshot %d: %w\nOutput: %s", id, err, out)
	}

	return nil
}

func (m *Manager) DeleteSnapshots(ids []int) error {
	if m.dryRun {
		return nil
	}

	idStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ids)), " "), "[]")
	args := []string{"-c", m.config, "delete", idStr}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete snapshots %v: %w\nOutput: %s", ids, err, out)
	}

	return nil
}

func (m *Manager) ListSnapshots() ([]Snapshot, error) {
	args := []string{"-c", m.config, "list", "--json"}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.Output()
	if err != nil {
		return m.listSnapshotsText()
	}

	var result struct {
		Snapshots []Snapshot `json:"snapshots"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return m.listSnapshotsText()
	}

	return result.Snapshots, nil
}

func (m *Manager) listSnapshotsText() ([]Snapshot, error) {
	args := []string{"-c", m.config, "list"}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []Snapshot
	lines := strings.Split(string(out), "\n")

	for i, line := range lines {
		if i < 2 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		id, _ := strconv.Atoi(fields[0])
		snap := Snapshot{
			ID:   id,
			Name: strings.Join(fields[3:], " "),
			Type: SnapshotType(fields[1]),
		}

		tsStr := fields[2]
		if ts, err := time.Parse("2006-01-02 00:00:00", tsStr); err == nil {
			snap.Timestamp = ts
		}

		snapshots = append(snapshots, snap)
	}

	return snapshots, nil
}

func (m *Manager) GetSnapshot(id int) (*Snapshot, error) {
	args := []string{"-c", m.config, "list", "--json", "--snapshot", strconv.Itoa(id)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot %d: %w", id, err)
	}

	var result struct {
		Snapshots []Snapshot `json:"snapshots"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot: %w", err)
	}

	if len(result.Snapshots) == 0 {
		return nil, fmt.Errorf("snapshot %d not found", id)
	}

	return &result.Snapshots[0], nil
}

func (m *Manager) GetConfig() (*Config, error) {
	args := []string{"-c", m.config, "get-config", "--json"}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(out, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func (m *Manager) CreateConfig(subvolume, fstype string) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "create-config", "-f", fstype, subvolume}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create config: %w\nOutput: %s", err, out)
	}

	return nil
}

func (m *Manager) SetConfig(key, value string) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "set-config", key + "=" + value}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set config %s: %w\nOutput: %s", key, err, out)
	}

	return nil
}

func (m *Manager) EnableTimelineSnapshots() error {
	if m.dryRun {
		return nil
	}

	if err := m.SetConfig("TIMELINE_CREATE", "yes"); err != nil {
		return err
	}
	if err := m.SetConfig("TIMELINE_CLEANUP", "yes"); err != nil {
		return err
	}
	return nil
}

func (m *Manager) SetTimelineLimits(daily, weekly, monthly, yearly int) error {
	if m.dryRun {
		return nil
	}

	if err := m.SetConfig("TIMELINE_LIMIT_DAILY", strconv.Itoa(daily)); err != nil {
		return err
	}
	if err := m.SetConfig("TIMELINE_LIMIT_WEEKLY", strconv.Itoa(weekly)); err != nil {
		return err
	}
	if err := m.SetConfig("TIMELINE_LIMIT_MONTHLY", strconv.Itoa(monthly)); err != nil {
		return err
	}
	if err := m.SetConfig("TIMELINE_LIMIT_YEARLY", strconv.Itoa(yearly)); err != nil {
		return err
	}
	return nil
}

func (m *Manager) SetNumberLimits(limit int, minAge string) error {
	if m.dryRun {
		return nil
	}

	if err := m.SetConfig("NUMBER_CLEANUP", "yes"); err != nil {
		return err
	}
	if err := m.SetConfig("NUMBER_LIMIT", strconv.Itoa(limit)); err != nil {
		return err
	}
	if err := m.SetConfig("NUMBER_MIN_AGE", minAge); err != nil {
		return err
	}
	return nil
}

func (m *Manager) SetupQuota() error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "setup-quota"}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to setup quota: %w\nOutput: %s", err, out)
	}

	return nil
}

func (m *Manager) RunCleanup(algorithm CleanupAlgorithm) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "cleanup", string(algorithm)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run cleanup %s: %w\nOutput: %s", algorithm, err, out)
	}

	return nil
}

func (m *Manager) CompareSnapshots(id1, id2 int) ([]DiffEntry, error) {
	args := []string{"-c", m.config, "status", strconv.Itoa(id1) + ".." + strconv.Itoa(id2)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to compare snapshots: %w", err)
	}

	return parseDiffOutput(string(out)), nil
}

func (m *Manager) CompareToCurrent(id int) ([]DiffEntry, error) {
	args := []string{"-c", m.config, "status", strconv.Itoa(id)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to compare snapshot to current: %w", err)
	}

	return parseDiffOutput(string(out)), nil
}

func parseDiffOutput(output string) []DiffEntry {
	var diffs []DiffEntry
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "Modifying") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		status := fields[0]
		path := fields[1]

		entry := DiffEntry{
			Path:   path,
			Status: status,
		}

		if len(fields) >= 5 {
			entry.Perms = fields[2]
			entry.Owner = fields[3]
			entry.Group = fields[4]
		}

		diffs = append(diffs, entry)
	}

	return diffs
}

func (m *Manager) UndoChanges(id1, id2 int) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "undo", strconv.Itoa(id1) + ".." + strconv.Itoa(id2)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to undo changes: %w\nOutput: %s", err, out)
	}

	return nil
}

func (m *Manager) MountSnapshot(id int) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "mount", strconv.Itoa(id)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mount snapshot %d: %w\nOutput: %s", id, err, out)
	}

	return nil
}

func (m *Manager) UmountSnapshot(id int) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "umount", strconv.Itoa(id)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unmount snapshot %d: %w\nOutput: %s", id, err, out)
	}

	return nil
}

func (m *Manager) SetDefaultSnapshot(id int) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "set-default", strconv.Itoa(id)}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set default snapshot %d: %w\nOutput: %s", id, err, out)
	}

	return nil
}

func (m *Manager) ModifySnapshot(id int, description, userData string) error {
	if m.dryRun {
		return nil
	}

	args := []string{"-c", m.config, "modify", strconv.Itoa(id)}
	if description != "" {
		args = append(args, "--description", description)
	}
	if userData != "" {
		args = append(args, "--userdata", userData)
	}

	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to modify snapshot %d: %w\nOutput: %s", id, err, out)
	}

	return nil
}

func (m *Manager) CreateSnapshotForRecipe(recipeName string) (*Snapshot, error) {
	description := fmt.Sprintf("Before installing recipe: %s", recipeName)
	return m.CreateSnapshotWithCleanup(string(SnapshotTypeSingle), description, CleanupNumber)
}

func (m *Manager) RollbackRecipe() error {
	snapshots, err := m.ListSnapshots()
	if err != nil {
		return err
	}

	var latest *Snapshot
	for i := range snapshots {
		snap := &snapshots[i]
		if strings.Contains(snap.Name, "recipe") {
			if latest == nil || snap.Timestamp.After(latest.Timestamp) {
				latest = snap
			}
		}
	}

	if latest == nil {
		return fmt.Errorf("no recipe snapshots found")
	}

	return m.RestoreSnapshot(latest.ID)
}

func (m *Manager) CreateAptPreSnapshot() (int, error) {
	return m.CreatePreSnapshot("apt pre")
}

func (m *Manager) CreateAptPostSnapshot() (*Snapshot, error) {
	return m.CreateSnapshotWithCleanup(string(SnapshotTypePost), "apt post", CleanupNumber)
}

func CheckAvailable() (bool, string) {
	if IsAvailable() {
		return true, "Snapper is available"
	}
	return false, "Snapper is not installed. Install with: sudo apt install snapper"
}

func GetConfigs() ([]Config, error) {
	cmd := exec.Command("sudo", "snapper", "list-configs", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	var result struct {
		Configs []Config `json:"configs"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to parse configs: %w", err)
	}

	return result.Configs, nil
}

type MountedSnapshot struct {
	SnapshotID int
	MountPoint string
}

func (m *Manager) ListMountedSnapshots() ([]MountedSnapshot, error) {
	args := []string{"-c", m.config, "list", "--mount"}
	cmd := exec.Command("sudo", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list mounted snapshots: %w", err)
	}

	var mounted []MountedSnapshot
	lines := strings.Split(string(out), "\n")

	for i, line := range lines {
		if i < 2 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 4 {
			id, _ := strconv.Atoi(fields[0])
			mounted = append(mounted, MountedSnapshot{
				SnapshotID: id,
				MountPoint: fields[3],
			})
		}
	}

	return mounted, nil
}
