package editorsystem

import (
	"testing"

	editorautotile "github.com/milk9111/sidescroller/cmd/editor/autotile"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
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

func TestEditorToolSystemBoxEraseClearsFilledRectangle(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	meta := &editorcomponent.LevelMeta{Width: 6, Height: 6}
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{
		ActiveTool:   editorcomponent.ToolBoxErase,
		CurrentLayer: 0,
	})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), meta)
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{LeftJustPressed: true, LeftDown: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{InCanvas: true, HasCell: true, CellX: 1, CellY: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.ToolStrokeComponent.Kind(), &editorcomponent.ToolStroke{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})
	_ = ecs.Add(w, sessionEntity, editorcomponent.AutotileStateComponent.Kind(), &editorcomponent.AutotileState{DirtyCells: map[int]map[int]struct{}{}, FullRebuild: map[int]bool{}})
	layerEntity := ecs.CreateEntity(w)
	tiles := make([]int, 36)
	usage := make([]*levels.TileInfo, 36)
	for y := 1; y <= 2; y++ {
		for x := 1; x <= 3; x++ {
			index := cellIndex(meta, x, y)
			tiles[index] = 9
			usage[index] = &levels.TileInfo{Path: "terrain.png", Index: 9, TileW: model.DefaultTileSize, TileH: model.DefaultTileSize}
		}
	}
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Layer 1", Order: 0, Tiles: tiles, TilesetUsage: usage})

	system := NewEditorToolSystem()
	system.Update(w)

	_, layer, _ := layerAt(w, 0)
	if layer.Tiles[cellIndex(meta, 1, 1)] != 9 {
		t.Fatal("expected box erase tool to wait until release before erasing")
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
		t.Fatalf("expected 6 preview cells for 3x2 box erase, got %d", len(stroke.Preview))
	}

	input.LeftDown = false
	input.LeftJustReleased = true
	system.Update(w)

	for y := 1; y <= 2; y++ {
		for x := 1; x <= 3; x++ {
			index := cellIndex(meta, x, y)
			if layer.Tiles[index] != 0 {
				t.Fatalf("expected tile value 0 at (%d,%d), got %d", x, y, layer.Tiles[index])
			}
			if layer.TilesetUsage[index] != nil {
				t.Fatalf("expected tile usage cleared at (%d,%d), got %+v", x, y, layer.TilesetUsage[index])
			}
		}
	}
	if stroke.Active {
		t.Fatal("expected box erase stroke to finish on release")
	}
	if stroke.Preview != nil {
		t.Fatal("expected box erase preview to clear on release")
	}
	_, session, _ := sessionState(w)
	if session.Status != "Box erased" {
		t.Fatalf("expected box erase status, got %q", session.Status)
	}
	_, undo, _ := undoState(w)
	if len(undo.Snapshots) != 1 {
		t.Fatalf("expected one undo snapshot, got %d", len(undo.Snapshots))
	}
}

func TestEditorToolSystemMoveMovesTilesEmptySpaceAndEntities(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	meta := &editorcomponent.LevelMeta{Width: 8, Height: 8}
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{
		ActiveTool:   editorcomponent.ToolMove,
		CurrentLayer: 0,
	})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), meta)
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{LeftJustPressed: true, LeftDown: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{InCanvas: true, HasCell: true, CellX: 1, CellY: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.ToolStrokeComponent.Kind(), &editorcomponent.ToolStroke{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.MoveSelectionComponent.Kind(), &editorcomponent.MoveSelectionState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{
		{ID: "enemy_1", Type: "enemy", X: 32, Y: 32, Props: map[string]interface{}{"layer": 0, "prefab": "enemy.yaml"}},
		{ID: "enemy_2", Type: "enemy", X: 224, Y: 224, Props: map[string]interface{}{"layer": 0, "prefab": "enemy.yaml"}},
	}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy", Preview: editorio.PrefabPreview{FrameW: 32, FrameH: 32}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})
	_ = ecs.Add(w, sessionEntity, editorcomponent.AutotileStateComponent.Kind(), &editorcomponent.AutotileState{DirtyCells: map[int]map[int]struct{}{}, FullRebuild: map[int]bool{}})
	layerOne := ecs.CreateEntity(w)
	layerOneTiles := make([]int, 64)
	layerOneUsage := make([]*levels.TileInfo, 64)
	for _, tc := range []struct {
		x, y, value int
	}{{1, 1, 1}, {3, 1, 2}, {1, 2, 3}, {2, 2, 4}, {3, 2, 5}, {2, 4, 9}, {3, 4, 9}, {4, 4, 9}, {2, 5, 9}, {3, 5, 9}, {4, 5, 9}} {
		index := cellIndex(meta, tc.x, tc.y)
		layerOneTiles[index] = tc.value
		layerOneUsage[index] = &levels.TileInfo{Path: "terrain.png", Index: tc.value, TileW: model.DefaultTileSize, TileH: model.DefaultTileSize}
	}
	_ = ecs.Add(w, layerOne, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Layer 1", Order: 0, Tiles: layerOneTiles, TilesetUsage: layerOneUsage})
	layerTwo := ecs.CreateEntity(w)
	layerTwoTiles := make([]int, 64)
	layerTwoUsage := make([]*levels.TileInfo, 64)
	for _, tc := range []struct {
		x, y, value int
	}{{1, 1, 6}, {2, 1, 7}, {3, 1, 8}, {1, 2, 10}, {2, 2, 11}, {3, 2, 12}, {2, 3, 13}, {3, 3, 13}, {4, 3, 13}, {2, 4, 13}, {3, 4, 13}, {4, 4, 13}} {
		index := cellIndex(meta, tc.x, tc.y)
		layerTwoTiles[index] = tc.value
		layerTwoUsage[index] = &levels.TileInfo{Path: "detail.png", Index: tc.value, TileW: model.DefaultTileSize, TileH: model.DefaultTileSize}
	}
	_ = ecs.Add(w, layerTwo, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Layer 2", Order: 1, Tiles: layerTwoTiles, TilesetUsage: layerTwoUsage})

	system := NewEditorToolSystem()
	system.Update(w)

	_, input, _ := rawInputState(w)
	_, pointer, _ := pointerState(w)
	input.LeftJustPressed = false
	input.LeftDown = true
	pointer.CellX = 3
	pointer.CellY = 2
	system.Update(w)

	input.LeftDown = false
	input.LeftJustReleased = true
	system.Update(w)

	_, move, _ := moveSelectionState(w)
	if !move.Active || move.Width != 3 || move.Height != 2 {
		t.Fatalf("expected active 3x2 move selection, got %+v", move)
	}

	input.LeftJustReleased = false
	input.LeftJustPressed = true
	input.LeftDown = true
	pointer.CellX = 1
	pointer.CellY = 1
	system.Update(w)

	input.LeftJustPressed = false
	input.LeftDown = true
	pointer.CellX = 2
	pointer.CellY = 3
	system.Update(w)

	input.LeftDown = false
	input.LeftJustReleased = true
	system.Update(w)

	_, firstLayer, _ := layerAt(w, 0)
	_, secondLayer, _ := layerAt(w, 1)
	for y := 1; y <= 2; y++ {
		for x := 1; x <= 3; x++ {
			index := cellIndex(meta, x, y)
			if firstLayer.Tiles[index] != 0 || firstLayer.TilesetUsage[index] != nil {
				t.Fatalf("expected source cleared on layer 1 at (%d,%d), got %d %+v", x, y, firstLayer.Tiles[index], firstLayer.TilesetUsage[index])
			}
			if secondLayer.Tiles[index] != 0 || secondLayer.TilesetUsage[index] != nil {
				t.Fatalf("expected source cleared on layer 2 at (%d,%d), got %d %+v", x, y, secondLayer.Tiles[index], secondLayer.TilesetUsage[index])
			}
		}
	}
	checks := []struct {
		layer *editorcomponent.LayerData
		x     int
		y     int
		want  int
		path  string
	}{
		{firstLayer, 2, 3, 1, "terrain.png"},
		{firstLayer, 3, 3, 0, ""},
		{firstLayer, 4, 3, 2, "terrain.png"},
		{firstLayer, 2, 4, 3, "terrain.png"},
		{firstLayer, 3, 4, 4, "terrain.png"},
		{firstLayer, 4, 4, 5, "terrain.png"},
		{secondLayer, 2, 3, 6, "detail.png"},
		{secondLayer, 3, 3, 7, "detail.png"},
		{secondLayer, 4, 3, 8, "detail.png"},
		{secondLayer, 2, 4, 10, "detail.png"},
		{secondLayer, 3, 4, 11, "detail.png"},
		{secondLayer, 4, 4, 12, "detail.png"},
	}
	for _, check := range checks {
		index := cellIndex(meta, check.x, check.y)
		if check.layer.Tiles[index] != check.want {
			t.Fatalf("expected %d at (%d,%d), got %d", check.want, check.x, check.y, check.layer.Tiles[index])
		}
		usage := check.layer.TilesetUsage[index]
		if check.path == "" {
			if usage != nil {
				t.Fatalf("expected empty destination usage at (%d,%d), got %+v", check.x, check.y, usage)
			}
			continue
		}
		if usage == nil || usage.Path != check.path || usage.Index != check.want {
			t.Fatalf("expected usage %s/%d at (%d,%d), got %+v", check.path, check.want, check.x, check.y, usage)
		}
	}
	_, entities, _ := entitiesState(w)
	if entities.Items[0].X != 2*TileSize || entities.Items[0].Y != 3*TileSize {
		t.Fatalf("expected selected entity moved to (%d,%d), got (%d,%d)", 2*TileSize, 3*TileSize, entities.Items[0].X, entities.Items[0].Y)
	}
	if entities.Items[1].X != 224 || entities.Items[1].Y != 224 {
		t.Fatalf("expected outside entity unchanged, got (%d,%d)", entities.Items[1].X, entities.Items[1].Y)
	}
	_, session, _ := sessionState(w)
	if session.Status != "Moved room" {
		t.Fatalf("expected moved room status, got %q", session.Status)
	}
	_, undo, _ := undoState(w)
	if len(undo.Snapshots) != 1 {
		t.Fatalf("expected one undo snapshot, got %d", len(undo.Snapshots))
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
