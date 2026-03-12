package editorio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milk9111/sidescroller/cmd/editor/model"
	"github.com/milk9111/sidescroller/levels"
)

func TestNormalizeLevelTargetUsesBasename(t *testing.T) {
	if got := NormalizeLevelTarget("nested/path/zone_a"); got != "zone_a.json" {
		t.Fatalf("expected basename-only normalization, got %q", got)
	}
	if got := NormalizeLevelTarget("../boss/arena.json"); got != "arena.json" {
		t.Fatalf("expected existing extension to remain, got %q", got)
	}
}

func TestSaveLevelWritesPrettyJSONToLevelsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "levels"), 0o755); err != nil {
		t.Fatalf("mkdir levels: %v", err)
	}
	doc := &model.LevelDocument{
		Width:  4,
		Height: 3,
		Layers: []model.Layer{{Name: "Background", Tiles: make([]int, 12), TilesetUsage: make([]*levels.TileInfo, 12)}},
	}

	normalized, err := SaveLevel(root, "levels", "nested/zone_a", doc)
	if err != nil {
		t.Fatalf("save level: %v", err)
	}
	if normalized != "zone_a.json" {
		t.Fatalf("expected normalized name zone_a.json, got %q", normalized)
	}
	data, err := os.ReadFile(filepath.Join(root, "levels", normalized))
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "\n  \"width\": 4,") {
		t.Fatalf("expected pretty-printed JSON, got %q", content)
	}
	if !strings.HasSuffix(content, "\n") {
		t.Fatal("expected trailing newline in saved JSON")
	}
	if _, err := os.Stat(filepath.Join(root, "levels", "nested")); !os.IsNotExist(err) {
		t.Fatalf("expected nested path to be ignored, stat err=%v", err)
	}
}
