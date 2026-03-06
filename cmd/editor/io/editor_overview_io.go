package editorio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/milk9111/sidescroller/levels"
)

type OverviewTransitionRecord struct {
	ID       string
	ToLevel  string
	LinkedID string
	EnterDir string
}

type OverviewLevelRecord struct {
	Name        string
	Width       int
	Height      int
	Transitions []OverviewTransitionRecord
}

type OverviewLayoutEntry struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type OverviewLayoutFile struct {
	Levels map[string]OverviewLayoutEntry `json:"levels"`
}

func ScanLevelsForOverview(workspaceRoot string) ([]OverviewLevelRecord, error) {
	root := filepath.Join(workspaceRoot, "levels")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read levels dir: %w", err)
	}
	records := make([]OverviewLevelRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		path := filepath.Join(root, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read level %q: %w", entry.Name(), err)
		}
		var level levels.Level
		if err := json.Unmarshal(data, &level); err != nil {
			return nil, fmt.Errorf("decode level %q: %w", entry.Name(), err)
		}
		record := OverviewLevelRecord{Name: entry.Name(), Width: level.Width, Height: level.Height}
		for _, entity := range level.Entities {
			if !strings.EqualFold(strings.TrimSpace(entity.Type), "transition") {
				continue
			}
			record.Transitions = append(record.Transitions, OverviewTransitionRecord{
				ID:       strings.TrimSpace(entityStringProp(entity.Props, "id")),
				ToLevel:  NormalizeLevelTarget(entityStringProp(entity.Props, "to_level")),
				LinkedID: strings.TrimSpace(entityStringProp(entity.Props, "linked_id")),
				EnterDir: strings.ToLower(strings.TrimSpace(entityStringProp(entity.Props, "enter_dir"))),
			})
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })
	return records, nil
}

func LoadOverviewLayout(workspaceRoot string) (map[string]OverviewLayoutEntry, error) {
	path := filepath.Join(workspaceRoot, ".level_overview_layout.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]OverviewLayoutEntry{}, nil
		}
		return nil, fmt.Errorf("read overview layout: %w", err)
	}
	var file OverviewLayoutFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("decode overview layout: %w", err)
	}
	if file.Levels == nil {
		file.Levels = map[string]OverviewLayoutEntry{}
	}
	return file.Levels, nil
}

func SaveOverviewLayout(workspaceRoot string, layout map[string]OverviewLayoutEntry) error {
	path := filepath.Join(workspaceRoot, ".level_overview_layout.json")
	file := OverviewLayoutFile{Levels: layout}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode overview layout: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write overview layout: %w", err)
	}
	return nil
}

func entityStringProp(props map[string]interface{}, key string) string {
	if props == nil {
		return ""
	}
	if value, ok := props[key].(string); ok {
		return value
	}
	return ""
}
