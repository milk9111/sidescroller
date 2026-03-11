package app

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreio "github.com/milk9111/sidescroller/internal/editorcore/io"
	coremodel "github.com/milk9111/sidescroller/internal/editorcore/model"
	"github.com/milk9111/sidescroller/levels"
)

func TestLayerActionsMutateDocumentAndEntityLayers(t *testing.T) {
	state := NewState(Config{
		WorkspaceRoot: t.TempDir(),
		SaveTarget:    "test.json",
		Level:         levelWithEntities(),
	})

	state.Apply(Action{Kind: ActionAddLayer})
	if len(state.Document.Layers) != 3 {
		t.Fatalf("expected 3 layers after add, got %d", len(state.Document.Layers))
	}
	if !state.Dirty {
		t.Fatal("expected add layer to mark state dirty")
	}

	state.Apply(Action{Kind: ActionMoveCurrentLayer, Delta: -2})
	if got := entityLayer(state.Document.Entities[0]); got != 1 {
		t.Fatalf("expected first entity layer to remap to 1, got %d", got)
	}
	if got := entityLayer(state.Document.Entities[1]); got != 2 {
		t.Fatalf("expected second entity layer to remap to 2, got %d", got)
	}

	state.Apply(Action{Kind: ActionDeleteCurrentLayer})
	if len(state.Document.Layers) != 2 {
		t.Fatalf("expected 2 layers after delete, got %d", len(state.Document.Layers))
	}
	if len(state.Document.Entities) != 2 {
		t.Fatalf("expected delete to preserve the original entities, got %d entities", len(state.Document.Entities))
	}
	if got := entityLayer(state.Document.Entities[0]); got != 0 {
		t.Fatalf("expected first remaining entity layer to shift to 0, got %d", got)
	}
	if got := entityLayer(state.Document.Entities[1]); got != 1 {
		t.Fatalf("expected second remaining entity layer to shift to 1, got %d", got)
	}
	if state.LayerNameInput == "" {
		t.Fatal("expected current layer name input to stay synced")
	}
}

func TestSaveLoadAndUndoFlow(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "levels"), 0o755); err != nil {
		t.Fatalf("mkdir levels: %v", err)
	}

	state := NewState(Config{
		WorkspaceRoot: root,
		SaveTarget:    "alpha.json",
		Level:         coremodel.NewLevelDocument(8, 6),
	})
	state.Document.Entities = []levels.Entity{{Type: "transition", Props: map[string]interface{}{"layer": 0}}}

	state.Apply(Action{Kind: ActionRenameCurrentLayer, Name: "Gameplay"})
	state.Apply(Action{Kind: ActionSaveLevel, Target: state.SaveTarget})
	if state.Dirty {
		t.Fatal("expected save to clear dirty flag")
	}
	if state.Document.Entities[0].ID == "" {
		t.Fatal("expected save to assign unique entity ids")
	}

	state.Apply(Action{Kind: ActionRenameCurrentLayer, Name: "Changed"})
	state.Apply(Action{Kind: ActionUndo})
	if got := state.Document.Layers[state.CurrentLayer].Name; got != "Gameplay" {
		t.Fatalf("expected undo to restore layer name, got %q", got)
	}

	state.Apply(Action{Kind: ActionLoadLevel, Target: "alpha"})
	if got := state.Document.Layers[state.CurrentLayer].Name; got != "Gameplay" {
		t.Fatalf("expected load to restore saved layer name, got %q", got)
	}
	if state.LoadedLevel != "alpha.json" {
		t.Fatalf("expected normalized loaded level, got %q", state.LoadedLevel)
	}
	if len(state.UndoStack) != 0 {
		t.Fatalf("expected load to clear undo stack, got %d entries", len(state.UndoStack))
	}
}

func TestEntityPrefabPlacementAndClipboardFlow(t *testing.T) {
	state := NewState(Config{
		WorkspaceRoot: t.TempDir(),
		SaveTarget:    "test.json",
		Level:         coremodel.NewLevelDocument(8, 6),
		Prefabs: []coreio.PrefabInfo{{
			Name:       "Enemy",
			Path:       "enemy.yaml",
			EntityType: "enemy",
			Components: map[string]any{
				"transform": map[string]any{"x": 0.0, "y": 0.0, "scale_x": 1.0, "scale_y": 1.0},
			},
		}},
	})

	state.Apply(Action{Kind: ActionSelectPrefab, Path: "enemy.yaml"})
	state.Apply(Action{Kind: ActionPlacePrefab, CellX: 4, CellY: 6})
	if len(state.Document.Entities) != 1 {
		t.Fatalf("expected one entity after placement, got %d", len(state.Document.Entities))
	}
	placed := state.Document.Entities[0]
	if placed.Type != "enemy" {
		t.Fatalf("expected enemy type, got %q", placed.Type)
	}
	if placed.X != 4*entityTileSize || placed.Y != 6*entityTileSize {
		t.Fatalf("expected placed coordinates (%d,%d), got (%d,%d)", 4*entityTileSize, 6*entityTileSize, placed.X, placed.Y)
	}
	if got, ok := placed.Props["layer"].(int); !ok || got != 0 {
		t.Fatalf("expected placed entity on layer 0, got %v", placed.Props["layer"])
	}
	if state.SelectedEntity != 0 {
		t.Fatalf("expected placed entity to become selected, got %d", state.SelectedEntity)
	}

	state.Apply(Action{Kind: ActionCopyEntity})
	state.Apply(Action{Kind: ActionPasteEntity})
	if len(state.Document.Entities) != 2 {
		t.Fatalf("expected pasted entity, got %d entities", len(state.Document.Entities))
	}
	if state.SelectedEntity != 1 {
		t.Fatalf("expected pasted entity selection at index 1, got %d", state.SelectedEntity)
	}
	if state.Document.Entities[1].ID == state.Document.Entities[0].ID && state.Document.Entities[1].ID != "" {
		t.Fatalf("expected pasted entity id to be unique, both were %q", state.Document.Entities[1].ID)
	}
	if prefabPath, _ := state.Document.Entities[1].Props["prefab"].(string); prefabPath != "enemy.yaml" {
		t.Fatalf("expected pasted prefab path enemy.yaml, got %q", prefabPath)
	}
}

func TestInspectorEditUpdatesOverridesAndTransform(t *testing.T) {
	state := NewState(Config{
		WorkspaceRoot: t.TempDir(),
		SaveTarget:    "test.json",
		Level:         coremodel.NewLevelDocument(8, 6),
		Prefabs: []coreio.PrefabInfo{{
			Name:       "Enemy",
			Path:       "enemy.yaml",
			EntityType: "enemy",
			Components: map[string]any{
				"transform": map[string]any{"x": 0.0, "y": 0.0, "scale_x": 1.0, "scale_y": 1.0, "rotation": 0.0},
				"color":     map[string]any{"hex": "#ffffff"},
			},
		}},
	})
	state.Document.Entities = []levels.Entity{{Type: "enemy", Props: map[string]interface{}{"prefab": "enemy.yaml", "layer": 0}}}
	state.syncDerivedState(false)

	state.Apply(Action{Kind: ActionSelectEntity, Index: 0})
	state.Apply(Action{Kind: ActionEditInspectorField, Name: "transform", Field: "x", Value: "96"})
	state.Apply(Action{Kind: ActionEditInspectorField, Name: "color", Field: "hex", Value: "#ff0000"})

	if state.Document.Entities[0].X != 96 {
		t.Fatalf("expected transform x edit to update entity X to 96, got %d", state.Document.Entities[0].X)
	}
	components, ok := state.Document.Entities[0].Props[entityComponentsKey].(map[string]any)
	if !ok {
		t.Fatalf("expected inspector edit to create component overrides, got %#v", state.Document.Entities[0].Props[entityComponentsKey])
	}
	color, ok := components["color"].(map[string]any)
	if !ok || color["hex"] != "#ff0000" {
		t.Fatalf("expected color override to be updated, got %#v", components["color"])
	}
	if state.InspectorInputs[inspectorFieldKey("color", "hex")] == nil || *state.InspectorInputs[inspectorFieldKey("color", "hex")] != "#ff0000" {
		got := ""
		if state.InspectorInputs[inspectorFieldKey("color", "hex")] != nil {
			got = *state.InspectorInputs[inspectorFieldKey("color", "hex")]
		}
		t.Fatalf("expected inspector input to resync to #ff0000, got %q", got)
	}
}

func TestConvertSelectedEntityToPrefabSavesAndRefreshesCatalog(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "prefabs"), 0o755); err != nil {
		t.Fatalf("mkdir prefabs: %v", err)
	}
	state := NewState(Config{
		WorkspaceRoot: root,
		SaveTarget:    "test.json",
		Level:         coremodel.NewLevelDocument(8, 6),
		Prefabs: []coreio.PrefabInfo{{
			Name:       "Enemy",
			Path:       "enemy.yaml",
			EntityType: "enemy",
			Components: map[string]any{
				"transform": map[string]any{"x": 0.0, "y": 0.0, "scale_x": 1.0, "scale_y": 1.0},
			},
		}},
	})
	state.Document.Entities = []levels.Entity{{
		Type: "enemy",
		X:    64,
		Y:    96,
		Props: map[string]interface{}{
			"prefab": "enemy.yaml",
			"layer":  0,
			entityComponentsKey: map[string]any{
				"color": map[string]any{"hex": "#00ff00"},
			},
		},
	}}
	state.syncDerivedState(false)

	state.Apply(Action{Kind: ActionSelectEntity, Index: 0})
	state.Apply(Action{Kind: ActionConvertToPrefab, Target: "enemy_variant"})

	if !strings.Contains(state.Status, "prefabs/enemy_variant.yaml") {
		t.Fatalf("expected successful convert status, got %q", state.Status)
	}
	if prefabPath, _ := state.Document.Entities[0].Props["prefab"].(string); prefabPath != "enemy_variant.yaml" {
		t.Fatalf("expected entity prefab path to update, got %q", prefabPath)
	}
	if _, ok := state.Document.Entities[0].Props[entityComponentsKey]; ok {
		t.Fatal("expected entity component overrides to be cleared after conversion")
	}
	if _, err := os.Stat(filepath.Join(root, "prefabs", "enemy_variant.yaml")); err != nil {
		t.Fatalf("expected converted prefab file to exist: %v", err)
	}
	found := false
	for _, prefab := range state.Prefabs {
		if prefab.Path == "enemy_variant.yaml" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected refreshed prefab catalog to include enemy_variant.yaml, got %#v", state.Prefabs)
	}
}

func TestViewportBrushUndoAndSamplingFlow(t *testing.T) {
	state := NewState(Config{
		WorkspaceRoot: t.TempDir(),
		SaveTarget:    "paint.json",
		Level:         coremodel.NewLevelDocument(8, 6),
		Assets: []coreio.AssetInfo{{
			Name:     "terrain.png",
			DiskPath: "/tmp/terrain.png",
			Relative: "terrain.png",
		}},
	})
	state.Apply(Action{Kind: ActionSelectTool, Name: string(ToolBrush)})
	rect := image.Rect(0, 0, 320, 320)

	state.UpdateViewport(rect, viewportInputAtCell(1, 1, ViewportInput{LeftDown: true, LeftJustPressed: true}))
	state.UpdateViewport(rect, viewportInputAtCell(3, 1, ViewportInput{LeftDown: true}))
	state.UpdateViewport(rect, viewportInputAtCell(3, 1, ViewportInput{LeftJustReleased: true}))

	layer := &state.Document.Layers[state.CurrentLayer]
	for x := 1; x <= 3; x++ {
		index := state.cellIndex(x, 1)
		if layer.Tiles[index] != state.SelectedTile.Index {
			t.Fatalf("expected painted tile at (%d,1), got %d", x, layer.Tiles[index])
		}
	}
	if len(state.UndoStack) != 1 {
		t.Fatalf("expected one undo snapshot after brush stroke, got %d", len(state.UndoStack))
	}

	state.UpdateViewport(rect, viewportInputAtCell(2, 1, ViewportInput{RightJustPressed: true}))
	if state.SelectedTile.Path != "terrain.png" {
		t.Fatalf("expected sampling to keep terrain.png selected, got %q", state.SelectedTile.Path)
	}

	state.Apply(Action{Kind: ActionUndo})
	layer = &state.Document.Layers[state.CurrentLayer]
	for x := 1; x <= 3; x++ {
		index := state.cellIndex(x, 1)
		if layer.Tiles[index] != 0 || layer.TilesetUsage[index] != nil {
			t.Fatalf("expected undo to clear tile at (%d,1), got %d %+v", x, layer.Tiles[index], layer.TilesetUsage[index])
		}
	}
}

func TestViewportBoxAutotileAndCamera(t *testing.T) {
	state := NewState(Config{
		WorkspaceRoot: t.TempDir(),
		SaveTarget:    "autotile.json",
		Level:         coremodel.NewLevelDocument(6, 6),
		Assets: []coreio.AssetInfo{{
			Name:     "grass.png",
			DiskPath: "/tmp/grass.png",
			Relative: "grass.png",
		}},
	})
	state.Apply(Action{Kind: ActionSelectAsset, Path: "grass.png"})
	state.Apply(Action{Kind: ActionToggleAutotile})
	state.Apply(Action{Kind: ActionSelectTool, Name: string(ToolBox)})
	rect := image.Rect(0, 0, 320, 320)

	state.UpdateViewport(rect, viewportInputAtCell(1, 1, ViewportInput{LeftDown: true, LeftJustPressed: true}))
	state.UpdateViewport(rect, viewportInputAtCell(3, 3, ViewportInput{LeftDown: true}))
	state.UpdateViewport(rect, viewportInputAtCell(3, 3, ViewportInput{LeftJustReleased: true}))

	center := state.Document.Layers[state.CurrentLayer].TilesetUsage[state.cellIndex(2, 2)]
	if center == nil || !center.Auto {
		t.Fatalf("expected autotile usage at center, got %+v", center)
	}
	if center.Mask == 0 {
		t.Fatalf("expected autotile mask at center, got %d", center.Mask)
	}

	beforeZoom := state.Camera.Zoom
	state.UpdateViewport(rect, viewportInputAtCell(2, 2, ViewportInput{WheelY: 1}))
	if state.Camera.Zoom <= beforeZoom {
		t.Fatalf("expected camera zoom to increase, before=%f after=%f", beforeZoom, state.Camera.Zoom)
	}
	state.UpdateViewport(rect, viewportInputAtCell(2, 2, ViewportInput{MiddleDown: true, MiddleJustPressed: true}))
	state.UpdateViewport(rect, viewportInputAtPixel(120, 120, ViewportInput{MiddleDown: true}))
	state.UpdateViewport(rect, viewportInputAtPixel(120, 120, ViewportInput{MiddleJustReleased: true}))
	if state.Camera.X == 0 && state.Camera.Y == 0 {
		t.Fatal("expected middle-button drag to pan camera")
	}
}

func levelWithEntities() *coremodel.LevelDocument {
	doc := coremodel.NewLevelDocument(10, 6)
	doc.Entities = []levels.Entity{
		{Type: "enemy", Props: map[string]interface{}{"layer": 0}},
		{Type: "pickup", Props: map[string]interface{}{"layer": 1}},
	}
	return doc
}

func entityLayer(item levels.Entity) int {
	if item.Props == nil {
		return 0
	}
	switch value := item.Props["layer"].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}

func viewportInputAtCell(cellX, cellY int, input ViewportInput) ViewportInput {
	input.MouseX = (cellX * tileSize) + 8
	input.MouseY = (cellY * tileSize) + 8
	return input
}

func viewportInputAtPixel(x, y int, input ViewportInput) ViewportInput {
	input.MouseX = x
	input.MouseY = y
	return input
}
