package entity

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/levels"
)

// LoadLevelToWorld loads a level into the ECS world, creating tile entities for each tile in each layer.
func LoadLevelToWorld(world *ecs.World, lvl *levels.Level) error {
	imgs := make(map[string]*ebiten.Image)

	tileSize := 32.0 // hardcoded for now
	if boundsEntity := world.CreateEntity(); boundsEntity.Valid() {
		if err := ecs.Add(world, boundsEntity, component.LevelBoundsComponent, component.LevelBounds{
			Width:  float64(lvl.Width) * tileSize,
			Height: float64(lvl.Height) * tileSize,
		}); err != nil {
			return err
		}
	}
	for layerIdx, layer := range lvl.Layers {
		for y := 0; y < lvl.Height; y++ {
			for x := 0; x < lvl.Width; x++ {
				tileIdx := y*lvl.Width + x
				tileID := layer[tileIdx]
				if tileID <= 0 {
					continue // skip empty tiles
				}

				tileInfo := lvl.TilesetUsage[layerIdx][tileIdx]
				if tileInfo == nil {
					continue
				}

				img, ok := imgs[tileInfo.Path]
				if !ok {
					var err error
					img, err = assets.LoadImage(tileInfo.Path)
					if err != nil {
						return err
					}
					imgs[tileInfo.Path] = img
				}

				imgW, imgH := img.Size()
				tileW := tileInfo.TileW
				tileH := tileInfo.TileH
				if tileW <= 0 {
					tileW = 32
				}
				if tileH <= 0 {
					tileH = 32
				}
				tilesX := imgW / tileW
				if tilesX <= 0 {
					continue
				}
				idx := tileInfo.Index
				srcX := (idx % tilesX) * tileW
				srcY := (idx / tilesX) * tileH
				if srcX < 0 || srcY < 0 || srcX+tileW > imgW || srcY+tileH > imgH {
					continue
				}

				e := world.CreateEntity()
				err := ecs.Add(world, e, component.TransformComponent, component.Transform{
					X:      float64(x) * tileSize,
					Y:      float64(y) * tileSize,
					ScaleX: 1,
					ScaleY: 1,
				})
				if err != nil {
					return err
				}

				err = ecs.Add(world, e, component.SpriteComponent, component.Sprite{
					Image:     img,
					Source:    image.Rect(srcX, srcY, srcX+tileW, srcY+tileH),
					UseSource: true,
					OriginX:   0,
					OriginY:   0,
				})
				if err != nil {
					return err
				}

				if err := ecs.Add(world, e, component.RenderLayerComponent, component.RenderLayer{Index: layerIdx}); err != nil {
					return err
				}
				// Optionally add a Layer or Z component if needed for sorting
			}
		}
	}

	for _, ent := range lvl.Entities {
		switch strings.ToLower(ent.Type) {
		case "player":
			if _, err := NewPlayerAt(world, float64(ent.X), float64(ent.Y)); err != nil {
				return err
			}
		case "camera":
			if _, err := NewCameraAt(world, float64(ent.X), float64(ent.Y)); err != nil {
				return err
			}
		default:
			// Unknown entity type; ignore for now.
		}
	}

	return nil
}
