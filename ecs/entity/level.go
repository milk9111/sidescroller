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
	boundsEntity := ecs.CreateEntity(world)
	if err := ecs.Add(world, boundsEntity, component.LevelBoundsComponent.Kind(), &component.LevelBounds{
		Width:  float64(lvl.Width) * tileSize,
		Height: float64(lvl.Height) * tileSize,
	}); err != nil {
		return err
	}

	for layerIdx, layer := range lvl.Layers {
		layerHasPhysics := false
		if layerIdx < len(lvl.LayerMeta) {
			layerHasPhysics = lvl.LayerMeta[layerIdx].Physics
		}
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

				e := ecs.CreateEntity(world)
				err := ecs.Add(world, e, component.TransformComponent.Kind(), &component.Transform{
					X:      float64(x) * tileSize,
					Y:      float64(y) * tileSize,
					ScaleX: 1,
					ScaleY: 1,
				})
				if err != nil {
					return err
				}

				err = ecs.Add(world, e, component.SpriteComponent.Kind(), &component.Sprite{
					Image:     img,
					Source:    image.Rect(srcX, srcY, srcX+tileW, srcY+tileH),
					UseSource: true,
					OriginX:   0,
					OriginY:   0,
				})
				if err != nil {
					return err
				}

				if err := ecs.Add(world, e, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: layerIdx}); err != nil {
					return err
				}
				if err := ecs.Add(world, e, component.StaticTileComponent.Kind(), &component.StaticTile{}); err != nil {
					return err
				}
				// Optionally add a Layer or Z component if needed for sorting
			}
		}
		if layerHasPhysics {
			if err := addMergedTileColliders(world, layer, lvl.Width, lvl.Height, tileSize); err != nil {
				return err
			}
		}
	}

	for _, ent := range lvl.Entities {
		switch strings.ToLower(ent.Type) {
		case "player":
			if _, err := NewPlayerAt(world, float64(ent.X), float64(ent.Y)); err != nil {
				return err
			}
		case "enemy":
			if _, err := NewEnemyAt(world, float64(ent.X), float64(ent.Y)); err != nil {
				return err
			}
		case "camera":
			if _, err := NewCameraAt(world, float64(ent.X), float64(ent.Y)); err != nil {
				return err
			}
		case "transition":
			te, err := NewTransition(world)
			if err != nil {
				return err
			}
			// Override transform position from level JSON.
			tr, _ := ecs.Get(world, te, component.TransformComponent.Kind())
			tr.X = float64(ent.X)
			tr.Y = float64(ent.Y)
			if tr.ScaleX == 0 {
				tr.ScaleX = 1
			}
			if tr.ScaleY == 0 {
				tr.ScaleY = 1
			}
			if err := ecs.Add(world, te, component.TransformComponent.Kind(), tr); err != nil {
				return err
			}

			// Props -> Transition component.
			props := ent.Props
			getString := func(key string) string {
				if props == nil {
					return ""
				}
				if v, ok := props[key]; ok {
					if s, ok := v.(string); ok {
						return s
					}
				}
				return ""
			}
			getFloat := func(key string) float64 {
				if props == nil {
					return 0
				}
				v, ok := props[key]
				if !ok {
					return 0
				}
				switch n := v.(type) {
				case float64:
					return n
				case float32:
					return float64(n)
				case int:
					return float64(n)
				case int32:
					return float64(n)
				case int64:
					return float64(n)
				case uint:
					return float64(n)
				case uint32:
					return float64(n)
				case uint64:
					return float64(n)
				default:
					return 0
				}
			}

			w := getFloat("w")
			h := getFloat("h")
			if w <= 0 {
				w = 32
			}
			if h <= 0 {
				h = 32
			}
			if w < 32 {
				w = 32
			}
			if h < 32 {
				h = 32
			}

			transComp := &component.Transition{
				ID:          getString("id"),
				TargetLevel: getString("to_level"),
				LinkedID:    getString("linked_id"),
				EnterDir:    component.TransitionDirection(strings.ToLower(getString("enter_dir"))),
				Bounds: component.AABB{
					X: 0,
					Y: 0,
					W: w,
					H: h,
				},
			}
			if err := ecs.Add(world, te, component.TransitionComponent.Kind(), transComp); err != nil {
				return err
			}
		default:
			// Unknown entity type; ignore for now.
		}
	}

	return nil
}

func addMergedTileColliders(world *ecs.World, layer []int, width, height int, tileSize float64) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	visited := make([]bool, width*height)
	index := func(x, y int) int { return y*width + x }

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := index(x, y)
			if idx < 0 || idx >= len(layer) {
				continue
			}
			if visited[idx] || layer[idx] <= 0 {
				continue
			}

			maxW := 0
			for x2 := x; x2 < width; x2++ {
				idx2 := index(x2, y)
				if idx2 >= len(layer) || visited[idx2] || layer[idx2] <= 0 {
					break
				}
				maxW++
			}
			if maxW == 0 {
				continue
			}

			maxH := 1
			for y2 := y + 1; y2 < height; y2++ {
				rowOK := true
				for x2 := x; x2 < x+maxW; x2++ {
					idx2 := index(x2, y2)
					if idx2 >= len(layer) || visited[idx2] || layer[idx2] <= 0 {
						rowOK = false
						break
					}
				}
				if !rowOK {
					break
				}
				maxH++
			}

			for yy := y; yy < y+maxH; yy++ {
				for xx := x; xx < x+maxW; xx++ {
					idx2 := index(xx, yy)
					if idx2 >= 0 && idx2 < len(visited) {
						visited[idx2] = true
					}
				}
			}

			e := ecs.CreateEntity(world)
			if err := ecs.Add(world, e, component.TransformComponent.Kind(), &component.Transform{
				X:      float64(x) * tileSize,
				Y:      float64(y) * tileSize,
				ScaleX: 1,
				ScaleY: 1,
			}); err != nil {
				return err
			}
			if err := ecs.Add(world, e, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{
				Width:        float64(maxW) * tileSize,
				Height:       float64(maxH) * tileSize,
				Friction:     0.9,
				Static:       true,
				AlignTopLeft: true,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
