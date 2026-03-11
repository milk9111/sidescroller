package app

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	coremodel "github.com/milk9111/sidescroller/internal/editorcore/model"
	"github.com/milk9111/sidescroller/levels"
)

func TestRebuildOverviewGraphBuildsNodesEdgesAndDiagnostics(t *testing.T) {
	root := t.TempDir()
	levelsDir := filepath.Join(root, "levels")
	if err := os.MkdirAll(levelsDir, 0o755); err != nil {
		t.Fatalf("mkdir levels: %v", err)
	}
	writeLevel := func(name string, level levels.Level) {
		data, err := json.Marshal(level)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(levelsDir, name), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeLevel("a.json", levels.Level{Width: 10, Height: 10, Entities: []levels.Entity{{Type: "transition", Props: map[string]interface{}{"to_level": "b", "enter_dir": "left"}}, {Type: "transition", Props: map[string]interface{}{"to_level": "missing", "enter_dir": "up"}}}})
	writeLevel("b.json", levels.Level{Width: 10, Height: 10})
	writeLevel("c.json", levels.Level{Width: 10, Height: 10, Entities: []levels.Entity{{Type: "transition", Props: map[string]interface{}{"to_level": "b", "enter_dir": "left"}}}})
	if err := os.WriteFile(filepath.Join(root, ".level_overview_layout.json"), []byte("{\n  \"levels\": {\n    \"b.json\": {\"x\": 420, \"y\": 36}\n  }\n}\n"), 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}

	state := NewState(Config{WorkspaceRoot: root, SaveTarget: "a.json", Level: coremodel.NewLevelDocument(8, 6)})
	state.Apply(Action{Kind: ActionToggleOverview})
	state.rebuildOverviewGraph()

	if len(state.Overview.Nodes) != 3 {
		t.Fatalf("expected 3 overview nodes, got %d", len(state.Overview.Nodes))
	}
	if len(state.Overview.Edges) != 3 {
		t.Fatalf("expected 3 overview edges, got %d", len(state.Overview.Edges))
	}
	aNode := findOverviewNode(&state.Overview, "a.json")
	if aNode == nil {
		t.Fatal("expected a.json node to exist")
	}
	bNode := findOverviewNode(&state.Overview, "b.json")
	if bNode == nil {
		t.Fatal("expected b.json node to exist")
	}
	if !bNode.HasManual || bNode.X != 420 || bNode.Y != 36 {
		t.Fatalf("expected b.json to use manual layout, got %+v", *bNode)
	}
	if len(aNode.Diagnostics) == 0 {
		t.Fatal("expected diagnostics for a.json")
	}
	foundMissing := false
	for _, diag := range aNode.Diagnostics {
		if diag == "Missing target missing.json" {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatalf("expected missing-target diagnostic, got %v", aNode.Diagnostics)
	}
}

func TestUpdateOverviewClickLoadsLevelAndClosesOverview(t *testing.T) {
	root := t.TempDir()
	levelsDir := filepath.Join(root, "levels")
	if err := os.MkdirAll(levelsDir, 0o755); err != nil {
		t.Fatalf("mkdir levels: %v", err)
	}
	writeLevel := func(name string, level levels.Level) {
		data, err := json.Marshal(level)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(levelsDir, name), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeLevel("alpha.json", levels.Level{Width: 8, Height: 6})
	writeLevel("beta.json", levels.Level{Width: 12, Height: 9})

	state := NewState(Config{WorkspaceRoot: root, SaveTarget: "alpha.json", LevelName: "alpha.json", Level: coremodel.NewLevelDocument(8, 6)})
	state.Apply(Action{Kind: ActionToggleOverview})
	state.rebuildOverviewGraph()

	target := findOverviewNode(&state.Overview, "beta.json")
	if target == nil {
		t.Fatal("expected beta.json node to exist")
	}
	rect := image.Rect(0, 0, 1200, 800)
	clickX := int(target.X + 12)
	clickY := int(target.Y + 12)

	state.UpdateOverview(rect, ViewportInput{MouseX: clickX, MouseY: clickY, LeftDown: true, LeftJustPressed: true})
	state.UpdateOverview(rect, ViewportInput{MouseX: clickX, MouseY: clickY, LeftJustReleased: true})

	if state.Overview.Open {
		t.Fatal("expected overview to close after loading level from node click")
	}
	if state.LoadedLevel != "beta.json" {
		t.Fatalf("expected beta.json to load, got %q", state.LoadedLevel)
	}
	if state.Document.Width != 12 || state.Document.Height != 9 {
		t.Fatalf("expected loaded level dimensions 12x9, got %dx%d", state.Document.Width, state.Document.Height)
	}
}