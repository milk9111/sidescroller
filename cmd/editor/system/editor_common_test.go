package editorsystem

import (
	"strconv"
	"testing"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

func TestEnsureUniqueEntityIDsRepairsDuplicatesAndTransitionProps(t *testing.T) {
	items := []levels.Entity{
		{ID: "enemy_1", Type: "enemy"},
		{ID: "enemy_1", Type: "enemy"},
		{Type: "transition", Props: map[string]interface{}{"id": "t4"}},
		{Type: "transition", Props: map[string]interface{}{"id": "t4", "linked_id": "t4"}},
		{Type: "gate"},
	}

	if !ensureUniqueEntityIDs(items) {
		t.Fatal("expected entity IDs to be rewritten")
	}
	seen := map[string]struct{}{}
	for _, item := range items {
		if item.ID == "" {
			t.Fatalf("expected non-empty ID for %+v", item)
		}
		if _, exists := seen[item.ID]; exists {
			t.Fatalf("expected unique ID, got duplicate %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		if isTransitionEntity(item) {
			if got := entityStringProp(item, "id"); got != item.ID {
				t.Fatalf("expected transition prop id %q, got %q", item.ID, got)
			}
		}
	}
}

func TestPushSnapshotCapsUndoDepth(t *testing.T) {
	w := ecs.NewWorld()
	entity := ecs.CreateEntity(w)
	_ = ecs.Add(w, entity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{})
	_ = ecs.Add(w, entity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 1, Height: 1})
	_ = ecs.Add(w, entity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{})
	_ = ecs.Add(w, entity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})
	layerEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Layer 1", Order: 0, Active: true, Tiles: []int{0}, TilesetUsage: make([]*levels.TileInfo, 1)})

	for index := 0; index < 105; index++ {
		pushSnapshot(w, strconv.Itoa(index))
	}

	_, undo, _ := undoState(w)
	if len(undo.Snapshots) != 100 {
		t.Fatalf("expected undo depth 100, got %d", len(undo.Snapshots))
	}
	if undo.Snapshots[0].StatusMessage != "5" {
		t.Fatalf("expected oldest retained snapshot to be 5, got %q", undo.Snapshots[0].StatusMessage)
	}
	if undo.Snapshots[len(undo.Snapshots)-1].StatusMessage != "104" {
		t.Fatalf("expected latest snapshot to be 104, got %q", undo.Snapshots[len(undo.Snapshots)-1].StatusMessage)
	}
}

func TestEntityBoundsKeepsSpikeAnchoredToCellWithCenteredOrigin(t *testing.T) {
	prefab := &editorio.PrefabInfo{Preview: editorio.PrefabPreview{FrameW: 32, FrameH: 32, CenterOrigin: true}}
	item := levels.Entity{Type: "spike", X: 64, Y: 96}

	left, top, width, height := entityBounds(item, prefab)
	if left != 64 || top != 96 {
		t.Fatalf("expected spike bounds anchored at cell top-left, got (%v,%v)", left, top)
	}
	if width != 32 || height != 32 {
		t.Fatalf("expected 32x32 spike bounds, got %vx%v", width, height)
	}
	anchorX, anchorY := entityAnchorPosition(item, 16, 16)
	if anchorX != 80 || anchorY != 112 {
		t.Fatalf("expected centered render anchor at (80,112), got (%v,%v)", anchorX, anchorY)
	}
}

func TestEntityBoundsApplyPrefabTransformScale(t *testing.T) {
	prefab := &editorio.PrefabInfo{Preview: editorio.PrefabPreview{FrameW: 32, FrameH: 16, ScaleX: 2, ScaleY: 0.5}}
	item := levels.Entity{Type: "heap", X: 64, Y: 96}

	left, top, width, height := entityBounds(item, prefab)
	if left != 64 || top != 96 {
		t.Fatalf("expected scaled bounds to stay anchored at top-left, got (%v,%v)", left, top)
	}
	if width != 64 || height != 8 {
		t.Fatalf("expected scaled bounds 64x8, got %vx%v", width, height)
	}
}

func TestEntityBoundsCoverFullCenteredSpriteFrame(t *testing.T) {
	prefab := &editorio.PrefabInfo{Preview: editorio.PrefabPreview{FrameW: 256, FrameH: 256, CenterOrigin: true}}
	item := levels.Entity{Type: "background", X: 64, Y: 96}

	left, top, width, height := entityBounds(item, prefab)
	if left != -64 || top != -32 {
		t.Fatalf("expected centered sprite bounds origin at (-64,-32), got (%v,%v)", left, top)
	}
	if width != 256 || height != 256 {
		t.Fatalf("expected full sprite bounds 256x256, got %vx%v", width, height)
	}
}

func TestLayerCellOccupiedTreatsZeroIndexTileAsFilledWhenUsageExists(t *testing.T) {
	layer := &editorcomponent.LayerData{
		Active:       true,
		Physics:      true,
		Tiles:        []int{0},
		TilesetUsage: []*levels.TileInfo{{Path: "terrain.png", Index: 0, TileW: 32, TileH: 32}},
	}
	if !layerCellOccupied(layer, 0) {
		t.Fatal("expected zero-index tile with usage metadata to count as occupied")
	}
}

func TestSolidCellAtTreatsZeroIndexTileAsSolidWhenUsageExists(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 1, Height: 1})
	layerEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{
		Name:         "Physics",
		Order:        0,
		Active:       true,
		Physics:      true,
		Tiles:        []int{0},
		TilesetUsage: []*levels.TileInfo{{Path: "terrain.png", Index: 0, TileW: 32, TileH: 32}},
	})

	_, meta, _ := levelMetaState(w)
	if !solidCellAt(w, meta, 0, 0) {
		t.Fatal("expected solidCellAt to treat zero-index tile with usage metadata as solid")
	}
}

func TestBreakableWallRotationForCellFacesOpenNeighbor(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 3, Height: 3})
	layerEntity := ecs.CreateEntity(w)
	tiles := []int{
		1, 0, 1,
		1, 0, 1,
		1, 1, 1,
	}
	usage := make([]*levels.TileInfo, len(tiles))
	for index, value := range tiles {
		if value != 0 {
			usage[index] = &levels.TileInfo{Path: "terrain.png", Index: 0, TileW: 32, TileH: 32}
		}
	}
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{
		Name:         "Physics",
		Order:        0,
		Active:       true,
		Physics:      true,
		Tiles:        tiles,
		TilesetUsage: usage,
	})
	_, meta, _ := levelMetaState(w)
	item := levels.Entity{Type: "breakable_wall", X: TileSize, Y: TileSize, Props: map[string]interface{}{"w": float64(TileSize), "h": float64(TileSize)}}

	rotation := breakableWallRotationForCell(w, meta, item, 1, 1)
	if rotation != 0 {
		t.Fatalf("expected breakable wall tile to face upward opening, got %v", rotation)
	}

	w = ecs.NewWorld()
	sessionEntity = ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 3, Height: 3})
	tiles = []int{
		1, 1, 1,
		1, 0, 0,
		1, 1, 1,
	}
	usage = make([]*levels.TileInfo, len(tiles))
	for index, value := range tiles {
		if value != 0 {
			usage[index] = &levels.TileInfo{Path: "terrain.png", Index: 0, TileW: 32, TileH: 32}
		}
	}
	layerEntity = ecs.CreateEntity(w)
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{
		Name:         "Physics",
		Order:        0,
		Active:       true,
		Physics:      true,
		Tiles:        tiles,
		TilesetUsage: usage,
	})
	_, meta, _ = levelMetaState(w)
	item = levels.Entity{Type: "breakable_wall", X: TileSize, Y: TileSize, Props: map[string]interface{}{"w": float64(TileSize), "h": float64(TileSize)}}
	rotation = breakableWallRotationForCell(w, meta, item, 1, 1)
	if rotation != 90 {
		t.Fatalf("expected breakable wall tile to face right opening, got %v", rotation)
	}
}
