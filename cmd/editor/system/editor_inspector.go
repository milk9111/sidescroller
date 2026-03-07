package editorsystem

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	editoruicomponents "github.com/milk9111/sidescroller/cmd/editor/ui/components"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/prefabs"
	"gopkg.in/yaml.v3"
)

var inspectorComponentTypes = map[string]reflect.Type{
	"player":              reflect.TypeOf(prefabs.PlayerComponentSpec{}),
	"transform":           reflect.TypeOf(prefabs.TransformComponentSpec{}),
	"parallax":            reflect.TypeOf(prefabs.ParallaxComponentSpec{}),
	"color":               reflect.TypeOf(prefabs.ColorComponentSpec{}),
	"spawn_children":      reflect.TypeOf(prefabs.SpawnChildrenComponentSpec{}),
	"sprite":              reflect.TypeOf(prefabs.SpriteComponentSpec{}),
	"render_layer":        reflect.TypeOf(prefabs.RenderLayerComponentSpec{}),
	"line_render":         reflect.TypeOf(prefabs.LineRenderComponentSpec{}),
	"camera":              reflect.TypeOf(prefabs.CameraComponentSpec{}),
	"ai":                  reflect.TypeOf(prefabs.AIComponentSpec{}),
	"pathfinding":         reflect.TypeOf(prefabs.PathfindingComponentSpec{}),
	"ai_config":           reflect.TypeOf(prefabs.AIConfigComponentSpec{}),
	"script":              reflect.TypeOf(prefabs.ScriptComponentSpec{}),
	"animation":           reflect.TypeOf(prefabs.AnimationComponentSpec{}),
	"audio":               reflect.TypeOf(prefabs.AudioComponentSpec{}),
	"music_player":        reflect.TypeOf(prefabs.MusicPlayerComponentSpec{}),
	"physics_body":        reflect.TypeOf(prefabs.PhysicsBodyComponentSpec{}),
	"ttl":                 reflect.TypeOf(prefabs.TTLComponentSpec{}),
	"collision_layer":     reflect.TypeOf(prefabs.CollisionLayerComponentSpec{}),
	"repulsion_layer":     reflect.TypeOf(prefabs.RepulsionLayerComponentSpec{}),
	"gravity_scale":       reflect.TypeOf(prefabs.GravityScaleComponentSpec{}),
	"hazard":              reflect.TypeOf(prefabs.HazardComponentSpec{}),
	"health":              reflect.TypeOf(prefabs.HealthComponentSpec{}),
	"hitboxes":            reflect.TypeOf([]prefabs.HitboxComponentSpec{}),
	"hurtboxes":           reflect.TypeOf([]prefabs.HurtboxComponentSpec{}),
	"anchor":              reflect.TypeOf(prefabs.AnchorComponentSpec{}),
	"pickup":              reflect.TypeOf(prefabs.PickupComponentSpec{}),
	"ai_phase_controller": reflect.TypeOf(prefabs.AIPhaseControllerComponentSpec{}),
	"arena_node":          reflect.TypeOf(prefabs.ArenaNodeComponentSpec{}),
}

var inspectorPreferredOrder = []string{
	"transform",
	"sprite",
	"animation",
	"color",
	"render_layer",
	"physics_body",
	"hazard",
	"hitboxes",
	"hurtboxes",
	"ai",
	"ai_config",
	"arena_node",
}

func buildInspectorState(catalog *editorcomponent.PrefabCatalog, entities *editorcomponent.LevelEntities, selectedIndex int) editoruicomponents.InspectorState {
	if entities == nil || selectedIndex < 0 || selectedIndex >= len(entities.Items) {
		return editoruicomponents.InspectorState{}
	}
	item := entities.Items[selectedIndex]
	prefab := prefabInfoForEntity(catalog, item)
	state := editoruicomponents.InspectorState{Active: true, EntityLabel: entityLabel(item)}
	if prefab == nil {
		return state
	}
	state.PrefabPath = prefab.Path
	components := editorio.MergeComponentMaps(prefab.Components, entityComponentOverrides(item.Props))
	if len(components) == 0 {
		return state
	}
	componentNames := make([]string, 0, len(components))
	for name := range components {
		componentNames = append(componentNames, name)
	}
	sort.Slice(componentNames, func(i, j int) bool {
		return inspectorComponentOrder(componentNames[i]) < inspectorComponentOrder(componentNames[j])
	})
	sections := make([]editoruicomponents.InspectorSectionState, 0, len(componentNames))
	for _, componentName := range componentNames {
		raw := components[componentName]
		section := buildInspectorSection(componentName, raw)
		if len(section.Fields) == 0 {
			continue
		}
		sections = append(sections, section)
	}
	state.Sections = sections
	return state
}

func buildInspectorSection(componentName string, raw any) editoruicomponents.InspectorSectionState {
	if specType, ok := inspectorComponentTypes[componentName]; ok {
		return buildRegisteredInspectorSection(componentName, specType, raw)
	}
	return buildFallbackInspectorSection(componentName, raw)
}

func buildRegisteredInspectorSection(componentName string, specType reflect.Type, raw any) editoruicomponents.InspectorSectionState {
	section := editoruicomponents.InspectorSectionState{Component: componentName, Label: humanizeKey(componentName)}
	if specType.Kind() == reflect.Slice {
		section.Fields = append(section.Fields, editoruicomponents.InspectorFieldState{
			Component: componentName,
			Field:     componentName,
			Label:     humanizeKey(componentName),
			TypeLabel: typeLabelFor(specType),
			Value:     valueToEditorString(valueOfType(raw, specType)),
		})
		return section
	}
	value := valueOfType(raw, specType)
	for index := 0; index < specType.NumField(); index++ {
		field := specType.Field(index)
		if field.PkgPath != "" {
			continue
		}
		fieldName := yamlFieldName(field)
		if fieldName == "" || fieldName == "-" {
			continue
		}
		section.Fields = append(section.Fields, editoruicomponents.InspectorFieldState{
			Component: componentName,
			Field:     fieldName,
			Label:     humanizeFieldName(field.Name, fieldName),
			TypeLabel: typeLabelFor(field.Type),
			Value:     valueToEditorString(value.Field(index)),
		})
	}
	return section
}

func buildFallbackInspectorSection(componentName string, raw any) editoruicomponents.InspectorSectionState {
	section := editoruicomponents.InspectorSectionState{Component: componentName, Label: humanizeKey(componentName)}
	values, ok := raw.(map[string]interface{})
	if !ok {
		section.Fields = append(section.Fields, editoruicomponents.InspectorFieldState{
			Component: componentName,
			Field:     componentName,
			Label:     humanizeKey(componentName),
			TypeLabel: typeLabelFor(reflect.TypeOf(raw)),
			Value:     valueToEditorString(reflect.ValueOf(raw)),
		})
		return section
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fieldType := reflect.TypeOf(values[key])
		section.Fields = append(section.Fields, editoruicomponents.InspectorFieldState{
			Component: componentName,
			Field:     key,
			Label:     humanizeKey(key),
			TypeLabel: typeLabelFor(fieldType),
			Value:     valueToEditorString(reflect.ValueOf(values[key])),
		})
	}
	return section
}

func applyInspectorFieldEdit(item *levels.Entity, prefab *editorio.PrefabInfo, componentName, fieldName, rawValue string) bool {
	if item == nil || prefab == nil || strings.TrimSpace(componentName) == "" || strings.TrimSpace(fieldName) == "" {
		return false
	}
	parsed, ok := parseInspectorFieldValue(prefab, componentName, fieldName, rawValue)
	if !ok {
		return false
	}
	componentValues := ensureEntityComponentOverrideValues(item, componentName)
	componentValues[fieldName] = parsed
	syncInspectorFieldToEntity(item, componentName, fieldName, parsed)
	return true
}

func parseInspectorFieldValue(prefab *editorio.PrefabInfo, componentName, fieldName, rawValue string) (any, bool) {
	targetType, ok := inspectorFieldType(prefab, componentName, fieldName)
	if !ok {
		return nil, false
	}
	value, err := parseEditorValue(rawValue, targetType)
	if err != nil {
		return nil, false
	}
	return value, true
}

func inspectorFieldType(prefab *editorio.PrefabInfo, componentName, fieldName string) (reflect.Type, bool) {
	if prefab == nil {
		return nil, false
	}
	if specType, ok := inspectorComponentTypes[componentName]; ok {
		if specType.Kind() == reflect.Slice {
			return specType, true
		}
		for index := 0; index < specType.NumField(); index++ {
			field := specType.Field(index)
			if yamlFieldName(field) == fieldName {
				return field.Type, true
			}
		}
	}
	if raw, ok := prefab.Components[componentName]; ok {
		if values, ok := raw.(map[string]interface{}); ok {
			if value, ok := values[fieldName]; ok && value != nil {
				return reflect.TypeOf(value), true
			}
		}
	}
	return reflect.TypeOf(""), true
}

func parseEditorValue(raw string, targetType reflect.Type) (any, error) {
	if targetType == nil {
		return raw, nil
	}
	if targetType.Kind() == reflect.Pointer {
		value, err := parseEditorValue(raw, targetType.Elem())
		if err != nil {
			return nil, err
		}
		pointer := reflect.New(targetType.Elem())
		if value == nil {
			return pointer.Interface(), nil
		}
		parsed := reflect.ValueOf(value)
		if parsed.Type().AssignableTo(targetType.Elem()) {
			pointer.Elem().Set(parsed)
		} else if parsed.Type().ConvertibleTo(targetType.Elem()) {
			pointer.Elem().Set(parsed.Convert(targetType.Elem()))
		} else {
			return nil, fmt.Errorf("cannot convert %T to %s", value, targetType.Elem())
		}
		return pointer.Elem().Interface(), nil
	}
	trimmed := strings.TrimSpace(raw)
	switch targetType.Kind() {
	case reflect.String:
		return raw, nil
	case reflect.Bool:
		if trimmed == "" {
			return false, nil
		}
		value, err := strconv.ParseBool(trimmed)
		if err != nil {
			return nil, err
		}
		return value, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if trimmed == "" {
			return reflect.Zero(targetType).Interface(), nil
		}
		value, err := strconv.ParseInt(trimmed, 10, targetType.Bits())
		if err != nil {
			return nil, err
		}
		parsed := reflect.New(targetType).Elem()
		parsed.SetInt(value)
		return parsed.Interface(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if trimmed == "" {
			return reflect.Zero(targetType).Interface(), nil
		}
		value, err := strconv.ParseUint(trimmed, 10, targetType.Bits())
		if err != nil {
			return nil, err
		}
		parsed := reflect.New(targetType).Elem()
		parsed.SetUint(value)
		return parsed.Interface(), nil
	case reflect.Float32, reflect.Float64:
		if trimmed == "" {
			return reflect.Zero(targetType).Interface(), nil
		}
		value, err := strconv.ParseFloat(trimmed, targetType.Bits())
		if err != nil {
			return nil, err
		}
		parsed := reflect.New(targetType).Elem()
		parsed.SetFloat(value)
		return parsed.Interface(), nil
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		parsed := reflect.New(targetType)
		if trimmed == "" {
			return parsed.Elem().Interface(), nil
		}
		if err := yaml.Unmarshal([]byte(raw), parsed.Interface()); err != nil {
			return nil, err
		}
		return parsed.Elem().Interface(), nil
	default:
		return raw, nil
	}
}

func syncInspectorFieldToEntity(item *levels.Entity, componentName, fieldName string, value any) {
	if item == nil {
		return
	}
	props := ensureEntityProps(item)
	switch componentName {
	case "transform":
		switch fieldName {
		case "x":
			item.X = int(reflect.ValueOf(value).Convert(reflect.TypeOf(float64(0))).Float())
		case "y":
			item.Y = int(reflect.ValueOf(value).Convert(reflect.TypeOf(float64(0))).Float())
		case "rotation":
			props["rotation"] = reflect.ValueOf(value).Convert(reflect.TypeOf(float64(0))).Float()
		case "scale_x", "scale_y":
			props[fieldName] = reflect.ValueOf(value).Convert(reflect.TypeOf(float64(0))).Float()
		}
	}
}

func inspectorComponentOrder(name string) int {
	for index, candidate := range inspectorPreferredOrder {
		if candidate == name {
			return index
		}
	}
	return len(inspectorPreferredOrder) + int(name[0])
}

func valueOfType(raw any, targetType reflect.Type) reflect.Value {
	if targetType == nil {
		return reflect.Value{}
	}
	decoded := reflect.New(targetType)
	if raw != nil {
		if bytes, err := yaml.Marshal(raw); err == nil {
			_ = yaml.Unmarshal(bytes, decoded.Interface())
		}
	}
	return decoded.Elem()
}

func yamlFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("yaml")
	if tag == "" {
		return strings.ToLower(field.Name)
	}
	parts := strings.Split(tag, ",")
	return parts[0]
}

func humanizeFieldName(fieldName, fallback string) string {
	if strings.TrimSpace(fieldName) != "" {
		return splitCamelCase(fieldName)
	}
	return humanizeKey(fallback)
}

func humanizeKey(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-'
	})
	for index := range parts {
		parts[index] = strings.Title(parts[index])
	}
	return strings.Join(parts, " ")
}

func splitCamelCase(value string) string {
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for index, r := range value {
		if index > 0 && unicode.IsUpper(r) {
			builder.WriteByte(' ')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func typeLabelFor(value reflect.Type) string {
	if value == nil {
		return "any"
	}
	if value.Kind() == reflect.Pointer {
		return typeLabelFor(value.Elem())
	}
	switch value.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "uint"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Struct:
		return value.Name()
	case reflect.Slice:
		return "yaml[]"
	case reflect.Map:
		return "yaml{}"
	default:
		return value.Kind().String()
	}
}

func valueToEditorString(value reflect.Value) string {
	if !value.IsValid() {
		return ""
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return valueToEditorString(reflect.Zero(value.Type().Elem()))
		}
		return valueToEditorString(value.Elem())
	}
	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Bool:
		return strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		bytes, err := yaml.Marshal(value.Interface())
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(bytes))
	default:
		return fmt.Sprintf("%v", value.Interface())
	}
}
