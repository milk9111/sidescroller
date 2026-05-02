package savegame

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const CurrentVersion = 1

type File struct {
	Version           int               `json:"version"`
	Level             string            `json:"level"`
	Player            PlayerState       `json:"player"`
	LevelLayerStates  map[string]bool   `json:"levelLayerStates,omitempty"`
	LevelEntityStates map[string]string `json:"levelEntityStates,omitempty"`
	SavedAt           time.Time         `json:"savedAt"`
}

type PlayerState struct {
	Health             HealthState              `json:"health"`
	Abilities          AbilitiesState           `json:"abilities"`
	Inventory          []InventoryItem          `json:"inventory,omitempty"`
	GearCount          int                      `json:"gearCount"`
	HealUses           int                      `json:"healUses"`
	Transform          TransformState           `json:"transform"`
	SafeRespawn        SafeRespawnState         `json:"safeRespawn"`
	Checkpoint         CheckpointState          `json:"checkpoint"`
	FacingLeft         bool                     `json:"facingLeft"`
	TransitionCooldown *TransitionCooldownState `json:"transitionCooldown,omitempty"`
	TransitionPop      *TransitionPopState      `json:"transitionPop,omitempty"`
}

type HealthState struct {
	Initial int `json:"initial"`
	Current int `json:"current"`
}

type AbilitiesState struct {
	DoubleJump bool `json:"doubleJump"`
	WallGrab   bool `json:"wallGrab"`
	Anchor     bool `json:"anchor"`
	Heal       bool `json:"heal"`
}

type InventoryItem struct {
	Prefab string `json:"prefab"`
	Count  int    `json:"count"`
}

type TransformState struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	ScaleX   float64 `json:"scaleX"`
	ScaleY   float64 `json:"scaleY"`
	Rotation float64 `json:"rotation"`
}

type SafeRespawnState struct {
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Initialized bool    `json:"initialized"`
}

type CheckpointState struct {
	Level       string  `json:"level,omitempty"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	FacingLeft  bool    `json:"facingLeft"`
	Health      int     `json:"health"`
	HealUses    int     `json:"healUses"`
	Initialized bool    `json:"initialized"`
}

type TransitionCooldownState struct {
	Active        bool     `json:"active"`
	TransitionID  string   `json:"transitionId,omitempty"`
	TransitionIDs []string `json:"transitionIds,omitempty"`
}

type TransitionPopState struct {
	VX          float64 `json:"vx"`
	VY          float64 `json:"vy"`
	FacingLeft  bool    `json:"facingLeft"`
	WallJumpDur int     `json:"wallJumpDur"`
	WallJumpX   float64 `json:"wallJumpX"`
	Applied     bool    `json:"applied"`
	Airborne    bool    `json:"airborne"`
}

type Store struct {
	disabled bool
	path     string
	logf     func(format string, args ...any)
	mu       sync.Mutex
	pending  *File
	writing  bool
}

type SlotInfo struct {
	FileName string
	Snapshot *File
}

func NewStore(fileName string, logf func(format string, args ...any)) (*Store, error) {
	if isWebTarget(runtime.GOOS, runtime.GOARCH) {
		return &Store{disabled: true, logf: logf}, nil
	}

	path, err := ResolvePath(fileName)
	if err != nil {
		return nil, err
	}

	return &Store{path: path, logf: logf}, nil
}

func ResolvePath(fileName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve save home directory: %w", err)
	}

	root, err := saveRootDirFor(runtime.GOOS, home)
	if err != nil {
		return "", err
	}

	name, err := normalizeFileName(fileName)
	if err != nil {
		return "", err
	}

	return filepath.Join(root, name), nil
}

func (s *Store) Load() (*File, error) {
	if s == nil {
		return nil, fmt.Errorf("load save: nil store")
	}
	if s.disabled {
		return nil, nil
	}

	return loadPath(s.path)
}

func ListSlots(limit int) ([]SlotInfo, error) {
	if limit <= 0 {
		return nil, nil
	}
	if isWebTarget(runtime.GOOS, runtime.GOARCH) {
		return nil, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve save home directory: %w", err)
	}

	root, err := saveRootDirFor(runtime.GOOS, home)
	if err != nil {
		return nil, err
	}

	return listSlotsInDir(root, limit)
}

func listSlotsInDir(root string, limit int) ([]SlotInfo, error) {
	if limit <= 0 {
		return nil, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list save slots %q: %w", root, err)
	}

	fileNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.ToLower(filepath.Ext(name)) != ".json" {
			continue
		}
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)

	slots := make([]SlotInfo, 0, min(limit, len(fileNames)))
	for _, fileName := range fileNames {
		snapshot, err := loadPath(filepath.Join(root, fileName))
		if err != nil {
			continue
		}
		slots = append(slots, SlotInfo{FileName: fileName, Snapshot: snapshot})
		if len(slots) >= limit {
			break
		}
	}

	return slots, nil
}

func loadPath(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load save %q: %w", path, err)
	}

	return decodeFile(path, data)
}

func decodeFile(path string, data []byte) (*File, error) {
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("decode save %q: %w", path, err)
	}
	if file.Version == 0 {
		file.Version = CurrentVersion
	}
	if file.Version != CurrentVersion {
		return nil, fmt.Errorf("load save %q: unsupported version %d", path, file.Version)
	}
	if strings.TrimSpace(file.Level) == "" {
		return nil, fmt.Errorf("load save %q: missing level", path)
	}
	if file.LevelEntityStates == nil {
		file.LevelEntityStates = map[string]string{}
	}
	if file.LevelLayerStates == nil {
		file.LevelLayerStates = map[string]bool{}
	}

	file.Player.Inventory = cloneInventoryItems(file.Player.Inventory)
	file.LevelLayerStates = cloneLevelLayerStates(file.LevelLayerStates)
	file.LevelEntityStates = cloneLevelEntityStates(file.LevelEntityStates)
	return &file, nil
}

func (s *Store) Save(snapshot *File) error {
	if s == nil {
		return fmt.Errorf("save game: nil store")
	}
	if s.disabled {
		return nil
	}
	if snapshot == nil {
		return fmt.Errorf("save game: nil snapshot")
	}

	cloned := cloneFile(snapshot)
	cloned.Version = CurrentVersion
	cloned.SavedAt = time.Now().UTC()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("save game: create directory: %w", err)
	}

	data, err := json.MarshalIndent(cloned, "", "  ")
	if err != nil {
		return fmt.Errorf("save game: encode json: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(s.path), "save-*.json")
	if err != nil {
		return fmt.Errorf("save game: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("save game: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save game: close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("save game: replace save file: %w", err)
	}

	return nil
}

func (s *Store) SaveAsync(snapshot *File) {
	if s == nil || s.disabled || snapshot == nil {
		return
	}

	s.mu.Lock()
	s.pending = cloneFile(snapshot)
	if s.writing {
		s.mu.Unlock()
		return
	}
	s.writing = true
	s.mu.Unlock()

	go s.drain()
}

func (s *Store) drain() {
	for {
		s.mu.Lock()
		snapshot := s.pending
		s.pending = nil
		s.mu.Unlock()

		if snapshot != nil {
			if err := s.Save(snapshot); err != nil && s.logf != nil {
				s.logf("save game: %v", err)
			}
		}

		s.mu.Lock()
		if s.pending == nil {
			s.writing = false
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
	}
}

func isWebTarget(goos, goarch string) bool {
	return goos == "js" && goarch == "wasm"
}

func saveRootDirFor(goos, home string) (string, error) {
	home = strings.TrimSpace(home)
	if home == "" {
		return "", fmt.Errorf("resolve save directory: home directory is empty")
	}

	switch goos {
	case "windows":
		return filepath.Join(home, "AppData", "LocalLow", "milk9111", "Defective"), nil
	case "linux":
		return filepath.Join(home, ".local", "share", "milk9111", "Defective"), nil
	default:
		configDir, err := os.UserConfigDir()
		if err == nil && strings.TrimSpace(configDir) != "" {
			return filepath.Join(configDir, "milk9111", "Defective"), nil
		}
		return filepath.Join(home, ".config", "milk9111", "Defective"), nil
	}
}

func normalizeFileName(fileName string) (string, error) {
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "save.json"
	}
	if strings.Contains(name, "/") || strings.Contains(name, `\\`) {
		return "", fmt.Errorf("save flag expects a file name, got %q", fileName)
	}
	if filepath.Base(name) != name {
		return "", fmt.Errorf("save flag expects a file name, got %q", fileName)
	}
	if filepath.Ext(name) == "" {
		name += ".json"
	}
	return name, nil
}

func cloneFile(file *File) *File {
	if file == nil {
		return nil
	}

	cloned := *file
	cloned.Player.Inventory = cloneInventoryItems(file.Player.Inventory)
	cloned.LevelLayerStates = cloneLevelLayerStates(file.LevelLayerStates)
	cloned.LevelEntityStates = cloneLevelEntityStates(file.LevelEntityStates)
	return &cloned
}

func cloneLevelLayerStates(states map[string]bool) map[string]bool {
	if len(states) == 0 {
		return map[string]bool{}
	}
	cloned := make(map[string]bool, len(states))
	for key, value := range states {
		cloned[key] = value
	}
	return cloned
}

func cloneInventoryItems(items []InventoryItem) []InventoryItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]InventoryItem, len(items))
	copy(cloned, items)
	return cloned
}

func cloneLevelEntityStates(states map[string]string) map[string]string {
	if len(states) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(states))
	for key, value := range states {
		cloned[key] = value
	}
	return cloned
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
