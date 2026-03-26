package editorio

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/milk9111/sidescroller/prefabs"
	"gopkg.in/yaml.v3"
)

type PrefabPreview struct {
	ImagePath    string
	FrameX       int
	FrameY       int
	FrameW       int
	FrameH       int
	OriginX      float64
	OriginY      float64
	ScaleX       float64
	ScaleY       float64
	CenterOrigin bool
	RenderLayer  int
	TintR        float64
	TintG        float64
	TintB        float64
	TintA        float64
	HasTint      bool
	FallbackSize int
}

type PrefabInfo struct {
	Name       string
	Path       string
	EntityType string
	Preview    PrefabPreview
	Components map[string]any
}

func ResolvePrefabPreview(info PrefabInfo, componentOverrides map[string]any) PrefabPreview {
	preview := info.Preview
	if len(componentOverrides) == 0 {
		return preview
	}
	if !hasRelevantPreviewOverride(componentOverrides) {
		return preview
	}
	if transformRaw, ok := mergedComponentForPreview(info.Components, componentOverrides, "transform"); ok {
		if transformSpec, err := prefabs.DecodeComponentSpec[prefabs.TransformComponentSpec](transformRaw); err == nil {
			preview.ScaleX = transformSpec.ScaleX
			preview.ScaleY = transformSpec.ScaleY
		}
	}
	if colorRaw, ok := mergedComponentForPreview(info.Components, componentOverrides, "color"); ok {
		if colorSpec, err := prefabs.DecodeComponentSpec[prefabs.ColorComponentSpec](colorRaw); err == nil {
			if tinted, ok := previewTintFromColor(colorSpec); ok {
				preview.TintR = tinted.r
				preview.TintG = tinted.g
				preview.TintB = tinted.b
				preview.TintA = tinted.a
				preview.HasTint = true
			} else {
				preview.HasTint = false
				preview.TintR = 0
				preview.TintG = 0
				preview.TintB = 0
				preview.TintA = 0
			}
		}
	}
	if renderRaw, ok := mergedComponentForPreview(info.Components, componentOverrides, "render_layer"); ok {
		if renderSpec, err := prefabs.DecodeComponentSpec[prefabs.RenderLayerComponentSpec](renderRaw); err == nil {
			preview.RenderLayer = renderSpec.Index
		}
	}
	spriteRaw, hasSprite := mergedComponentForPreview(info.Components, componentOverrides, "sprite")
	if hasSprite {
		if spriteSpec, err := prefabs.DecodeComponentSpec[prefabs.SpriteComponentSpec](spriteRaw); err == nil {
			preview.OriginX = spriteSpec.OriginX
			preview.OriginY = spriteSpec.OriginY
			preview.CenterOrigin = spriteSpec.CenterOriginIfZero
			if resolved, ok := previewFromSprite(spriteAdapterFromComponentSpec(&spriteSpec)); ok {
				preview = mergeResolvedPreview(preview, resolved)
			}
			if spriteSpec.UseSource && spriteSpec.SourceW > 0 && spriteSpec.SourceH > 0 {
				preview.FrameX = spriteSpec.SourceX
				preview.FrameY = spriteSpec.SourceY
				preview.FrameW = spriteSpec.SourceW
				preview.FrameH = spriteSpec.SourceH
				if preview.FallbackSize <= 0 {
					preview.FallbackSize = max(spriteSpec.SourceW, spriteSpec.SourceH)
				}
			}
		}
	}
	if animationRaw, ok := mergedComponentForPreview(info.Components, componentOverrides, "animation"); ok && preview.ImagePath == "" {
		if animationSpec, err := prefabs.DecodeComponentSpec[prefabs.AnimationSpec](animationRaw); err == nil {
			var spriteSpec *prefabs.SpriteComponentSpec
			if hasSprite {
				if decoded, decodeErr := prefabs.DecodeComponentSpec[prefabs.SpriteComponentSpec](spriteRaw); decodeErr == nil {
					spriteSpec = &decoded
				}
			}
			if resolved, ok := previewFromAnimation(&animationSpec, spriteAdapterFromComponentSpec(spriteSpec)); ok {
				preview = mergeResolvedPreview(preview, resolved)
			}
		}
	}
	return preview
}

func hasRelevantPreviewOverride(componentOverrides map[string]any) bool {
	if len(componentOverrides) == 0 {
		return false
	}
	for _, key := range []string{"transform", "color", "render_layer", "sprite", "animation"} {
		if _, ok := componentOverrides[key]; ok {
			return true
		}
	}
	return false
}

func mergedComponentForPreview(base, overrides map[string]any, key string) (any, bool) {
	override, hasOverride := overrides[key]
	if !hasOverride {
		value, ok := base[key]
		return value, ok
	}
	baseValue, hasBase := base[key]
	if !hasBase {
		return cloneComponentValue(override), true
	}
	return mergeComponentValue(baseValue, override), true
}

func MergeComponentMaps(base, overrides map[string]any) map[string]any {
	if len(base) == 0 && len(overrides) == 0 {
		return nil
	}
	merged := cloneComponentMap(base)
	if merged == nil {
		merged = make(map[string]any)
	}
	for key, value := range overrides {
		if existing, ok := merged[key]; ok {
			merged[key] = mergeComponentValue(existing, value)
			continue
		}
		merged[key] = cloneComponentValue(value)
	}
	return merged
}

func cloneComponentMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = cloneComponentValue(value)
	}
	return cloned
}

func cloneComponentValue(value any) any {
	switch typed := value.(type) {
	case map[string]interface{}:
		converted := make(map[string]any, len(typed))
		for key, nested := range typed {
			converted[key] = cloneComponentValue(nested)
		}
		return converted
	case []interface{}:
		cloned := make([]any, len(typed))
		for index := range typed {
			cloned[index] = cloneComponentValue(typed[index])
		}
		return cloned
	default:
		return typed
	}
}

func mergeComponentValue(base, override any) any {
	baseMap, baseIsMap := normalizeComponentMap(base)
	overrideMap, overrideIsMap := normalizeComponentMap(override)
	if baseIsMap && overrideIsMap {
		return MergeComponentMaps(baseMap, overrideMap)
	}
	return cloneComponentValue(override)
}

func normalizeComponentMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]interface{}:
		converted := make(map[string]any, len(typed))
		for key, nested := range typed {
			converted[key] = nested
		}
		return converted, true
	default:
		return nil, false
	}
}

func ScanPrefabCatalog(workspaceRoot string, prefabDir string) ([]PrefabInfo, error) {
	root := filepath.Join(workspaceRoot, prefabDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read prefab dir %q: %w", root, err)
	}
	previousWD, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("read working directory: %w", err)
	}
	if err := os.Chdir(workspaceRoot); err != nil {
		return nil, fmt.Errorf("change directory to %q: %w", workspaceRoot, err)
	}
	defer func() {
		_ = os.Chdir(previousWD)
	}()

	items := make([]PrefabInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		info, err := loadPrefabInfo(entry.Name())
		if err != nil {
			return nil, err
		}
		items = append(items, info)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Path < items[j].Path
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func NormalizePrefabTarget(target string) (string, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "", fmt.Errorf("prefab name is required")
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return "", fmt.Errorf("prefab name must not contain path separators")
	}
	if ext := strings.ToLower(filepath.Ext(trimmed)); ext == "" {
		trimmed += ".yaml"
	} else if ext != ".yaml" && ext != ".yml" {
		trimmed += ".yaml"
	}
	return trimmed, nil
}

func SavePrefab(workspaceRoot, target string, spec prefabs.EntityBuildSpec) (string, error) {
	normalized, err := NormalizePrefabTarget(target)
	if err != nil {
		return "", err
	}
	if spec.Components == nil {
		spec.Components = map[string]any{}
	}
	data, err := yaml.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshal prefab %q: %w", normalized, err)
	}
	path := filepath.Join(workspaceRoot, "prefabs", normalized)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create prefab dir for %q: %w", normalized, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write prefab %q: %w", normalized, err)
	}
	return normalized, nil
}

func loadPrefabInfo(path string) (PrefabInfo, error) {
	buildSpec, buildErr := prefabs.LoadEntityBuildSpec(path)
	previewSpec, previewErr := prefabs.LoadPreviewSpec(path)
	if buildErr != nil && previewErr != nil {
		return PrefabInfo{}, fmt.Errorf("load prefab %q: %w", path, buildErr)
	}

	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	entityType := base
	name := base
	if buildErr == nil {
		if strings.TrimSpace(buildSpec.EntityType) != "" {
			entityType = strings.TrimSpace(buildSpec.EntityType)
		}
		if strings.TrimSpace(buildSpec.Name) != "" {
			name = strings.TrimSpace(buildSpec.Name)
			if strings.TrimSpace(buildSpec.EntityType) == "" {
				entityType = name
			}
		}
	}

	preview := PrefabPreview{FallbackSize: 32}
	if previewErr == nil {
		if resolved, ok := previewFromAnimation(previewSpec.Animation, spriteAdapterFromSpriteSpec(previewSpec.Sprite)); ok {
			preview = resolved
		} else if resolved, ok := previewFromSprite(spriteAdapterFromSpriteSpec(previewSpec.Sprite)); ok {
			preview = resolved
		}
	}
	if buildErr == nil {
		if transformRaw, ok := buildSpec.Components["transform"]; ok {
			if transformSpec, err := prefabs.DecodeComponentSpec[prefabs.TransformComponentSpec](transformRaw); err == nil {
				preview.ScaleX = transformSpec.ScaleX
				preview.ScaleY = transformSpec.ScaleY
			}
		}
		if colorRaw, ok := buildSpec.Components["color"]; ok {
			if colorSpec, err := prefabs.DecodeComponentSpec[prefabs.ColorComponentSpec](colorRaw); err == nil {
				if tinted, ok := previewTintFromColor(colorSpec); ok {
					preview.TintR = tinted.r
					preview.TintG = tinted.g
					preview.TintB = tinted.b
					preview.TintA = tinted.a
					preview.HasTint = true
				}
			}
		}
		if renderRaw, ok := buildSpec.Components["render_layer"]; ok {
			if renderSpec, err := prefabs.DecodeComponentSpec[prefabs.RenderLayerComponentSpec](renderRaw); err == nil {
				preview.RenderLayer = renderSpec.Index
			}
		}
		if preview.ImagePath == "" {
			var spriteSpec *prefabs.SpriteComponentSpec
			if spriteRaw, ok := buildSpec.Components["sprite"]; ok {
				if decoded, err := prefabs.DecodeComponentSpec[prefabs.SpriteComponentSpec](spriteRaw); err == nil {
					spriteSpec = &decoded
				}
			}
			if animRaw, ok := buildSpec.Components["animation"]; ok {
				if decoded, err := prefabs.DecodeComponentSpec[prefabs.AnimationSpec](animRaw); err == nil {
					if resolved, ok := previewFromAnimation(&decoded, spriteAdapterFromComponentSpec(spriteSpec)); ok {
						preview = mergeResolvedPreview(preview, resolved)
					}
				}
			}
			if preview.ImagePath == "" && spriteSpec != nil {
				if resolved, ok := previewFromSprite(spriteAdapterFromComponentSpec(spriteSpec)); ok {
					preview = mergeResolvedPreview(preview, resolved)
				}
			}
		}
	}
	preview = resolvePreviewImageFrame(preview)

	return PrefabInfo{
		Name:       name,
		Path:       path,
		EntityType: entityType,
		Preview:    preview,
		Components: buildSpec.Components,
	}, nil
}

func previewFromAnimation(animation *prefabs.AnimationSpec, sprite *previewSpriteAdapter) (PrefabPreview, bool) {
	if animation == nil || animation.Sheet == "" || len(animation.Defs) == 0 {
		return PrefabPreview{}, false
	}
	defName := animation.Current
	if defName == "" {
		keys := make([]string, 0, len(animation.Defs))
		for key := range animation.Defs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		defName = keys[0]
	}
	def, ok := animation.Defs[defName]
	if !ok {
		for _, next := range animation.Defs {
			def = next
			ok = true
			break
		}
	}
	if !ok || def.FrameW <= 0 || def.FrameH <= 0 {
		return PrefabPreview{}, false
	}
	preview := PrefabPreview{
		ImagePath:    animation.Sheet,
		FrameX:       def.ColStart * def.FrameW,
		FrameY:       def.Row * def.FrameH,
		FrameW:       def.FrameW,
		FrameH:       def.FrameH,
		FallbackSize: max(32, def.FrameW),
	}
	if sprite != nil {
		preview.OriginX = sprite.originX
		preview.OriginY = sprite.originY
		preview.CenterOrigin = sprite.centerOriginIfZero
	}
	return preview, true
}

func previewFromSprite(sprite *previewSpriteAdapter) (PrefabPreview, bool) {
	if sprite == nil || strings.TrimSpace(sprite.image) == "" {
		return PrefabPreview{}, false
	}
	return PrefabPreview{
		ImagePath:    strings.TrimSpace(sprite.image),
		OriginX:      sprite.originX,
		OriginY:      sprite.originY,
		CenterOrigin: sprite.centerOriginIfZero,
		FallbackSize: 32,
	}, true
}

func resolvePreviewImageFrame(preview PrefabPreview) PrefabPreview {
	if strings.TrimSpace(preview.ImagePath) == "" || (preview.FrameW > 0 && preview.FrameH > 0) {
		return preview
	}
	width, height, ok := previewImageDimensions(preview.ImagePath)
	if !ok {
		return preview
	}
	if preview.FrameW <= 0 {
		preview.FrameW = width
	}
	if preview.FrameH <= 0 {
		preview.FrameH = height
	}
	if preview.FallbackSize <= 0 {
		preview.FallbackSize = max(width, height)
	}
	return preview
}

func previewImageDimensions(imagePath string) (int, int, bool) {
	trimmed := strings.TrimSpace(imagePath)
	if trimmed == "" {
		return 0, 0, false
	}
	candidates := previewImageCandidates(trimmed)
	for _, candidate := range candidates {
		file, err := os.Open(candidate)
		if err != nil {
			continue
		}
		cfg, _, err := image.DecodeConfig(file)
		_ = file.Close()
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
			continue
		}
		return cfg.Width, cfg.Height, true
	}
	return 0, 0, false
}

func previewImageCandidates(imagePath string) []string {
	if filepath.IsAbs(imagePath) {
		return []string{imagePath}
	}
	candidates := []string{imagePath, filepath.Join("assets", imagePath)}
	cwd, err := os.Getwd()
	if err != nil {
		return candidates
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidates = append(candidates, filepath.Join(dir, imagePath), filepath.Join(dir, "assets", imagePath))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return candidates
}

type previewSpriteAdapter struct {
	image              string
	originX            float64
	originY            float64
	centerOriginIfZero bool
}

func spriteAdapterFromSpriteSpec(spec *prefabs.SpriteSpec) *previewSpriteAdapter {
	if spec == nil {
		return nil
	}
	return &previewSpriteAdapter{image: spec.Image, originX: spec.OriginX, originY: spec.OriginY}
}

func spriteAdapterFromComponentSpec(spec *prefabs.SpriteComponentSpec) *previewSpriteAdapter {
	if spec == nil {
		return nil
	}
	return &previewSpriteAdapter{image: spec.Image, originX: spec.OriginX, originY: spec.OriginY, centerOriginIfZero: spec.CenterOriginIfZero}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func mergeResolvedPreview(existing, resolved PrefabPreview) PrefabPreview {
	resolved.ScaleX = existing.ScaleX
	resolved.ScaleY = existing.ScaleY
	resolved.RenderLayer = existing.RenderLayer
	resolved.TintR = existing.TintR
	resolved.TintG = existing.TintG
	resolved.TintB = existing.TintB
	resolved.TintA = existing.TintA
	resolved.HasTint = existing.HasTint
	if resolved.FallbackSize <= 0 {
		resolved.FallbackSize = existing.FallbackSize
	}
	return resolved
}

type previewTint struct {
	r float64
	g float64
	b float64
	a float64
}

func previewTintFromColor(spec prefabs.ColorComponentSpec) (previewTint, bool) {
	tint := previewTint{r: 1, g: 1, b: 1, a: 1}
	hasTint := false
	if strings.TrimSpace(spec.Hex) != "" {
		parsed, err := parseHexColor(spec.Hex)
		if err == nil {
			nrgba := color.NRGBAModel.Convert(parsed).(color.NRGBA)
			tint.r = float64(nrgba.R) / 255.0
			tint.g = float64(nrgba.G) / 255.0
			tint.b = float64(nrgba.B) / 255.0
			tint.a = float64(nrgba.A) / 255.0
			hasTint = true
		}
	}
	if spec.R != nil {
		tint.r = *spec.R
		hasTint = true
	}
	if spec.G != nil {
		tint.g = *spec.G
		hasTint = true
	}
	if spec.B != nil {
		tint.b = *spec.B
		hasTint = true
	}
	if spec.A != nil {
		tint.a = *spec.A
		hasTint = true
	}
	return tint, hasTint
}

func parseHexColor(v string) (color.Color, error) {
	s := strings.TrimPrefix(strings.TrimSpace(v), "#")
	if len(s) != 6 && len(s) != 8 {
		return nil, fmt.Errorf("invalid color format: %q", v)
	}
	parse := func(start int) (uint8, error) {
		n, err := strconv.ParseUint(s[start:start+2], 16, 8)
		return uint8(n), err
	}
	r, err := parse(0)
	if err != nil {
		return nil, fmt.Errorf("parse red component: %w", err)
	}
	g, err := parse(2)
	if err != nil {
		return nil, fmt.Errorf("parse green component: %w", err)
	}
	b, err := parse(4)
	if err != nil {
		return nil, fmt.Errorf("parse blue component: %w", err)
	}
	a := uint8(255)
	if len(s) == 8 {
		a, err = parse(6)
		if err != nil {
			return nil, fmt.Errorf("parse alpha component: %w", err)
		}
	}
	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}
