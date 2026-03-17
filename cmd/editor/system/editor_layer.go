package editorsystem

import (
	"fmt"
	"strings"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

type EditorLayerSystem struct{}

func NewEditorLayerSystem() *EditorLayerSystem {
	return &EditorLayerSystem{}
}

func (s *EditorLayerSystem) Update(w *ecs.World) {
	_, actions, ok := actionState(w)
	if !ok || actions == nil {
		return
	}
	_, session, ok := sessionState(w)
	if !ok || session == nil {
		return
	}
	_, meta, _ := levelMetaState(w)

	if actions.SelectLayer >= 0 {
		session.CurrentLayer = actions.SelectLayer
		clampCurrentLayer(w, session)
		actions.SelectLayer = -1
	}

	if actions.ApplyLevelResize {
		actions.ApplyLevelResize = false
		if meta == nil {
			session.Status = "Level metadata unavailable"
		} else if actions.ResizeWidth <= 0 || actions.ResizeHeight <= 0 {
			session.Status = "Level size must be at least 1x1"
		} else if actions.ResizeWidth == meta.Width && actions.ResizeHeight == meta.Height {
			session.Status = fmt.Sprintf("Level size unchanged (%dx%d)", meta.Width, meta.Height)
		} else {
			pushSnapshot(w, "level-resize")
			resizeLevelContents(w, meta, actions.ResizeWidth, actions.ResizeHeight)
			setDirty(w, true)
			session.Status = fmt.Sprintf("Resized level to %dx%d", actions.ResizeWidth, actions.ResizeHeight)
		}
	}

	if actions.AddLayer {
		actions.AddLayer = false
		if meta != nil {
			pushSnapshot(w, "layer-add")
			cellCount := meta.Width * meta.Height
			entity := ecs.CreateEntity(w)
			order := len(layerEntities(w))
			_ = ecs.Add(w, entity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{
				Name:         nextLayerName(w),
				Order:        order,
				Active:       true,
				Tiles:        make([]int, cellCount),
				TilesetUsage: make([]*levels.TileInfo, cellCount),
			})
			session.CurrentLayer = order
			setDirty(w, true)
			session.Status = "Added layer"
		}
	}

	if actions.DeleteCurrentLayer {
		actions.DeleteCurrentLayer = false
		layers := layerEntities(w)
		if len(layers) <= 1 {
			session.Status = "Cannot delete last layer"
		} else {
			pushSnapshot(w, "layer-delete")
			current := clampInt(session.CurrentLayer, 0, len(layers)-1)
			if deleteLayerAndContents(w, current) {
				clampCurrentLayer(w, session)
				setDirty(w, true)
				session.Status = "Deleted layer"
			}
		}
	}

	if actions.MoveLayerDelta != 0 {
		delta := actions.MoveLayerDelta
		actions.MoveLayerDelta = 0
		layers := layerEntities(w)
		current := clampInt(session.CurrentLayer, 0, maxInt(0, len(layers)-1))
		if len(layers) > 0 {
			next := clampInt(current+delta, 0, len(layers)-1)
			if next != current {
				pushSnapshot(w, "layer-move")
				mapping := reorderLayerEntities(w, current, next)
				if _, entities, ok := entitiesState(w); ok && entities != nil {
					remapEntityLayerProps(entities.Items, mapping)
				}
				session.CurrentLayer = next
				setDirty(w, true)
				session.Status = "Moved layer"
			}
		}
	}

	if actions.ApplyRename {
		actions.ApplyRename = false
		if _, layer, ok := layerAt(w, session.CurrentLayer); ok && layer != nil {
			name := strings.TrimSpace(actions.RenameLayer)
			if name != "" && name != layer.Name {
				pushSnapshot(w, "layer-rename")
				layer.Name = name
				setDirty(w, true)
				session.Status = "Renamed layer"
			}
		}
	}

	if actions.ToggleLayerPhysics {
		actions.ToggleLayerPhysics = false
		if _, layer, ok := layerAt(w, session.CurrentLayer); ok && layer != nil {
			pushSnapshot(w, "layer-physics")
			layer.Physics = !layer.Physics
			setDirty(w, true)
			if layer.Physics {
				session.Status = "Layer physics enabled"
			} else {
				session.Status = "Layer physics disabled"
			}
		}
	}

	if actions.ToggleLayerActive {
		actions.ToggleLayerActive = false
		if _, layer, ok := layerAt(w, session.CurrentLayer); ok && layer != nil {
			pushSnapshot(w, "layer-active")
			layer.Active = !layer.Active
			setDirty(w, true)
			if layer.Active {
				session.Status = "Layer activated"
			} else {
				session.Status = "Layer deactivated"
			}
		}
	}

	if actions.ToggleLayerVisibility {
		actions.ToggleLayerVisibility = false
		if _, layer, ok := layerAt(w, session.CurrentLayer); ok && layer != nil {
			layer.Hidden = !layer.Hidden
			if layer.Hidden {
				session.Status = "Layer hidden"
			} else {
				session.Status = "Layer shown"
			}
		}
	}

	if actions.TogglePhysicsHighlight {
		actions.TogglePhysicsHighlight = false
		session.PhysicsHighlight = !session.PhysicsHighlight
		if session.PhysicsHighlight {
			session.Status = "Physics highlight enabled"
		} else {
			session.Status = "Physics highlight disabled"
		}
	}

	clampCurrentLayer(w, session)
}

func resizeLevelContents(w *ecs.World, meta *editorcomponent.LevelMeta, newWidth, newHeight int) {
	if w == nil || meta == nil || newWidth <= 0 || newHeight <= 0 {
		return
	}
	oldWidth := meta.Width
	oldHeight := meta.Height
	for _, entity := range layerEntities(w) {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if layer == nil {
			continue
		}
		resizeLayerData(layer, oldWidth, oldHeight, newWidth, newHeight)
	}
	meta.Width = newWidth
	meta.Height = newHeight
	trimEntitiesToLevelBounds(w, newWidth, newHeight)
	resetTransientResizeState(w)
	markResizeDependentStateDirty(w)
}

func resizeLayerData(layer *editorcomponent.LayerData, oldWidth, oldHeight, newWidth, newHeight int) {
	if layer == nil || oldWidth <= 0 || oldHeight <= 0 || newWidth <= 0 || newHeight <= 0 {
		return
	}
	resizedTiles := make([]int, newWidth*newHeight)
	resizedUsage := make([]*levels.TileInfo, newWidth*newHeight)
	copyWidth := minInt(oldWidth, newWidth)
	copyHeight := minInt(oldHeight, newHeight)
	for y := 0; y < copyHeight; y++ {
		for x := 0; x < copyWidth; x++ {
			oldIndex := y*oldWidth + x
			newIndex := y*newWidth + x
			if oldIndex >= 0 && oldIndex < len(layer.Tiles) {
				resizedTiles[newIndex] = layer.Tiles[oldIndex]
			}
			if oldIndex >= 0 && oldIndex < len(layer.TilesetUsage) {
				resizedUsage[newIndex] = layer.TilesetUsage[oldIndex]
			}
		}
	}
	layer.Tiles = resizedTiles
	layer.TilesetUsage = resizedUsage
}

func trimEntitiesToLevelBounds(w *ecs.World, width, height int) {
	_, entities, ok := entitiesState(w)
	if !ok || entities == nil {
		return
	}
	_, selection, _ := entitySelectionState(w)
	_, catalog, _ := prefabCatalogState(w)
	selectedIndex := -1
	hoveredIndex := -1
	if selection != nil {
		selectedIndex = selection.SelectedIndex
		hoveredIndex = selection.HoveredIndex
	}
	filtered := make([]levels.Entity, 0, len(entities.Items))
	newSelected := -1
	newHovered := -1
	for index, item := range entities.Items {
		if !entityInsideLevelBounds(width, height, item, prefabInfoForEntity(catalog, item)) {
			continue
		}
		if index == selectedIndex {
			newSelected = len(filtered)
		}
		if index == hoveredIndex {
			newHovered = len(filtered)
		}
		filtered = append(filtered, item)
	}
	entities.Items = filtered
	if selection != nil {
		selection.SelectedIndex = newSelected
		selection.HoveredIndex = newHovered
		selection.Dragging = false
		selection.DragOffsetCellX = 0
		selection.DragOffsetCellY = 0
		selection.DragSnapshotDone = false
		selection.PropertySnapshotDone = false
	}
}

func entityInsideLevelBounds(width, height int, item levels.Entity, prefab *editorio.PrefabInfo) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	left, top, entityWidth, entityHeight := entityBounds(item, prefab)
	if entityWidth <= 0 || entityHeight <= 0 {
		return false
	}
	maxX := float64(width * TileSize)
	maxY := float64(height * TileSize)
	return left >= 0 && top >= 0 && left+entityWidth <= maxX && top+entityHeight <= maxY
}

func resetTransientResizeState(w *ecs.World) {
	if _, drag, ok := areaDragState(w); ok && drag != nil {
		*drag = editorcomponent.AreaDragState{EntityIndex: -1}
	}
	if _, stroke, ok := strokeState(w); ok && stroke != nil {
		stroke.Active = false
		stroke.Touched = nil
		stroke.Preview = nil
	}
}

func markResizeDependentStateDirty(w *ecs.World) {
	if _, overview, ok := overviewState(w); ok && overview != nil {
		overview.NeedsRefresh = true
	}
	if _, autotile, ok := autotileState(w); ok && autotile != nil {
		clearAutotileQueues(autotile)
		for index := range layerEntities(w) {
			autotile.FullRebuild[index] = true
		}
	}
}

var _ ecs.System = (*EditorLayerSystem)(nil)
