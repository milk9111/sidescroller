package editorsystem

import (
	"strings"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
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
				Tiles:        make([]int, cellCount),
				TilesetUsage: make([]*levels.TileInfo, cellCount),
			})
			session.CurrentLayer = order
			setDirty(w, true)
			session.Status = "Added layer"
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

var _ ecs.System = (*EditorLayerSystem)(nil)
