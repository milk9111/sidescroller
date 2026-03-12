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
	"circle_render":       reflect.TypeOf(prefabs.CircleRenderComponentSpec{}),
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
	"persistent":          reflect.TypeOf(prefabs.PersistentComponentSpec{}),
}

var inspectorPreferredOrder = []string{
	"transform",
	"sprite",
	"animation",
	"color",
	"render_layer",
	"circle_render",
	"physics_body",
	"hazard",
	"hitboxes",
	"hurtboxes",
	"ai",
	"ai_config",
	"arena_node",
	"persistent",
}

func buildInspectorState(catalog *editorcomponent.PrefabCatalog, entities *editorcomponent.LevelEntities, selectedIndex int) editoruicomponents.InspectorState {
	state := buildDefaultInspectorState()
	if entities == nil || selectedIndex < 0 || selectedIndex >= len(entities.Items) {
		return state
	}
	item := entities.Items[selectedIndex]
	prefab := prefabInfoForEntity(catalog, item)
	state.Active = true
	state.EntityLabel = entityLabel(item)
	components := entityComponentOverrides(item.Props)
	if prefab != nil {
		state.PrefabPath = prefab.Path
		components = editorio.MergeComponentMaps(prefab.Components, components)
	}
	if len(components) == 0 {
		if prefab != nil {
			state.StatusMessage = "No editable prefab components found"
		} else {
			state.StatusMessage = "No editable components found"
		}
		return state
	}
	document, err := buildInspectorDocument(components)
	if err != nil {
		state.ParseError = err.Error()
		state.StatusMessage = "Unable to build component YAML"
		return state
	}
	state.DocumentText = document
	return state
}

func buildDefaultInspectorState() editoruicomponents.InspectorState {
	return editoruicomponents.InspectorState{StatusMessage: "Select an entity to inspect"}
}

func buildInspectorDocument(components map[string]any) (string, error) {
	if len(components) == 0 {
		return "", nil
	}
	componentNames := make([]string, 0, len(components))
	for name := range components {
		componentNames = append(componentNames, name)
	}
	sort.Slice(componentNames, func(i, j int) bool {
		return lessInspectorComponentName(componentNames[i], componentNames[j])
	})
	root := yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, componentName := range componentNames {
		valueNode, err := yamlNodeForInspectorValue(components[componentName])
		if err != nil {
			return "", fmt.Errorf("marshal component %q: %w", componentName, err)
		}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: componentName},
			valueNode,
		)
	}
	var builder strings.Builder
	encoder := yaml.NewEncoder(&builder)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return strings.TrimRight(builder.String(), "\n"), nil
}

func parseInspectorDocument(document string) (map[string]any, error) {
	trimmed := strings.TrimSpace(document)
	if trimmed == "" {
		return nil, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(document), &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return nil, nil
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("inspector YAML must be a mapping of components")
	}
	components := make(map[string]any, len(mapping.Content)/2)
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		name := strings.TrimSpace(mapping.Content[index].Value)
		if name == "" {
			return nil, fmt.Errorf("component names must be non-empty")
		}
		var value any
		if err := mapping.Content[index+1].Decode(&value); err != nil {
			return nil, fmt.Errorf("decode component %q: %w", name, err)
		}
		components[name] = normalizeInspectorValue(value)
	}
	if len(components) == 0 {
		return nil, nil
	}
	return components, nil
}

func yamlNodeForInspectorValue(value any) (*yaml.Node, error) {
	bytes, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(bytes, &node); err != nil {
		return nil, err
	}
	if len(node.Content) == 0 {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	}
	return node.Content[0], nil
}

func applyInspectorDocumentEdit(item *levels.Entity, prefab *editorio.PrefabInfo, document string) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("no selected entity")
	}
	editedComponents, err := parseInspectorDocument(document)
	if err != nil {
		return false, err
	}
	updated := cloneEditorEntity(*item)
	var overrides map[string]any
	effectiveComponents := cloneInspectorComponentMap(editedComponents)
	if prefab != nil {
		overrides = diffInspectorComponentMaps(prefab.Components, editedComponents)
		effectiveComponents = editorio.MergeComponentMaps(prefab.Components, overrides)
	} else {
		overrides = cloneInspectorComponentMap(editedComponents)
	}
	writeEntityComponentOverrides(&updated, overrides)
	syncInspectorEffectiveComponents(&updated, effectiveComponents)
	if reflect.DeepEqual(*item, updated) {
		return false, nil
	}
	*item = updated
	return true, nil
}

func diffInspectorComponentMaps(base, edited map[string]any) map[string]any {
	if len(edited) == 0 {
		return nil
	}
	overrides := make(map[string]any)
	for name, editedValue := range edited {
		baseValue, hasBase := base[name]
		if !hasBase {
			overrides[name] = cloneInspectorValue(editedValue)
			continue
		}
		diff, changed := diffInspectorValue(baseValue, editedValue)
		if changed {
			overrides[name] = diff
		}
	}
	if len(overrides) == 0 {
		return nil
	}
	return overrides
}

func diffInspectorValue(base, edited any) (any, bool) {
	base = normalizeInspectorValue(base)
	edited = normalizeInspectorValue(edited)
	baseMap, baseIsMap := inspectorMapValue(base)
	editedMap, editedIsMap := inspectorMapValue(edited)
	if baseIsMap && editedIsMap {
		result := make(map[string]any)
		for key, editedValue := range editedMap {
			baseValue, hasBase := baseMap[key]
			if !hasBase {
				result[key] = cloneInspectorValue(editedValue)
				continue
			}
			diff, changed := diffInspectorValue(baseValue, editedValue)
			if changed {
				result[key] = diff
			}
		}
		if len(result) == 0 {
			return nil, false
		}
		return result, true
	}
	if inspectorValuesEqual(base, edited) {
		return nil, false
	}
	return cloneInspectorValue(edited), true
}

func normalizeInspectorValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, nested := range typed {
			normalized[key] = normalizeInspectorValue(nested)
		}
		return normalized
	case map[interface{}]interface{}:
		normalized := make(map[string]any, len(typed))
		for key, nested := range typed {
			normalized[fmt.Sprint(key)] = normalizeInspectorValue(nested)
		}
		return normalized
	case []any:
		normalized := make([]any, len(typed))
		for index := range typed {
			normalized[index] = normalizeInspectorValue(typed[index])
		}
		return normalized
	default:
		return typed
	}
}

func cloneInspectorComponentMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = cloneInspectorValue(value)
	}
	return cloned
}

func cloneInspectorValue(value any) any {
	switch typed := normalizeInspectorValue(value).(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, nested := range typed {
			cloned[key] = cloneInspectorValue(nested)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for index := range typed {
			cloned[index] = cloneInspectorValue(typed[index])
		}
		return cloned
	default:
		return typed
	}
}

func inspectorValuesEqual(left, right any) bool {
	left = normalizeInspectorValue(left)
	right = normalizeInspectorValue(right)
	leftMap, leftIsMap := inspectorMapValue(left)
	rightMap, rightIsMap := inspectorMapValue(right)
	if leftIsMap || rightIsMap {
		if !leftIsMap || !rightIsMap || len(leftMap) != len(rightMap) {
			return false
		}
		for key, leftValue := range leftMap {
			rightValue, ok := rightMap[key]
			if !ok || !inspectorValuesEqual(leftValue, rightValue) {
				return false
			}
		}
		return true
	}
	leftList, leftIsList := inspectorSliceValue(left)
	rightList, rightIsList := inspectorSliceValue(right)
	if leftIsList || rightIsList {
		if !leftIsList || !rightIsList || len(leftList) != len(rightList) {
			return false
		}
		for index := range leftList {
			if !inspectorValuesEqual(leftList[index], rightList[index]) {
				return false
			}
		}
		return true
	}
	leftNumber, leftIsNumber := inspectorNumericValue(left)
	rightNumber, rightIsNumber := inspectorNumericValue(right)
	if leftIsNumber || rightIsNumber {
		return leftIsNumber && rightIsNumber && leftNumber == rightNumber
	}
	return reflect.DeepEqual(left, right)
}

func inspectorMapValue(value any) (map[string]any, bool) {
	typed, ok := normalizeInspectorValue(value).(map[string]any)
	return typed, ok
}

func inspectorSliceValue(value any) ([]any, bool) {
	typed, ok := normalizeInspectorValue(value).([]any)
	return typed, ok
}

func inspectorNumericValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return 0, false
	}
}

func writeEntityComponentOverrides(item *levels.Entity, overrides map[string]any) {
	if item == nil {
		return
	}
	props := ensureEntityProps(item)
	if len(overrides) == 0 {
		delete(props, entityComponentsKey)
		return
	}
	props[entityComponentsKey] = cloneInspectorComponentMap(overrides)
}

func syncInspectorEffectiveComponents(item *levels.Entity, components map[string]any) {
	if item == nil {
		return
	}
	props := ensureEntityProps(item)
	transform, _ := inspectorMapValue(nil)
	if components != nil {
		transform, _ = inspectorMapValue(components["transform"])
	}
	if x, ok := inspectorNumericValue(transform["x"]); ok {
		item.X = int(x)
	}
	if y, ok := inspectorNumericValue(transform["y"]); ok {
		item.Y = int(y)
	}
	syncInspectorTransformProp(props, transform, "rotation")
	syncInspectorTransformProp(props, transform, "scale_x")
	syncInspectorTransformProp(props, transform, "scale_y")
	syncLegacyInspectorTransform(props, transform)
}

func syncInspectorTransformProp(props map[string]interface{}, transform map[string]any, key string) {
	if props == nil {
		return
	}
	if value, ok := inspectorNumericValue(transform[key]); ok {
		props[key] = value
		return
	}
	delete(props, key)
}

func syncLegacyInspectorTransform(props map[string]interface{}, transform map[string]any) {
	if props == nil {
		return
	}
	if len(transform) == 0 {
		delete(props, "transform")
		return
	}
	legacy := make(map[string]any)
	for _, key := range []string{"x", "y", "scale_x", "scale_y", "rotation"} {
		if value, ok := inspectorNumericValue(transform[key]); ok {
			legacy[key] = value
		}
	}
	if len(legacy) == 0 {
		delete(props, "transform")
		return
	}
	props["transform"] = legacy
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

func lessInspectorComponentName(left, right string) bool {
	leftOrder := inspectorComponentOrder(left)
	rightOrder := inspectorComponentOrder(right)
	if leftOrder == rightOrder {
		return left < right
	}
	return leftOrder < rightOrder
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
