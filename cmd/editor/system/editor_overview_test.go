package editorsystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

func TestEditorOverviewSystemRebuildsGraphAndDiagnostics(t *testing.T) {
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

	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{OverviewOpen: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.OverviewStateComponent.Kind(), &editorcomponent.OverviewState{Zoom: 1, NeedsRefresh: true})

	system := NewEditorOverviewSystem(root)
	_, session, _ := sessionState(w)
	_, state, _ := overviewState(w)
	system.rebuildGraph(w, session, state)

	if len(state.Nodes) != 3 {
		t.Fatalf("expected 3 overview nodes, got %d", len(state.Nodes))
	}
	if len(state.Edges) != 3 {
		t.Fatalf("expected 3 overview edges, got %d", len(state.Edges))
	}
	aNode := findOverviewNode(state, "a.json")
	if aNode == nil {
		t.Fatal("expected a.json node to exist")
	}
	bNode := findOverviewNode(state, "b.json")
	if bNode == nil {
		t.Fatal("expected b.json node to exist")
	}
	cNode := findOverviewNode(state, "c.json")
	if cNode == nil {
		t.Fatal("expected c.json node to exist")
	}
	if !bNode.HasManual || bNode.X != 420 || bNode.Y != 36 {
		t.Fatalf("expected b.json to use manual layout, got %+v", *bNode)
	}
	if aNode.W != 72 || aNode.H != 72 {
		t.Fatalf("expected 10x10 level node to clamp to 72x72, got %vx%v", aNode.W, aNode.H)
	}
	if cNode.W != 72 || cNode.H != 72 {
		t.Fatalf("expected c.json to use same 10x10 node size, got %vx%v", cNode.W, cNode.H)
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
	if bNode.X == 0 && bNode.Y == 0 {
		t.Fatal("expected b.json node to retain manual position")
	}
}

func TestOverviewNodeSizeScalesWithLevelDimensions(t *testing.T) {
	smallW, smallH := overviewNodeSize(10, 8)
	largeW, largeH := overviewNodeSize(60, 40)
	if largeW <= smallW {
		t.Fatalf("expected larger level width to produce wider node, got small=%v large=%v", smallW, largeW)
	}
	if largeH <= smallH {
		t.Fatalf("expected larger level height to produce taller node, got small=%v large=%v", smallH, largeH)
	}
	if cappedW, cappedH := overviewNodeSize(500, 500); cappedW != overviewNodeMaxWidth || cappedH != overviewNodeMaxHeight {
		t.Fatalf("expected large nodes to clamp to max size, got %vx%v", cappedW, cappedH)
	}
}
