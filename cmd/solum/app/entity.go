package app

import (
	"encoding/json"
	"fmt"
	"strings"

	coreio "github.com/milk9111/sidescroller/internal/editorcore/io"
	corelevelops "github.com/milk9111/sidescroller/internal/editorcore/levelops"
	coremodel "github.com/milk9111/sidescroller/internal/editorcore/model"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/prefabs"
)

const (
	entityComponentsKey = "components"
	entityTileSize      = 32
)

type EntityClipboardState struct {
	Entity levels.Entity
	Valid  bool
}

func (s *State) selectPrefab(path string) {
	info := prefabInfoByPath(s.Prefabs, path, "")
	if info == nil {
		s.Status = "Select a valid prefab"
		return
	}
	s.PrefabPlacement.SelectedPath = info.Path
	s.PrefabPlacement.SelectedType = info.EntityType
	s.SelectedEntity = -1
	s.syncEntityDerivedState()
	s.Status = "Selected prefab " + info.Name
}

func (s *State) selectEntity(index int) {
	if index < 0 || index >= len(s.Document.Entities) {
		s.Status = "Select a valid entity"
		return
	}
	if !entitySelectableOnCurrentLayer(s.Document.Entities[index], s.CurrentLayer) {
		s.Status = "Entity is not on the current layer"
		return
	}
	s.SelectedEntity = index
	s.PrefabPlacement = PrefabPlacementState{}
	s.syncEntityDerivedState()
	s.Status = "Selected entity"
}

func (s *State) clearEntitySelection() {
	s.SelectedEntity = -1
	s.syncEntityDerivedState()
	s.Status = "Cleared entity selection"
}

func (s *State) placeSelectedPrefab(cellX, cellY int) {
	if strings.TrimSpace(s.PrefabPlacement.SelectedType) == "" {
		s.Status = "Select a prefab before placing"
		return
	}
	if cellX < 0 || cellY < 0 {
		s.Status = "Placement coordinates must be zero or greater"
		return
	}
	s.pushSnapshot("entity-place")
	props := map[string]interface{}{
		"layer":  s.CurrentLayer,
		"prefab": s.PrefabPlacement.SelectedPath,
	}
	s.Document.Entities = append(s.Document.Entities, levels.Entity{
		Type:  s.PrefabPlacement.SelectedType,
		X:     cellX * entityTileSize,
		Y:     cellY * entityTileSize,
		Props: props,
	})
	s.Dirty = true
	s.SelectedEntity = len(s.Document.Entities) - 1
	s.PrefabPlacement = PrefabPlacementState{}
	s.PlacementCellXInput = fmt.Sprintf("%d", cellX)
	s.PlacementCellYInput = fmt.Sprintf("%d", cellY)
	s.syncEntityDerivedState()
	s.Status = "Placed entity " + s.Document.Entities[s.SelectedEntity].Type
}

func (s *State) deleteSelectedEntity() {
	if s.SelectedEntity < 0 || s.SelectedEntity >= len(s.Document.Entities) {
		s.Status = "Select an entity to delete"
		return
	}
	s.pushSnapshot("entity-delete")
	s.Document.Entities = append(s.Document.Entities[:s.SelectedEntity], s.Document.Entities[s.SelectedEntity+1:]...)
	if s.SelectedEntity >= len(s.Document.Entities) {
		s.SelectedEntity = len(s.Document.Entities) - 1
	}
	s.Dirty = true
	s.syncEntityDerivedState()
	s.Status = "Deleted entity"
}

func (s *State) copySelectedEntity() {
	if s.SelectedEntity < 0 || s.SelectedEntity >= len(s.Document.Entities) {
		s.Status = "Select an entity to copy"
		return
	}
	s.Clipboard.Entity = cloneEntity(s.Document.Entities[s.SelectedEntity])
	s.Clipboard.Valid = true
	s.Status = "Copied entity"
}

func (s *State) pasteCopiedEntity() {
	if !s.Clipboard.Valid {
		s.Status = "Copy an entity first"
		return
	}
	s.pushSnapshot("entity-paste")
	copy := cloneEntity(s.Clipboard.Entity)
	s.Document.Entities = append(s.Document.Entities, copy)
	if corelevelops.EnsureUniqueEntityIDs(&s.Document) {
		s.syncDerivedState(false)
	}
	s.SelectedEntity = len(s.Document.Entities) - 1
	s.PrefabPlacement = PrefabPlacementState{}
	s.Dirty = true
	s.syncEntityDerivedState()
	s.Status = "Pasted entity"
}

func (s *State) editInspectorField(componentName, fieldName, rawValue string) {
	if s.SelectedEntity < 0 || s.SelectedEntity >= len(s.Document.Entities) {
		s.Status = "Select an entity to edit"
		return
	}
	selected := &s.Document.Entities[s.SelectedEntity]
	prefab := prefabInfoForEntity(s.Prefabs, *selected)
	if prefab == nil {
		s.Status = "Selected entity has no prefab-backed inspector"
		return
	}
	parsed, ok := parseInspectorFieldValue(prefab, componentName, fieldName, rawValue)
	if !ok {
		s.Status = "Inspector value is invalid"
		return
	}
	s.pushSnapshot("entity-inspector")
	componentValues := ensureEntityComponentOverrideValues(selected, componentName)
	componentValues[fieldName] = parsed
	syncInspectorFieldToEntity(selected, componentName, fieldName, parsed)
	s.Dirty = true
	s.syncEntityDerivedState()
	s.Status = "Updated entity component"
}

func (s *State) convertSelectedEntityToPrefab(target string) {
	if s.SelectedEntity < 0 || s.SelectedEntity >= len(s.Document.Entities) {
		s.Status = "Select an entity to convert"
		return
	}
	item := &s.Document.Entities[s.SelectedEntity]
	spec := buildPrefabSpecFromEntity(s.Prefabs, *item)
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = strings.TrimSpace(item.Type)
	}
	normalized, err := coreio.SavePrefab(s.WorkspaceRoot, target, spec)
	if err != nil {
		s.Status = fmt.Sprintf("Convert prefab failed: %v", err)
		return
	}
	s.pushSnapshot("entity-convert-prefab")
	props := ensureEntityProps(item)
	props["prefab"] = normalized
	delete(props, entityComponentsKey)
	items, err := coreio.ScanPrefabCatalog(s.WorkspaceRoot)
	if err != nil {
		s.Dirty = true
		s.syncEntityDerivedState()
		s.Status = fmt.Sprintf("Prefab saved but refresh failed: %v", err)
		return
	}
	s.Prefabs = items
	s.ConvertPrefabTarget = normalized
	s.Dirty = true
	s.syncEntityDerivedState()
	s.Status = "Created prefabs/" + normalized
}

func (s *State) syncEntityDerivedState() {
	if s == nil {
		return
	}
	if s.SelectedEntity >= len(s.Document.Entities) {
		s.SelectedEntity = len(s.Document.Entities) - 1
	}
	if s.SelectedEntity < -1 {
		s.SelectedEntity = -1
	}
	if s.SelectedEntity >= 0 && !entitySelectableOnCurrentLayer(s.Document.Entities[s.SelectedEntity], s.CurrentLayer) {
		s.SelectedEntity = -1
	}
	s.Inspector = buildInspectorState(s.Prefabs, s.Document.Entities, s.SelectedEntity)
	s.InspectorInputs = make(map[string]*string)
	for _, section := range s.Inspector.Sections {
		for _, field := range section.Fields {
			value := field.Value
			s.InspectorInputs[inspectorFieldKey(field.Component, field.Field)] = &value
		}
	}
	if s.SelectedEntity >= 0 && s.SelectedEntity < len(s.Document.Entities) {
		entity := s.Document.Entities[s.SelectedEntity]
		s.PlacementCellXInput = fmt.Sprintf("%d", entity.X/entityTileSize)
		s.PlacementCellYInput = fmt.Sprintf("%d", entity.Y/entityTileSize)
	} else {
		if strings.TrimSpace(s.PlacementCellXInput) == "" {
			s.PlacementCellXInput = "0"
		}
		if strings.TrimSpace(s.PlacementCellYInput) == "" {
			s.PlacementCellYInput = "0"
		}
	}
	if strings.TrimSpace(s.ConvertPrefabTarget) == "" && s.SelectedEntity >= 0 && s.SelectedEntity < len(s.Document.Entities) {
		entity := s.Document.Entities[s.SelectedEntity]
		label := strings.TrimSpace(entity.Type)
		if label == "" {
			label = strings.TrimSuffix(prefabPathForEntity(entity), ".yaml")
		}
		if label != "" {
			s.ConvertPrefabTarget = label + ".yaml"
		}
	}
	if s.SelectedEntity == -1 && s.Inspector.Active == false {
		s.ConvertPrefabTarget = strings.TrimSpace(s.ConvertPrefabTarget)
	}
}

func currentLayerEntityIndexes(doc coremodel.LevelDocument, currentLayer int) []int {
	indexes := make([]int, 0)
	for index, item := range doc.Entities {
		if entitySelectableOnCurrentLayer(item, currentLayer) {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func entitySelectableOnCurrentLayer(item levels.Entity, currentLayer int) bool {
	layerIndex, ok := corelevelops.EntityLayerIndex(item.Props)
	if !ok {
		return currentLayer == 0
	}
	return layerIndex == currentLayer
}

func cloneEntity(item levels.Entity) levels.Entity {
	copy := item
	copy.Props = cloneEntityProps(item.Props)
	return copy
}

func cloneEntityProps(props map[string]interface{}) map[string]interface{} {
	if props == nil {
		return nil
	}
	encoded, err := json.Marshal(props)
	if err != nil {
		fallback := make(map[string]interface{}, len(props))
		for key, value := range props {
			fallback[key] = value
		}
		return fallback
	}
	var cloned map[string]interface{}
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		fallback := make(map[string]interface{}, len(props))
		for key, value := range props {
			fallback[key] = value
		}
		return fallback
	}
	return cloned
}

func ensureEntityProps(item *levels.Entity) map[string]interface{} {
	if item == nil {
		return nil
	}
	if item.Props == nil {
		item.Props = make(map[string]interface{})
	}
	return item.Props
}

func prefabPathForEntity(item levels.Entity) string {
	if item.Props != nil {
		if prefabPath, ok := item.Props["prefab"].(string); ok && strings.TrimSpace(prefabPath) != "" {
			return strings.TrimSpace(prefabPath)
		}
	}
	if strings.TrimSpace(item.Type) == "" {
		return ""
	}
	return strings.TrimSpace(item.Type) + ".yaml"
}

func prefabInfoForEntity(catalog []coreio.PrefabInfo, item levels.Entity) *coreio.PrefabInfo {
	return prefabInfoByPath(catalog, prefabPathForEntity(item), item.Type)
}

func prefabInfoByPath(catalog []coreio.PrefabInfo, path, entityType string) *coreio.PrefabInfo {
	cleanPath := strings.TrimSpace(path)
	cleanType := strings.TrimSpace(entityType)
	for index := range catalog {
		item := &catalog[index]
		if cleanPath != "" && item.Path == cleanPath {
			return item
		}
	}
	for index := range catalog {
		item := &catalog[index]
		if cleanType != "" && item.EntityType == cleanType {
			return item
		}
	}
	return nil
}

func entityComponentOverrides(props map[string]interface{}) map[string]any {
	if props == nil {
		return nil
	}
	raw, ok := props[entityComponentsKey]
	if !ok || raw == nil {
		return nil
	}
	if typed, ok := raw.(map[string]interface{}); ok {
		converted := make(map[string]any, len(typed))
		for key, value := range typed {
			converted[key] = value
		}
		return converted
	}
	return nil
}

func ensureEntityComponentOverrideValues(item *levels.Entity, componentName string) map[string]any {
	props := ensureEntityProps(item)
	raw, ok := props[entityComponentsKey]
	if !ok || raw == nil {
		overrides := make(map[string]any)
		props[entityComponentsKey] = overrides
		componentValues := make(map[string]any)
		overrides[componentName] = componentValues
		return componentValues
	}
	overrides, ok := raw.(map[string]any)
	if !ok {
		if converted, convertedOK := raw.(map[string]interface{}); convertedOK {
			overrides = make(map[string]any, len(converted))
			for key, value := range converted {
				overrides[key] = value
			}
			props[entityComponentsKey] = overrides
		} else {
			overrides = make(map[string]any)
			props[entityComponentsKey] = overrides
		}
	}
	if rawComponent, ok := overrides[componentName]; ok && rawComponent != nil {
		if typed, typedOK := rawComponent.(map[string]interface{}); typedOK {
			converted := make(map[string]any, len(typed))
			for key, value := range typed {
				converted[key] = value
			}
			overrides[componentName] = converted
			return converted
		}
		if typed, typedOK := rawComponent.(map[string]any); typedOK {
			return typed
		}
	}
	componentValues := make(map[string]any)
	overrides[componentName] = componentValues
	return componentValues
}

func buildPrefabSpecFromEntity(catalog []coreio.PrefabInfo, item levels.Entity) prefabs.EntityBuildSpec {
	components := map[string]any{}
	if prefab := prefabInfoForEntity(catalog, item); prefab != nil {
		components = coreio.MergeComponentMaps(prefab.Components, entityComponentOverrides(item.Props))
	} else {
		components = coreio.MergeComponentMaps(nil, entityComponentOverrides(item.Props))
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
		if typed, typedOK := existing.(map[string]any); typedOK {
			return typed
		}
		if typed, typedOK := existing.(map[string]interface{}); typedOK {
			converted := make(map[string]any, len(typed))
			for key, value := range typed {
				converted[key] = value
			}
			components[name] = converted
			return converted
		}
	}
	created := map[string]any{}
	components[name] = created
	return created
}
