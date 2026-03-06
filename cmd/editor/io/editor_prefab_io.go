package editorio

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/milk9111/sidescroller/prefabs"
)

type PrefabPreview struct {
	ImagePath    string
	FrameX       int
	FrameY       int
	FrameW       int
	FrameH       int
	OriginX      float64
	OriginY      float64
	RenderLayer  int
	FallbackSize int
}

type PrefabInfo struct {
	Name       string
	Path       string
	EntityType string
	Preview    PrefabPreview
}

func ScanPrefabCatalog(workspaceRoot string) ([]PrefabInfo, error) {
	root := filepath.Join(workspaceRoot, "prefabs")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read prefab dir %q: %w", root, err)
	}

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
		if strings.TrimSpace(buildSpec.Name) != "" {
			name = strings.TrimSpace(buildSpec.Name)
			entityType = name
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
						resolved.RenderLayer = preview.RenderLayer
						preview = resolved
					}
				}
			}
			if preview.ImagePath == "" && spriteSpec != nil {
				if resolved, ok := previewFromSprite(spriteAdapterFromComponentSpec(spriteSpec)); ok {
					resolved.RenderLayer = preview.RenderLayer
					preview = resolved
				}
			}
		}
	}

	return PrefabInfo{
		Name:       name,
		Path:       path,
		EntityType: entityType,
		Preview:    preview,
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
		FallbackSize: 32,
	}, true
}

type previewSpriteAdapter struct {
	image   string
	originX float64
	originY float64
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
	return &previewSpriteAdapter{image: spec.Image, originX: spec.OriginX, originY: spec.OriginY}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
