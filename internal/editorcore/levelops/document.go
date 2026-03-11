package levelops

import (
	"fmt"
	"strings"

	coremodel "github.com/milk9111/sidescroller/internal/editorcore/model"
	"github.com/milk9111/sidescroller/levels"
)

func ClampLayerIndex(doc coremodel.LevelDocument, index int) int {
	if len(doc.Layers) == 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= len(doc.Layers) {
		return len(doc.Layers) - 1
	}
	return index
}

func NextLayerName(doc coremodel.LevelDocument) string {
	existing := make(map[string]struct{}, len(doc.Layers))
	for _, layer := range doc.Layers {
		existing[strings.TrimSpace(layer.Name)] = struct{}{}
	}
	for index := 1; ; index++ {
		candidate := fmt.Sprintf("Layer %d", index)
		if _, ok := existing[candidate]; !ok {
			return candidate
		}
	}
}

func AddLayer(doc *coremodel.LevelDocument) int {
	if doc == nil {
		return -1
	}
	cellCount := doc.Width * doc.Height
	if cellCount < 0 {
		cellCount = 0
	}
	doc.Layers = append(doc.Layers, coremodel.Layer{
		Name:         NextLayerName(*doc),
		Tiles:        make([]int, cellCount),
		TilesetUsage: make([]*levels.TileInfo, cellCount),
	})
	return len(doc.Layers) - 1
}

func DeleteLayer(doc *coremodel.LevelDocument, index int) bool {
	if doc == nil || len(doc.Layers) <= 1 || index < 0 || index >= len(doc.Layers) {
		return false
	}
	doc.Layers = append(doc.Layers[:index], doc.Layers[index+1:]...)
	doc.Entities = deleteEntitiesOnLayer(doc.Entities, index)
	return true
}

func MoveLayer(doc *coremodel.LevelDocument, currentIndex, nextIndex int) bool {
	if doc == nil || len(doc.Layers) == 0 {
		return false
	}
	currentIndex = ClampLayerIndex(*doc, currentIndex)
	nextIndex = ClampLayerIndex(*doc, nextIndex)
	if currentIndex == nextIndex {
		return false
	}
	moving := doc.Layers[currentIndex]
	doc.Layers = append(doc.Layers[:currentIndex], doc.Layers[currentIndex+1:]...)
	if nextIndex >= len(doc.Layers) {
		doc.Layers = append(doc.Layers, moving)
	} else {
		doc.Layers = append(doc.Layers[:nextIndex], append([]coremodel.Layer{moving}, doc.Layers[nextIndex:]...)...)
	}
	mapping := make(map[int]int, len(doc.Layers))
	for index := range doc.Layers {
		mapping[index] = index
	}
	mapping[currentIndex] = nextIndex
	if currentIndex < nextIndex {
		for index := currentIndex + 1; index <= nextIndex; index++ {
			mapping[index] = index - 1
		}
	} else {
		for index := nextIndex; index < currentIndex; index++ {
			mapping[index] = index + 1
		}
	}
	remapEntityLayerProps(doc.Entities, mapping)
	return true
}

func RenameLayer(doc *coremodel.LevelDocument, index int, name string) bool {
	if doc == nil || len(doc.Layers) == 0 {
		return false
	}
	index = ClampLayerIndex(*doc, index)
	name = strings.TrimSpace(name)
	if name == "" || doc.Layers[index].Name == name {
		return false
	}
	doc.Layers[index].Name = name
	return true
}

func ToggleLayerPhysics(doc *coremodel.LevelDocument, index int) (bool, bool) {
	if doc == nil || len(doc.Layers) == 0 {
		return false, false
	}
	index = ClampLayerIndex(*doc, index)
	doc.Layers[index].Physics = !doc.Layers[index].Physics
	return doc.Layers[index].Physics, true
}

func EnsureUniqueEntityIDs(doc *coremodel.LevelDocument) bool {
	if doc == nil {
		return false
	}
	used := make(map[string]struct{}, len(doc.Entities))
	nextByPrefix := make(map[string]int)
	nextTransition := 1
	changed := false
	for index := range doc.Entities {
		candidate := canonicalEntityID(doc.Entities[index])
		if candidate != "" {
			if _, exists := used[candidate]; !exists {
				used[candidate] = struct{}{}
				if syncEntityID(&doc.Entities[index], candidate) {
					changed = true
				}
				continue
			}
		}
		replacement := nextGeneratedEntityID(doc.Entities[index], used, nextByPrefix, &nextTransition)
		used[replacement] = struct{}{}
		if syncEntityID(&doc.Entities[index], replacement) {
			changed = true
		}
	}
	return changed
}

func deleteEntitiesOnLayer(items []levels.Entity, deletedIndex int) []levels.Entity {
	if len(items) == 0 {
		return items
	}
	filtered := items[:0]
	for _, item := range items {
		layerIndex, ok := EntityLayerIndex(item.Props)
		if ok {
			if layerIndex == deletedIndex {
				continue
			}
			if layerIndex > deletedIndex {
				if item.Props == nil {
					item.Props = make(map[string]interface{})
				}
				item.Props["layer"] = layerIndex - 1
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func remapEntityLayerProps(items []levels.Entity, mapping map[int]int) {
	for index := range items {
		layerIndex, ok := EntityLayerIndex(items[index].Props)
		if !ok {
			continue
		}
		mapped, exists := mapping[layerIndex]
		if !exists {
			continue
		}
		if items[index].Props == nil {
			items[index].Props = make(map[string]interface{})
		}
		items[index].Props["layer"] = mapped
	}
}

func EntityLayerIndex(props map[string]interface{}) (int, bool) {
	if props == nil {
		return 0, false
	}
	raw, ok := props["layer"]
	if !ok {
		return 0, false
	}
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float32:
		return int(value), true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

func canonicalEntityID(item levels.Entity) string {
	id := strings.TrimSpace(item.ID)
	if id != "" {
		return id
	}
	if item.Props == nil {
		return ""
	}
	if value, ok := item.Props["id"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func syncEntityID(item *levels.Entity, id string) bool {
	if item == nil {
		return false
	}
	changed := strings.TrimSpace(item.ID) != id
	item.ID = id
	if strings.EqualFold(strings.TrimSpace(item.Type), "transition") || hasStringProp(item.Props, "id") {
		if item.Props == nil {
			item.Props = make(map[string]interface{})
		}
		if value, _ := item.Props["id"].(string); value != id {
			item.Props["id"] = id
			changed = true
		}
	}
	return changed
}

func hasStringProp(props map[string]interface{}, key string) bool {
	if props == nil {
		return false
	}
	_, ok := props[key].(string)
	return ok
}

func nextGeneratedEntityID(item levels.Entity, used map[string]struct{}, nextByPrefix map[string]int, nextTransition *int) string {
	if strings.EqualFold(strings.TrimSpace(item.Type), "transition") {
		for {
			candidate := fmt.Sprintf("t%d", *nextTransition)
			*nextTransition++
			if _, exists := used[candidate]; !exists {
				return candidate
			}
		}
	}
	prefix := sanitizeEntityIDPrefix(item.Type)
	for {
		nextByPrefix[prefix]++
		candidate := fmt.Sprintf("%s_%d", prefix, nextByPrefix[prefix])
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

func sanitizeEntityIDPrefix(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "entity"
	}
	var builder strings.Builder
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			if builder.Len() == 0 || strings.HasSuffix(builder.String(), "_") {
				continue
			}
			builder.WriteByte('_')
		}
	}
	prefix := strings.Trim(builder.String(), "_")
	if prefix == "" {
		return "entity"
	}
	return prefix
}
