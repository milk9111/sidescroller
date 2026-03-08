package editorsystem

import (
	"testing"

	editorautotile "github.com/milk9111/sidescroller/cmd/editor/autotile"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

func TestEditorToolSystemBoxPaintsFilledRectangle(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{
		ActiveTool:   editorcomponent.ToolBox,
		CurrentLayer: 0,
		SelectedTile: modelSelectionForTest("terrain.png", 7),
	})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 6, Height: 6})
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{LeftJustPressed: true, LeftDown: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{InCanvas: true, HasCell: true, CellX: 1, CellY: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.ToolStrokeComponent.Kind(), &editorcomponent.ToolStroke{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})
	_ = ecs.Add(w, sessionEntity, editorcomponent.AutotileStateComponent.Kind(), &editorcomponent.AutotileState{DirtyCells: map[int]map[int]struct{}{}, FullRebuild: map[int]bool{}})
	layerEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Layer 1", Order: 0, Tiles: make([]int, 36), TilesetUsage: make([]*levels.TileInfo, 36)})

	system := NewEditorToolSystem()
	system.Update(w)

	_, layer, _ := layerAt(w, 0)
	if layer.Tiles[cellIndex(&editorcomponent.LevelMeta{Width: 6, Height: 6}, 1, 1)] != 0 {
		t.Fatal("expected box tool to wait until release before painting")
	}
	_, stroke, _ := strokeState(w)
	if len(stroke.Preview) != 1 {
		t.Fatalf("expected single-cell preview on press, got %d cells", len(stroke.Preview))
	}

	_, input, _ := rawInputState(w)
	_, pointer, _ := pointerState(w)
	input.LeftJustPressed = false
	input.LeftDown = true
	pointer.CellX = 3
	pointer.CellY = 2
	system.Update(w)

	if len(stroke.Preview) != 6 {
		t.Fatalf("expected 6 preview cells for 3x2 box, got %d", len(stroke.Preview))
	}

	input.LeftDown = false
	input.LeftJustReleased = true
	system.Update(w)

	for y := 1; y <= 2; y++ {
		for x := 1; x <= 3; x++ {
			index := cellIndex(&editorcomponent.LevelMeta{Width: 6, Height: 6}, x, y)
			if layer.Tiles[index] != 7 {
				t.Fatalf("expected tile value 7 at (%d,%d), got %d", x, y, layer.Tiles[index])
			}
			usage := layer.TilesetUsage[index]
			if usage == nil || usage.Path != "terrain.png" || usage.Index != 7 {
				t.Fatalf("expected selected tile usage at (%d,%d), got %+v", x, y, usage)
			}
		}
	}
	if stroke.Active {
		t.Fatal("expected box stroke to finish on release")
	}
	if stroke.Preview != nil {
		t.Fatal("expected box preview to clear on release")
	}
	_, session, _ := sessionState(w)
	if session.Status != "Box placed" {
		t.Fatalf("expected box placement status, got %q", session.Status)
	}
	_, undo, _ := undoState(w)
	if len(undo.Snapshots) != 1 {
		t.Fatalf("expected one undo snapshot, got %d", len(undo.Snapshots))
	}
}

func TestEditorToolSystemBoxSupportsAutotile(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	meta := &editorcomponent.LevelMeta{Width: 5, Height: 5}
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{
		ActiveTool:   editorcomponent.ToolBox,
		CurrentLayer: 0,
		SelectedTile: modelSelectionForTest("terrain.png", 12),
	})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), meta)
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{LeftJustPressed: true, LeftDown: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{InCanvas: true, HasCell: true, CellX: 1, CellY: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.ToolStrokeComponent.Kind(), &editorcomponent.ToolStroke{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})
	_ = ecs.Add(w, sessionEntity, editorcomponent.AutotileStateComponent.Kind(), &editorcomponent.AutotileState{Enabled: true, DirtyCells: map[int]map[int]struct{}{}, FullRebuild: map[int]bool{}})
	layerEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Layer 1", Order: 0, Tiles: make([]int, 25), TilesetUsage: make([]*levels.TileInfo, 25)})

	toolSystem := NewEditorToolSystem()
	toolSystem.Update(w)

	_, input, _ := rawInputState(w)
	_, pointer, _ := pointerState(w)
	input.LeftJustPressed = false
	input.LeftDown = true
	pointer.CellX = 3
	pointer.CellY = 3
	toolSystem.Update(w)

	input.LeftDown = false
	input.LeftJustReleased = true
	toolSystem.Update(w)
	NewEditorAutotileSystem().Update(w)

	_, layer, _ := layerAt(w, 0)
	center := layer.TilesetUsage[cellIndex(meta, 2, 2)]
	if center == nil || !center.Auto {
		t.Fatalf("expected autotile usage at center, got %+v", center)
	}
	expectedMask := editorautotile.BuildMask(true, true, true, true, true, true, true, true)
	if center.Mask != expectedMask {
		t.Fatalf("expected center autotile mask %d, got %d", expectedMask, center.Mask)
	}
	if center.Index == 0 {
		t.Fatal("expected autotile system to resolve a non-zero center index")
	}
	corner := layer.TilesetUsage[cellIndex(meta, 1, 1)]
	if corner == nil || !corner.Auto {
		t.Fatalf("expected autotile usage at corner, got %+v", corner)
	}
	if corner.Mask == center.Mask {
		t.Fatalf("expected corner mask to differ from center mask, both were %d", corner.Mask)
	}
}

func modelSelectionForTest(path string, index int) model.TileSelection {
	return model.TileSelection{Path: path, Index: index, TileW: model.DefaultTileSize, TileH: model.DefaultTileSize}
}

func TestApplySpikeAtCreatesAndReusesSpikeEntity(t *testing.T) {
	w := ecs.NewWorld()
	entity := ecs.CreateEntity(w)
	_ = ecs.Add(w, entity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 1})
	_ = ecs.Add(w, entity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 4, Height: 4})
	_ = ecs.Add(w, entity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{})
	physicsLayer := ecs.CreateEntity(w)
	tiles := make([]int, 16)
	tiles[2*4+1] = 1
	_ = ecs.Add(w, physicsLayer, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Physics", Order: 0, Physics: true, Tiles: tiles, TilesetUsage: make([]*levels.TileInfo, 16)})

	tool := NewEditorToolSystem()
	_, session, _ := sessionState(w)
	_, meta, _ := levelMetaState(w)
	if !tool.applySpikeAt(w, session, meta, 1, 1) {
		t.Fatal("expected first spike placement to create entity")
	}
	_, entities, _ := entitiesState(w)
	if len(entities.Items) != 1 {
		t.Fatalf("expected one spike entity, got %d", len(entities.Items))
	}
	placed := entities.Items[0]
	if !isSpikeEntity(placed) {
		t.Fatalf("expected spike type, got %q", placed.Type)
	}
	if got := toFloat(placed.Props["rotation"]); got != 0 {
		t.Fatalf("expected upward spike rotation 0, got %v", got)
	}
	if got, ok := entityLayerIndex(placed.Props); !ok || got != 1 {
		t.Fatalf("expected layer 1, got %d (ok=%t)", got, ok)
	}

	session.CurrentLayer = 2
	if tool.applySpikeAt(w, session, meta, 1, 1) {
		if len(entities.Items) != 1 {
			t.Fatalf("expected existing spike to be reused, got %d entities", len(entities.Items))
		}
	}
	if got, ok := entityLayerIndex(entities.Items[0].Props); !ok || got != 2 {
		t.Fatalf("expected reused spike layer 2, got %d (ok=%t)", got, ok)
	}
}
