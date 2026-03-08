package editorsystem

import (
	"testing"

	editorautotile "github.com/milk9111/sidescroller/cmd/editor/autotile"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

func TestEditorAutotileSystemRecomputesDirtyCells(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 3, Height: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.AutotileStateComponent.Kind(), &editorcomponent.AutotileState{Enabled: true, DirtyCells: map[int]map[int]struct{}{0: {0: {}, 1: {}, 2: {}}}, FullRebuild: make(map[int]bool)})

	usage := func() *levels.TileInfo {
		return &levels.TileInfo{Path: "grass.png", TileW: 32, TileH: 32, Auto: true, BaseIndex: 0}
	}
	entity := ecs.CreateEntity(w)
	_ = ecs.Add(w, entity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{
		Name:  "Ground",
		Order: 0,
		Tiles: []int{0, 0, 0},
		TilesetUsage: []*levels.TileInfo{
			usage(), usage(), usage(),
		},
	})

	NewEditorAutotileSystem().Update(w)

	_, layer, _ := layerAt(w, 0)
	left := layer.TilesetUsage[0]
	middle := layer.TilesetUsage[1]
	right := layer.TilesetUsage[2]
	leftMask := editorautotile.BuildMask(true, true, true, true, true, true, true, true)
	middleMask := editorautotile.BuildMask(true, true, true, true, true, true, true, true)
	rightMask := editorautotile.BuildMask(true, true, true, true, true, true, true, true)
	if left.Mask != leftMask || middle.Mask != middleMask || right.Mask != rightMask {
		t.Fatalf("unexpected masks: left=%08b middle=%08b right=%08b", left.Mask, middle.Mask, right.Mask)
	}
	leftOffset, _ := editorautotile.DefaultOffset(leftMask)
	middleOffset, _ := editorautotile.DefaultOffset(middleMask)
	rightOffset, _ := editorautotile.DefaultOffset(rightMask)
	if left.Index != leftOffset || middle.Index != middleOffset || right.Index != rightOffset {
		t.Fatalf("unexpected indices: left=%d middle=%d right=%d", left.Index, middle.Index, right.Index)
	}
}

func TestEditorAutotileSystemTreatsLevelBoundsAsConnectedNeighbors(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 1, Height: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.AutotileStateComponent.Kind(), &editorcomponent.AutotileState{Enabled: true, DirtyCells: map[int]map[int]struct{}{0: {0: {}}}, FullRebuild: make(map[int]bool)})

	usage := &levels.TileInfo{Path: "grass.png", TileW: 32, TileH: 32, Auto: true, BaseIndex: 0}
	entity := ecs.CreateEntity(w)
	_ = ecs.Add(w, entity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{
		Name:  "Ground",
		Order: 0,
		Tiles: []int{0},
		TilesetUsage: []*levels.TileInfo{
			usage,
		},
	})

	NewEditorAutotileSystem().Update(w)

	_, layer, _ := layerAt(w, 0)
	mask := editorautotile.BuildMask(true, true, true, true, true, true, true, true)
	if layer.TilesetUsage[0].Mask != mask {
		t.Fatalf("expected full boundary mask %08b, got %08b", mask, layer.TilesetUsage[0].Mask)
	}
	offset, _ := editorautotile.DefaultOffset(mask)
	if layer.TilesetUsage[0].Index != offset {
		t.Fatalf("expected index %d, got %d", offset, layer.TilesetUsage[0].Index)
	}
}
