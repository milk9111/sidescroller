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
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Layer 1", Order: 0, Tiles: []int{0}, TilesetUsage: make([]*levels.TileInfo, 1)})

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
