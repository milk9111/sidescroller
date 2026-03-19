package entity

import (
	"fmt"
	"image"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/levels"
)

const (
	spikeHazardBaseWidth   = 32.0
	spikeHazardBaseHeight  = 26.0
	spikeHazardBaseOffsetY = 6.0
	spikeHazardOrigin      = 16.0
	spikeHazardEndInset    = 4.0
	spikeCellSize          = 32
)

type loadedSpikePlacement struct {
	x          int
	y          int
	rotation   int
	layerIndex int
}

type mergedSpikeHazard struct {
	x          float64
	y          float64
	w          float64
	h          float64
	layerIndex int
}

// LoadLevelToWorld loads a level into the ECS world, creating tile entities for each tile in each layer.
func LoadLevelToWorld(world *ecs.World, lvl *levels.Level) error {
	imgs := make(map[string]*ebiten.Image)
	spikes := make([]loadedSpikePlacement, 0, 16)

	tileSize := 32.0 // hardcoded for now
	levelGrid := buildLevelGridData(lvl, tileSize)
	loadedLayers := make([]bool, len(lvl.Layers))
	for layerIdx := range lvl.Layers {
		loadedLayers[layerIdx] = levelLayerActive(lvl, layerIdx)
	}
	boundsEntity := ecs.CreateEntity(world)
	if err := ecs.Add(world, boundsEntity, component.LevelBoundsComponent.Kind(), &component.LevelBounds{
		Width:  float64(lvl.Width) * tileSize,
		Height: float64(lvl.Height) * tileSize,
	}); err != nil {
		return err
	}
	if err := ecs.Add(world, boundsEntity, component.LevelGridComponent.Kind(), levelGrid); err != nil {
		return err
	}
	if err := ecs.Add(world, boundsEntity, component.LevelRuntimeComponent.Kind(), &component.LevelRuntime{
		Level:        lvl,
		TileSize:     tileSize,
		LoadedLayers: loadedLayers,
	}); err != nil {
		return err
	}
	// Attach a StaticTileBatchState to the same bounds entity so systems can
	// mark the static tile batch as dirty when tiles or layer visibility
	// changes occur.
	_ = ecs.Add(world, boundsEntity, component.StaticTileBatchStateComponent.Kind(), &component.StaticTileBatchState{Dirty: true})

	for layerIdx, layer := range lvl.Layers {
		if !levelLayerActive(lvl, layerIdx) {
			continue
		}
		layerHasPhysics := false
		if layerIdx < len(lvl.LayerMeta) {
			layerHasPhysics = lvl.LayerMeta[layerIdx].Physics
		}
		var layerUsage []*levels.TileInfo
		if layerIdx < len(lvl.TilesetUsage) {
			layerUsage = lvl.TilesetUsage[layerIdx]
		}
		for y := 0; y < lvl.Height; y++ {
			for x := 0; x < lvl.Width; x++ {
				tileIdx := y*lvl.Width + x
				tileID := layer[tileIdx]
				tileInfo := tileInfoAt(layerUsage, tileIdx)
				if !levelTileOccupied(tileID, tileInfo) {
					continue // skip empty tiles
				}
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
				if err := ecs.Add(world, e, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: layerIdx}); err != nil {
					return err
				}
				if err := ecs.Add(world, e, component.StaticTileComponent.Kind(), &component.StaticTile{}); err != nil {
					return err
				}
				// Optionally add a Layer or Z component if needed for sorting
			}
		}
		_ = layerHasPhysics
	}

	if err := RebuildMergedLevelPhysics(world, lvl, tileSize); err != nil {
		return err
	}

	usedGameEntityIDs := map[string]bool{}
	for i, ent := range lvl.Entities {
		gameEntityID := strings.TrimSpace(ent.ID)
		if gameEntityID == "" {
			gameEntityID = fmt.Sprintf("e%d", i+1)
		}
		for usedGameEntityIDs[gameEntityID] {
			gameEntityID = fmt.Sprintf("%s_%d", gameEntityID, i+1)
		}
		usedGameEntityIDs[gameEntityID] = true

		entityType := strings.ToLower(ent.Type)
		prefabPath := prefabPathForLevelEntity(entityType, ent.Props)
		props := ent.Props
		componentOverrides := componentOverridesFromLevelProps(props)
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
		getBool := func(key string, fallback bool) bool {
			if props == nil {
				return fallback
			}
			v, ok := props[key]
			if !ok {
				return fallback
			}
			b, ok := v.(bool)
			if !ok {
				return fallback
			}
			return b
		}
		layerIndex := levelEntityLayerIndex(props)
		if !levelLayerActive(lvl, layerIndex) {
			continue
		}

		switch entityType {
		case "transition":
			if prefabPath == "" {
				prefabPath = "transition.yaml"
			}
			te, err := BuildEntityWithOverrides(world, prefabPath, componentOverrides)
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

			bounds := levelEntityAreaBounds(props, true)

			transComp := &component.Transition{
				ID:          getString("id"),
				TargetLevel: getString("to_level"),
				LinkedID:    getString("linked_id"),
				EnterDir:    component.TransitionDirection(strings.ToLower(getString("enter_dir"))),
				Bounds:      bounds,
			}
			if err := ecs.Add(world, te, component.TransitionComponent.Kind(), transComp); err != nil {
				return err
			}
			if err := applyAreaBoundsComponent(world, te, bounds); err != nil {
				return err
			}
			if err := applyAreaTileStampRotation(world, te, levelEntityVisualRotation(props)); err != nil {
				return err
			}
			if err := ecs.Add(world, te, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: gameEntityID}); err != nil {
				return err
			}
			if err := ecs.Add(world, te, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: layerIndex}); err != nil {
				return err
			}
		case "gate":
			if prefabPath == "" {
				prefabPath = "gate.yaml"
			}
			ge, err := BuildEntityWithOverrides(world, prefabPath, componentOverrides)
			if err != nil {
				return err
			}

			bounds := levelEntityAreaBounds(props, true)

			tr, ok := ecs.Get(world, ge, component.TransformComponent.Kind())
			if !ok || tr == nil {
				tr = &component.Transform{}
			}
			tr.X = float64(ent.X)
			tr.Y = float64(ent.Y)
			if tr.ScaleX == 0 {
				tr.ScaleX = 1
			}
			if tr.ScaleY == 0 {
				tr.ScaleY = 1
			}
			tr.Rotation = 0
			if err := ecs.Add(world, ge, component.TransformComponent.Kind(), tr); err != nil {
				return err
			}
			if err := applyAreaBoundsComponent(world, ge, bounds); err != nil {
				return err
			}
			if err := applyAreaTileStampRotation(world, ge, levelEntityVisualRotation(props)); err != nil {
				return err
			}

			body, ok := ecs.Get(world, ge, component.PhysicsBodyComponent.Kind())
			if !ok || body == nil {
				body = &component.PhysicsBody{}
			}
			body.Width = bounds.W
			body.Height = bounds.H
			body.OffsetX = bounds.W / 2
			body.OffsetY = bounds.H / 2
			body.AlignTopLeft = false
			body.Body = nil
			body.Shape = nil
			if err := ecs.Add(world, ge, component.PhysicsBodyComponent.Kind(), body); err != nil {
				return err
			}

			node, ok := ecs.Get(world, ge, component.ArenaNodeComponent.Kind())
			if ok && node != nil {
				if group := getString("group"); group != "" {
					node.Group = group
				}
				node.Active = getBool("active", node.Active)
				if err := ecs.Add(world, ge, component.ArenaNodeComponent.Kind(), node); err != nil {
					return err
				}
			}

			if err := ecs.Add(world, ge, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: gameEntityID}); err != nil {
				return err
			}
			if err := ecs.Add(world, ge, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: layerIndex}); err != nil {
				return err
			}
		case "trigger":
			if prefabPath == "" {
				prefabPath = "trigger.yaml"
			}
			te, err := BuildEntityWithOverrides(world, prefabPath, componentOverrides)
			if err != nil {
				return err
			}

			tr, ok := ecs.Get(world, te, component.TransformComponent.Kind())
			if !ok || tr == nil {
				tr = &component.Transform{}
			}
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

			bounds := levelEntityAreaBounds(props, false)

			triggerComp, ok := ecs.Get(world, te, component.TriggerComponent.Kind())
			if !ok || triggerComp == nil {
				triggerComp = &component.Trigger{}
			}
			triggerComp.Name = getString("id")
			if triggerComp.Name == "" {
				triggerComp.Name = gameEntityID
			}
			triggerComp.Bounds = bounds
			triggerComp.Disabled = getBool("disabled", triggerComp.Disabled)
			if err := ecs.Add(world, te, component.TriggerComponent.Kind(), triggerComp); err != nil {
				return err
			}
			if err := applyAreaBoundsComponent(world, te, bounds); err != nil {
				return err
			}

			if err := ecs.Add(world, te, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: gameEntityID}); err != nil {
				return err
			}
			if err := ecs.Add(world, te, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: layerIndex}); err != nil {
				return err
			}
		case "breakable_wall":
			if prefabPath == "" {
				prefabPath = "breakable_wall.yaml"
			}
			be, err := BuildEntityWithOverrides(world, prefabPath, componentOverrides)
			if err != nil {
				return err
			}

			bounds := levelEntityAreaBounds(props, true)

			tr, ok := ecs.Get(world, be, component.TransformComponent.Kind())
			if !ok || tr == nil {
				tr = &component.Transform{}
			}
			tr.X = float64(ent.X)
			tr.Y = float64(ent.Y)
			if tr.ScaleX == 0 {
				tr.ScaleX = 1
			}
			if tr.ScaleY == 0 {
				tr.ScaleY = 1
			}
			tr.Rotation = 0
			if err := ecs.Add(world, be, component.TransformComponent.Kind(), tr); err != nil {
				return err
			}
			if err := applyAreaBoundsComponent(world, be, bounds); err != nil {
				return err
			}
			if err := applyAreaTileStampRotation(world, be, levelEntityVisualRotation(props)); err != nil {
				return err
			}

			if body, ok := ecs.Get(world, be, component.PhysicsBodyComponent.Kind()); ok && body != nil {
				body.Width = bounds.W
				body.Height = bounds.H
				body.OffsetX = bounds.W / 2
				body.OffsetY = bounds.H / 2
				body.AlignTopLeft = false
				body.Body = nil
				body.Shape = nil
				if err := ecs.Add(world, be, component.PhysicsBodyComponent.Kind(), body); err != nil {
					return err
				}
			}

			if hurtboxes, ok := ecs.Get(world, be, component.HurtboxComponent.Kind()); ok && hurtboxes != nil {
				newHurtboxes := make([]component.Hurtbox, len(*hurtboxes))
				for i, hb := range *hurtboxes {
					newHurtboxes[i] = component.Hurtbox{
						Width:   hb.Width,
						Height:  hb.Height,
						OffsetX: hb.OffsetX + bounds.W/2,
						OffsetY: hb.OffsetY + bounds.H/2,
					}
				}
				*hurtboxes = newHurtboxes
				if err := ecs.Add(world, be, component.HurtboxComponent.Kind(), hurtboxes); err != nil {
					return err
				}
			}

			if err := ecs.Add(world, be, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: gameEntityID}); err != nil {
				return err
			}
			if err := ecs.Add(world, be, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: layerIndex}); err != nil {
				return err
			}
		default:
			if prefabPath == "" {
				// Unknown entity type with no explicit prefab.
				continue
			}

			e, err := BuildEntityWithOverrides(world, prefabPath, componentOverrides)
			if err != nil {
				return err
			}

			rot := 0.0
			if entityType == "spike" {
				rotDeg := toFloat64(ent.Props["rotation"])
				rot = rotDeg * math.Pi / 180.0
			}

			x := float64(ent.X)
			y := float64(ent.Y)
			if entityType == "spike" {
				if s, ok := ecs.Get(world, e, component.SpriteComponent.Kind()); ok && s != nil {
					x += s.OriginX
					y += s.OriginY
				}
			}

			if err := SetEntityTransform(world, e, x, y, rot); err != nil {
				return err
			}

			if entityType == "spike" {
				spikes = append(spikes, loadedSpikePlacement{
					x:          ent.X,
					y:          ent.Y,
					rotation:   normalizeSpikeRotation(toFloat64(ent.Props["rotation"])),
					layerIndex: layerIndex,
				})
				ecs.Remove(world, e, component.HazardComponent.Kind())
			}

			if err := ecs.Add(world, e, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: gameEntityID}); err != nil {
				return err
			}
			if err := ecs.Add(world, e, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: layerIndex}); err != nil {
				return err
			}
		}
	}

	if err := addMergedSpikeHazards(world, spikes); err != nil {
		return err
	}

	return nil
}

func addMergedSpikeHazards(world *ecs.World, spikes []loadedSpikePlacement) error {
	for _, hazard := range buildMergedSpikeHazards(spikes) {
		e := ecs.CreateEntity(world)
		if err := ecs.Add(world, e, component.TransformComponent.Kind(), &component.Transform{
			X:      hazard.x + hazard.w/2,
			Y:      hazard.y + hazard.h/2,
			ScaleX: 1,
			ScaleY: 1,
		}); err != nil {
			return err
		}
		if err := ecs.Add(world, e, component.HazardComponent.Kind(), &component.Hazard{
			Width:  hazard.w,
			Height: hazard.h,
		}); err != nil {
			return err
		}
		if err := ecs.Add(world, e, component.SpikeTagComponent.Kind(), &component.SpikeTag{}); err != nil {
			return err
		}
		if err := ecs.Add(world, e, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: hazard.layerIndex}); err != nil {
			return err
		}
	}

	return nil
}

func buildMergedSpikeHazards(spikes []loadedSpikePlacement) []mergedSpikeHazard {
	if len(spikes) == 0 {
		return nil
	}

	type runKey struct {
		layerIndex int
		rotation   int
		line       int
	}

	groups := make(map[runKey][]loadedSpikePlacement, len(spikes))
	for _, spike := range spikes {
		line := spike.y
		if spikeRunIsVertical(spike.rotation) {
			line = spike.x
		}
		key := runKey{layerIndex: spike.layerIndex, rotation: spike.rotation, line: line}
		groups[key] = append(groups[key], spike)
	}

	hazards := make([]mergedSpikeHazard, 0, len(groups))
	for _, group := range groups {
		sort.Slice(group, func(i, j int) bool {
			if spikeRunIsVertical(group[i].rotation) {
				return group[i].y < group[j].y
			}
			return group[i].x < group[j].x
		})

		runStart := 0
		for i := 1; i <= len(group); i++ {
			if i < len(group) && spikeRunGap(group[i-1], group[i]) == spikeCellSize {
				continue
			}
			hazards = append(hazards, mergedSpikeHazardForRun(group[runStart:i]))
			runStart = i
		}
	}

	sort.Slice(hazards, func(i, j int) bool {
		if hazards[i].layerIndex != hazards[j].layerIndex {
			return hazards[i].layerIndex < hazards[j].layerIndex
		}
		if hazards[i].y != hazards[j].y {
			return hazards[i].y < hazards[j].y
		}
		return hazards[i].x < hazards[j].x
	})

	return hazards
}

func mergedSpikeHazardForRun(run []loadedSpikePlacement) mergedSpikeHazard {
	minX := math.Inf(1)
	minY := math.Inf(1)
	maxX := math.Inf(-1)
	maxY := math.Inf(-1)

	for _, spike := range run {
		bounds := singleSpikeHazardBounds(spike)
		if bounds.x < minX {
			minX = bounds.x
		}
		if bounds.y < minY {
			minY = bounds.y
		}
		if bounds.x+bounds.w > maxX {
			maxX = bounds.x + bounds.w
		}
		if bounds.y+bounds.h > maxY {
			maxY = bounds.y + bounds.h
		}
	}

	if len(run) > 1 {
		if spikeRunIsVertical(run[0].rotation) {
			trim := math.Min(spikeHazardEndInset, (maxY-minY)/2)
			minY += trim
			maxY -= trim
		} else {
			trim := math.Min(spikeHazardEndInset, (maxX-minX)/2)
			minX += trim
			maxX -= trim
		}
	}

	return mergedSpikeHazard{
		x:          minX,
		y:          minY,
		w:          maxX - minX,
		h:          maxY - minY,
		layerIndex: run[0].layerIndex,
	}
}

func singleSpikeHazardBounds(spike loadedSpikePlacement) mergedSpikeHazard {
	x := float64(spike.x)
	y := float64(spike.y)
	left := x
	top := y + spikeHazardBaseOffsetY
	if spike.rotation == 0 {
		return mergedSpikeHazard{x: left, y: top, w: spikeHazardBaseWidth, h: spikeHazardBaseHeight}
	}

	cx := x + spikeHazardOrigin
	cy := y + spikeHazardOrigin
	angle := float64(spike.rotation) * math.Pi / 180.0
	cosR := math.Cos(angle)
	sinR := math.Sin(angle)
	corners := [4][2]float64{
		{left, top},
		{left + spikeHazardBaseWidth, top},
		{left, top + spikeHazardBaseHeight},
		{left + spikeHazardBaseWidth, top + spikeHazardBaseHeight},
	}

	minX := math.Inf(1)
	minY := math.Inf(1)
	maxX := math.Inf(-1)
	maxY := math.Inf(-1)
	for _, corner := range corners {
		dx := corner[0] - cx
		dy := corner[1] - cy
		rx := dx*cosR - dy*sinR + cx
		ry := dx*sinR + dy*cosR + cy
		if rx < minX {
			minX = rx
		}
		if ry < minY {
			minY = ry
		}
		if rx > maxX {
			maxX = rx
		}
		if ry > maxY {
			maxY = ry
		}
	}

	return mergedSpikeHazard{x: minX, y: minY, w: maxX - minX, h: maxY - minY}
}

func spikeRunGap(a, b loadedSpikePlacement) int {
	if spikeRunIsVertical(a.rotation) {
		return b.y - a.y
	}
	return b.x - a.x
}

func spikeRunIsVertical(rotation int) bool {
	return rotation == 90 || rotation == 270
}

func normalizeSpikeRotation(rotation float64) int {
	rotation = math.Mod(rotation, 360)
	if rotation < 0 {
		rotation += 360
	}
	return (int(math.Round(rotation/90.0)) * 90) % 360
}

func LoadLevelLayerToWorld(world *ecs.World, lvl *levels.Level, layerIdx int, tileSize float64) error {
	if world == nil {
		return fmt.Errorf("world is nil")
	}
	if lvl == nil {
		return fmt.Errorf("level is nil")
	}
	if layerIdx < 0 || layerIdx >= len(lvl.Layers) {
		return fmt.Errorf("layer index %d out of bounds", layerIdx)
	}

	imgs := make(map[string]*ebiten.Image)
	layer := lvl.Layers[layerIdx]
	layerHasPhysics := layerIdx < len(lvl.LayerMeta) && lvl.LayerMeta[layerIdx].Physics
	var layerUsage []*levels.TileInfo
	if layerIdx < len(lvl.TilesetUsage) {
		layerUsage = lvl.TilesetUsage[layerIdx]
	}

	for y := 0; y < lvl.Height; y++ {
		for x := 0; x < lvl.Width; x++ {
			tileIdx := y*lvl.Width + x
			if tileIdx < 0 || tileIdx >= len(layer) {
				continue
			}
			tileID := layer[tileIdx]
			tileInfo := tileInfoAt(layerUsage, tileIdx)
			if !levelTileOccupied(tileID, tileInfo) {
				continue
			}
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
			if err := ecs.Add(world, e, component.TransformComponent.Kind(), &component.Transform{
				X:      float64(x) * tileSize,
				Y:      float64(y) * tileSize,
				ScaleX: 1,
				ScaleY: 1,
			}); err != nil {
				return err
			}
			if err := ecs.Add(world, e, component.SpriteComponent.Kind(), &component.Sprite{
				Image:     img,
				Source:    image.Rect(srcX, srcY, srcX+tileW, srcY+tileH),
				UseSource: true,
				OriginX:   0,
				OriginY:   0,
			}); err != nil {
				return err
			}
			if err := ecs.Add(world, e, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: layerIdx}); err != nil {
				return err
			}
			if err := ecs.Add(world, e, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: layerIdx}); err != nil {
				return err
			}
			if err := ecs.Add(world, e, component.StaticTileComponent.Kind(), &component.StaticTile{}); err != nil {
				return err
			}
			// mark static tile batch dirty on the level bounds entity
			if b, ok := ecs.First(world, component.LevelGridComponent.Kind()); ok {
				if st, ok := ecs.Get(world, b, component.StaticTileBatchStateComponent.Kind()); ok && st != nil {
					st.Dirty = true
				}
			}
		}
	}

	if layerHasPhysics {
		_ = layerHasPhysics
	}

	return nil
}

func BuildLevelGridData(lvl *levels.Level, tileSize float64) *component.LevelGrid {
	return buildLevelGridData(lvl, tileSize)
}

func RebuildMergedLevelPhysics(world *ecs.World, lvl *levels.Level, tileSize float64) error {
	if world == nil {
		return fmt.Errorf("world is nil")
	}
	clearMergedLevelPhysics(world)
	if lvl == nil || lvl.Width <= 0 || lvl.Height <= 0 {
		return nil
	}
	return addMergedTileCollidersFromMask(world, buildMergedPhysicsMask(lvl), lvl.Width, lvl.Height, tileSize)
}

func buildLevelGridData(lvl *levels.Level, tileSize float64) *component.LevelGrid {
	grid := &component.LevelGrid{TileSize: tileSize}
	if lvl == nil || lvl.Width <= 0 || lvl.Height <= 0 {
		return grid
	}

	cellCount := lvl.Width * lvl.Height
	grid.Width = lvl.Width
	grid.Height = lvl.Height
	grid.Occupied = make([]bool, cellCount)
	grid.Solid = make([]bool, cellCount)

	for layerIdx, layer := range lvl.Layers {
		if !levelLayerActive(lvl, layerIdx) {
			continue
		}
		layerHasPhysics := layerIdx < len(lvl.LayerMeta) && lvl.LayerMeta[layerIdx].Physics
		var layerUsage []*levels.TileInfo
		if layerIdx < len(lvl.TilesetUsage) {
			layerUsage = lvl.TilesetUsage[layerIdx]
		}

		maxIndex := minInt(cellCount, len(layer))
		for idx := 0; idx < maxIndex; idx++ {
			if !levelTileOccupied(layer[idx], tileInfoAt(layerUsage, idx)) {
				continue
			}
			grid.Occupied[idx] = true
			if layerHasPhysics {
				grid.Solid[idx] = true
			}
		}
	}

	return grid
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func levelLayerActive(lvl *levels.Level, index int) bool {
	if lvl == nil || index < 0 {
		return false
	}
	if index >= len(lvl.LayerMeta) {
		return true
	}
	return lvl.LayerMeta[index].IsActive()
}

func levelLayerHasPhysics(lvl *levels.Level, index int) bool {
	if lvl == nil || index < 0 || index >= len(lvl.LayerMeta) {
		return false
	}
	return lvl.LayerMeta[index].Physics
}

func prefabPathForLevelEntity(entityType string, props map[string]interface{}) string {
	if props != nil {
		if v, ok := props["prefab"].(string); ok && v != "" {
			return v
		}
	}

	return entityType + ".yaml"
}

func addBreakableWallVisualTiles(world *ecs.World, lvl *levels.Level, root ecs.Entity, x, y int, width, height float64, layerIndex int) error {
	if world == nil || lvl == nil {
		return nil
	}
	sprite, ok := ecs.Get(world, root, component.SpriteComponent.Kind())
	if !ok || sprite == nil || sprite.Image == nil {
		return nil
	}
	renderLayer := &component.RenderLayer{Index: layerIndex}
	if existing, ok := ecs.Get(world, root, component.RenderLayerComponent.Kind()); ok && existing != nil {
		copied := *existing
		renderLayer = &copied
	}
	widthCells := maxInt(1, int(math.Round(width/float64(spikeCellSize))))
	heightCells := maxInt(1, int(math.Round(height/float64(spikeCellSize))))
	baseCellX := x / spikeCellSize
	baseCellY := y / spikeCellSize
	for row := 0; row < heightCells; row++ {
		for col := 0; col < widthCells; col++ {
			cellX := baseCellX + col
			cellY := baseCellY + row
			visual := ecs.CreateEntity(world)
			if err := ecs.Add(world, visual, component.TransformComponent.Kind(), &component.Transform{
				X:        float64(cellX * spikeCellSize),
				Y:        float64(cellY * spikeCellSize),
				ScaleX:   1,
				ScaleY:   1,
				Rotation: breakableWallRotationForLoadedCell(lvl, baseCellX, baseCellY, widthCells, heightCells, cellX, cellY),
			}); err != nil {
				return err
			}
			spriteCopy := *sprite
			spriteCopy.Disabled = false
			if err := ecs.Add(world, visual, component.SpriteComponent.Kind(), &spriteCopy); err != nil {
				return err
			}
			renderLayerCopy := *renderLayer
			if err := ecs.Add(world, visual, component.RenderLayerComponent.Kind(), &renderLayerCopy); err != nil {
				return err
			}
			if err := ecs.Add(world, visual, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: layerIndex}); err != nil {
				return err
			}
		}
	}
	return nil
}

func breakableWallRotationForLoadedCell(lvl *levels.Level, leftCell, topCell, widthCells, heightCells, cellX, cellY int) float64 {
	if lvl == nil {
		return 0
	}
	rightCell := leftCell + widthCells - 1
	bottomCell := topCell + heightCells - 1
	openChecks := []struct {
		nextX    int
		nextY    int
		rotation float64
	}{
		{nextX: cellX, nextY: cellY - 1, rotation: 0},
		{nextX: cellX + 1, nextY: cellY, rotation: math.Pi / 2},
		{nextX: cellX, nextY: cellY + 1, rotation: math.Pi},
		{nextX: cellX - 1, nextY: cellY, rotation: 3 * math.Pi / 2},
	}
	for _, check := range openChecks {
		insideWall := check.nextX >= leftCell && check.nextX <= rightCell && check.nextY >= topCell && check.nextY <= bottomCell
		if insideWall {
			continue
		}
		if check.nextX < 0 || check.nextY < 0 || check.nextX >= lvl.Width || check.nextY >= lvl.Height || !levelCellSolid(lvl, check.nextX, check.nextY) {
			return check.rotation
		}
	}
	return 0
}

func levelCellSolid(lvl *levels.Level, cellX, cellY int) bool {
	if lvl == nil || cellX < 0 || cellY < 0 || cellX >= lvl.Width || cellY >= lvl.Height {
		return false
	}
	cellIndex := cellY*lvl.Width + cellX
	for layerIdx, layer := range lvl.Layers {
		if !levelLayerActive(lvl, layerIdx) || !levelLayerHasPhysics(lvl, layerIdx) || cellIndex < 0 || cellIndex >= len(layer) {
			continue
		}
		var usage []*levels.TileInfo
		if layerIdx < len(lvl.TilesetUsage) {
			usage = lvl.TilesetUsage[layerIdx]
		}
		if levelTileOccupied(layer[cellIndex], tileInfoAt(usage, cellIndex)) {
			return true
		}
	}
	return false
}

func componentOverridesFromLevelProps(props map[string]interface{}) map[string]any {
	if props == nil {
		return nil
	}
	raw, ok := props["components"]
	if !ok || raw == nil {
		return nil
	}
	overrides, ok := raw.(map[string]interface{})
	if !ok || len(overrides) == 0 {
		return nil
	}
	converted := make(map[string]any, len(overrides))
	for key, value := range overrides {
		converted[key] = value
	}
	return converted
}

func levelEntityAreaBounds(props map[string]interface{}, clampToTile bool) component.AABB {
	width := 0.0
	height := 0.0
	if props != nil {
		width = toFloat64(props["w"])
		height = toFloat64(props["h"])
	}
	if width <= 0 {
		width = float64(spikeCellSize)
	}
	if height <= 0 {
		height = float64(spikeCellSize)
	}
	if clampToTile {
		if width < float64(spikeCellSize) {
			width = float64(spikeCellSize)
		}
		if height < float64(spikeCellSize) {
			height = float64(spikeCellSize)
		}
	}
	return component.AABB{W: width, H: height}
}

func levelEntityVisualRotation(props map[string]interface{}) float64 {
	if props == nil {
		return 0
	}
	if overrides := componentOverridesFromLevelProps(props); overrides != nil {
		if rawTransform, ok := overrides["transform"]; ok {
			if typed, ok := rawTransform.(map[string]interface{}); ok {
				if rotation, ok := typed["rotation"]; ok {
					return toFloat64(rotation) * math.Pi / 180.0
				}
			}
		}
	}
	return toFloat64(props["rotation"]) * math.Pi / 180.0
}

func applyAreaBoundsComponent(world *ecs.World, entity ecs.Entity, bounds component.AABB) error {
	areaBounds, ok := ecs.Get(world, entity, component.AreaBoundsComponent.Kind())
	if !ok || areaBounds == nil {
		areaBounds = &component.AreaBounds{}
	}
	areaBounds.Bounds = bounds
	return ecs.Add(world, entity, component.AreaBoundsComponent.Kind(), areaBounds)
}

func applyAreaTileStampRotation(world *ecs.World, entity ecs.Entity, rotation float64) error {
	stamp, ok := ecs.Get(world, entity, component.AreaTileStampComponent.Kind())
	if !ok || stamp == nil {
		return nil
	}
	stamp.RotationOffset = rotation
	if stamp.TileWidth <= 0 {
		stamp.TileWidth = float64(spikeCellSize)
	}
	if stamp.TileHeight <= 0 {
		stamp.TileHeight = float64(spikeCellSize)
	}
	return ecs.Add(world, entity, component.AreaTileStampComponent.Kind(), stamp)
}

func levelEntityLayerIndex(props map[string]interface{}) int {
	if props == nil {
		return 0
	}
	layer := int(toFloat64(props["layer"]))
	if layer < 0 {
		return 0
	}
	return layer
}

func toFloat64(v interface{}) float64 {
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

func addMergedTileColliders(world *ecs.World, layerIndex int, layer []int, usage []*levels.TileInfo, width, height int, tileSize float64) error {
	mask := make([]bool, width*height)
	maxIndex := minInt(len(mask), len(layer))
	for idx := 0; idx < maxIndex; idx++ {
		if levelTileOccupied(layer[idx], tileInfoAt(usage, idx)) {
			mask[idx] = true
		}
	}
	return addMergedTileCollidersFromMask(world, mask, width, height, tileSize)
}

func addMergedTileCollidersFromMask(world *ecs.World, solid []bool, width, height int, tileSize float64) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	visited := make([]bool, width*height)
	index := func(x, y int) int { return y*width + x }

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := index(x, y)
			if idx < 0 || idx >= len(solid) {
				continue
			}
			if visited[idx] || !solid[idx] {
				continue
			}

			maxW := 0
			for x2 := x; x2 < width; x2++ {
				idx2 := index(x2, y)
				if idx2 >= len(solid) || visited[idx2] || !solid[idx2] {
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
					if idx2 >= len(solid) || visited[idx2] || !solid[idx2] {
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
				Width:    float64(maxW) * tileSize,
				Height:   float64(maxH) * tileSize,
				Friction: 0.9,
				Static:   true,
				OffsetX:  float64(maxW) * tileSize / 2,
				OffsetY:  float64(maxH) * tileSize / 2,
			}); err != nil {
				return err
			}
			if err := ecs.Add(world, e, component.MergedLevelPhysicsComponent.Kind(), &component.MergedLevelPhysics{}); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildMergedPhysicsMask(lvl *levels.Level) []bool {
	if lvl == nil || lvl.Width <= 0 || lvl.Height <= 0 {
		return nil
	}
	solid := make([]bool, lvl.Width*lvl.Height)
	for layerIdx, layer := range lvl.Layers {
		if !levelLayerActive(lvl, layerIdx) || !levelLayerHasPhysics(lvl, layerIdx) {
			continue
		}
		var usage []*levels.TileInfo
		if layerIdx < len(lvl.TilesetUsage) {
			usage = lvl.TilesetUsage[layerIdx]
		}
		maxIndex := minInt(len(solid), len(layer))
		for idx := 0; idx < maxIndex; idx++ {
			if levelTileOccupied(layer[idx], tileInfoAt(usage, idx)) {
				solid[idx] = true
			}
		}
	}
	return solid
}

func clearMergedLevelPhysics(world *ecs.World) {
	entities := make([]ecs.Entity, 0, 16)
	ecs.ForEach(world, component.MergedLevelPhysicsComponent.Kind(), func(e ecs.Entity, _ *component.MergedLevelPhysics) {
		entities = append(entities, e)
	})
	for _, e := range entities {
		ecs.DestroyEntity(world, e)
	}
}

func tileInfoAt(usage []*levels.TileInfo, index int) *levels.TileInfo {
	if index < 0 || index >= len(usage) {
		return nil
	}
	return usage[index]
}

func levelTileOccupied(tileID int, info *levels.TileInfo) bool {
	return info != nil || tileID > 0
}
