package editorsystem

import (
	"fmt"
	"strings"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/prefabs"
)

type EditorPrefabSystem struct {
	workspaceRoot string
}

func NewEditorPrefabSystem(workspaceRoot string) *EditorPrefabSystem {
	return &EditorPrefabSystem{workspaceRoot: workspaceRoot}
}

func (s *EditorPrefabSystem) Update(w *ecs.World) {
	_, session, ok := sessionState(w)
	if !ok || session == nil {
		return
	}
	_, actions, ok := actionState(w)
	if !ok || actions == nil || !actions.ApplyConvertSelectedEntityToPrefab {
		return
	}
	defer func() {
		actions.ApplyConvertSelectedEntityToPrefab = false
		actions.ConvertSelectedEntityToPrefabName = ""
	}()

	_, entities, ok := entitiesState(w)
	if !ok || entities == nil {
		return
	}
	_, selection, ok := entitySelectionState(w)
	if !ok || selection == nil || selection.SelectedIndex < 0 || selection.SelectedIndex >= len(entities.Items) {
		session.Status = "Select an entity to convert"
		return
	}
	_, catalog, _ := prefabCatalogState(w)
	item := &entities.Items[selection.SelectedIndex]
	spec := buildPrefabSpecFromEntity(catalog, *item)
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = strings.TrimSpace(item.Type)
	}
	normalized, err := editorio.SavePrefab(s.workspaceRoot, actions.ConvertSelectedEntityToPrefabName, spec)
	if err != nil {
		session.Status = fmt.Sprintf("Convert prefab failed: %v", err)
		return
	}
	props := ensureEntityProps(item)
	props["prefab"] = normalized
	delete(props, entityComponentsKey)
	items, err := editorio.ScanPrefabCatalog(s.workspaceRoot)
	if err != nil {
		session.Status = fmt.Sprintf("Prefab saved but refresh failed: %v", err)
		setDirty(w, true)
		return
	}
	if catalog != nil {
		catalog.Items = items
	}
	setDirty(w, true)
	session.Status = "Created prefabs/" + normalized
}

func buildPrefabSpecFromEntity(catalog *editorcomponent.PrefabCatalog, item levels.Entity) prefabs.EntityBuildSpec {
	components := map[string]any{}
	if prefab := prefabInfoForEntity(catalog, item); prefab != nil {
		components = editorio.MergeComponentMaps(prefab.Components, entityComponentOverrides(item.Props))
	} else {
		components = editorio.MergeComponentMaps(nil, entityComponentOverrides(item.Props))
	}
	if components == nil {
		components = map[string]any{}
	}
	mergeTopLevelTransformOverrides(components, item)
	name := strings.TrimSpace(item.Type)
	if name == "" {
		name = strings.TrimSuffix(prefabPathForEntity(item), ".yaml")
	}
	return prefabs.EntityBuildSpec{
		Name:       name,
		Components: components,
	}
}

func mergeTopLevelTransformOverrides(components map[string]any, item levels.Entity) {
	if item.Props == nil {
		return
	}
	transform := ensureComponentMap(components, "transform")
	if raw, ok := item.Props["transform"]; ok {
		if values, ok := raw.(map[string]interface{}); ok {
			for key, value := range values {
				transform[key] = value
			}
		}
	}
	for _, key := range []string{"scale_x", "scale_y", "rotation"} {
		if value, ok := item.Props[key]; ok {
			transform[key] = value
		}
	}
	if len(transform) == 0 {
		delete(components, "transform")
	}
}

func ensureComponentMap(components map[string]any, name string) map[string]any {
	if existing, ok := components[name]; ok {
		switch typed := existing.(type) {
		case map[string]any:
			return typed
		}
	}
	created := map[string]any{}
	components[name] = created
	return created
}

var _ ecs.System = (*EditorPrefabSystem)(nil)
