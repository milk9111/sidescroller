package editorsystem

import (
	editorautotile "github.com/milk9111/sidescroller/cmd/editor/autotile"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

type EditorAutotileSystem struct{}

func NewEditorAutotileSystem() *EditorAutotileSystem {
	return &EditorAutotileSystem{}
}

func (s *EditorAutotileSystem) Update(w *ecs.World) {
	_, state, ok := autotileState(w)
	if !ok || state == nil {
		return
	}
	if _, actions, ok := actionState(w); ok && actions != nil && actions.ToggleAutotile {
		actions.ToggleAutotile = false
		state.Enabled = !state.Enabled
		if _, session, ok := sessionState(w); ok && session != nil {
			session.SelectedTile = session.SelectedTile.Normalize()
			if state.Enabled {
				session.SelectedTile.Index = 0
				session.Status = "Autotile enabled"
			} else {
				session.Status = "Autotile disabled"
			}
		}
	}
	if !state.Enabled {
		clearAutotileQueues(state)
		return
	}
	_, meta, _ := levelMetaState(w)
	for layerIndex := range state.FullRebuild {
		s.recomputeFullLayer(w, meta, layerIndex, state)
	}
	for layerIndex, cells := range state.DirtyCells {
		if state.FullRebuild[layerIndex] {
			continue
		}
		for index := range cells {
			s.recomputeCell(w, meta, layerIndex, index, state)
		}
	}
	clearAutotileQueues(state)
}

func (s *EditorAutotileSystem) recomputeFullLayer(w *ecs.World, meta *editorcomponent.LevelMeta, layerIndex int, state *editorcomponent.AutotileState) {
	_, layer, ok := layerAt(w, layerIndex)
	if !ok || layer == nil || meta == nil {
		return
	}
	for index, usage := range layer.TilesetUsage {
		if usage == nil || !usage.Auto {
			continue
		}
		s.applyComputedUsage(layer, meta, index, usage, state)
	}
}

func (s *EditorAutotileSystem) recomputeCell(w *ecs.World, meta *editorcomponent.LevelMeta, layerIndex, index int, state *editorcomponent.AutotileState) {
	_, layer, ok := layerAt(w, layerIndex)
	if !ok || layer == nil || meta == nil || index < 0 || index >= len(layer.TilesetUsage) {
		return
	}
	usage := layer.TilesetUsage[index]
	if usage == nil || !usage.Auto {
		return
	}
	s.applyComputedUsage(layer, meta, index, usage, state)
}

func (s *EditorAutotileSystem) applyComputedUsage(layer *editorcomponent.LayerData, meta *editorcomponent.LevelMeta, index int, usage *levels.TileInfo, state *editorcomponent.AutotileState) {
	if usage == nil || !usage.Auto || meta == nil {
		return
	}
	cellX := index % meta.Width
	cellY := index / meta.Width
	mask := autotileMaskFor(layer, meta, cellX, cellY, usage)
	offset, ok := editorautotile.ResolveOffset(mask, state.Remap)
	if !ok {
		offset = 0
	}
	usage.Mask = mask
	usage.Index = usage.BaseIndex + offset
	layer.Tiles[index] = usage.Index
}

var _ ecs.System = (*EditorAutotileSystem)(nil)
