package entity

import (
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
       for layerIdx, layer := range lvl.Layers {
	       for y := 0; y < lvl.Height; y++ {
		       for x := 0; x < lvl.Width; x++ {
			       tileIdx := y*lvl.Width + x
			       tileID := layer[tileIdx]
			       if tileID < 0 {
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
				       Image:   img,
				       OriginX: 0,
				       OriginY: 0,
			       })
			       if err != nil {
				       return err
			       }
			       // Optionally add a Layer or Z component if needed for sorting
		       }
	       }
       }
       return nil
}
