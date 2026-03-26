package system

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type RenderSystem struct {
	camEntity     ecs.Entity
	sourceCache   map[spriteSourceKey]*ebiten.Image
	drawEntities  []ecs.Entity
	batch         staticTileBatch
	lastLoadSeq   uint64
	lastStaticSig uint64
}

type staticTileBatch struct {
	world     *ecs.World
	chunkSize float64
	chunks    []staticTileChunk
}

type staticTileChunk struct {
	layer int
	x     float64
	y     float64
	w     float64
	h     float64
	img   *ebiten.Image
}

type staticChunkKey struct {
	layer int
	cx    int
	cy    int
}

type staticTileDraw struct {
	t *component.Transform
	s *component.Sprite
}

type spriteSourceKey struct {
	img *ebiten.Image
	src image.Rectangle
}

func NewRenderSystem() *RenderSystem {
	return &RenderSystem{sourceCache: make(map[spriteSourceKey]*ebiten.Image)}
}

func drawLine(target *ebiten.Image, line *component.LineRender, screenSpace bool, camX, camY, zoom float64) {
	if target == nil || line == nil || line.Width <= 0 {
		return
	}
	startX := line.StartX
	startY := line.StartY
	endX := line.EndX
	endY := line.EndY
	if !screenSpace {
		startX = (line.StartX - camX) * zoom
		startY = (line.StartY - camY) * zoom
		endX = (line.EndX - camX) * zoom
		endY = (line.EndY - camY) * zoom
	}
	vector.StrokeLine(target, float32(startX), float32(startY), float32(endX), float32(endY), line.Width, line.Color, line.AntiAlias)
}

func spriteShakeOffset(w *ecs.World, e ecs.Entity) (float64, float64) {
	if w == nil {
		return 0, 0
	}
	shake, ok := ecs.Get(w, e, component.SpriteShakeComponent.Kind())
	if !ok || shake == nil || shake.Frames <= 0 {
		return 0, 0
	}
	return shake.OffsetX, shake.OffsetY
}

func spriteGeoM(w *ecs.World, e ecs.Entity, t *component.Transform, s *component.Sprite, img *ebiten.Image) ebiten.GeoM {
	var geoM ebiten.GeoM
	if t == nil || s == nil || img == nil {
		return geoM
	}

	geoM.Translate(-s.OriginX, -s.OriginY)
	tx, ty, tsx, tsy, trot := resolvedTransform(t)

	sx := tsx
	if sx == 0 {
		sx = 1
	}
	if s.FacingLeft {
		sx = -sx
		geoM.Translate(float64(-img.Bounds().Dx()), 0)
	}

	sy := tsy
	if sy == 0 {
		sy = 1
	}

	geoM.Scale(sx, sy)
	geoM.Rotate(trot)
	geoM.Translate(tx, ty)

	if body, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind()); ok && body != nil {
		if pivotWorldX, pivotWorldY, ok := physicsBodyCenter(w, e, t, body); ok {
			if pivotLocalX, pivotLocalY, ok := spriteBodyPivotLocal(w, e, s, body); ok {
				renderPivotX, renderPivotY := geoM.Apply(pivotLocalX, pivotLocalY)
				geoM.Translate(pivotWorldX-renderPivotX, pivotWorldY-renderPivotY)
			}
		}
	}

	shakeOffsetX, shakeOffsetY := spriteShakeOffset(w, e)
	geoM.Translate(shakeOffsetX, shakeOffsetY)

	return geoM
}

func (r *RenderSystem) Draw(w *ecs.World, screen *ebiten.Image) {
	if r == nil || screen == nil {
		return
	}

	// Use level background color from LevelRuntime if provided, otherwise black
	var bg color.Color = color.Black
	if ent, ok := ecs.First(w, component.LevelRuntimeComponent.Kind()); ok {
		if runtimeComp, ok2 := ecs.Get(w, ent, component.LevelRuntimeComponent.Kind()); ok2 && runtimeComp != nil && runtimeComp.Level != nil {
			if strings.TrimSpace(runtimeComp.Level.BackgroundColor) != "" {
				if parsed, err := parseHexColor(runtimeComp.Level.BackgroundColor); err == nil {
					if c, ok := parsed.(color.NRGBA); ok {
						bg = color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
					} else if c2, ok2 := parsed.(color.RGBA); ok2 {
						bg = c2
					}
				}
			}
		}
	}
	screen.Fill(bg)

	// The world is recreated on level transitions. Entity IDs can be reused across
	// worlds, so a cached entity may still be "alive" but refer to the wrong thing.
	// Validate the required components before reusing the cached camera.
	if r.camEntity.Valid() && ecs.IsAlive(w, r.camEntity) {
		if !ecs.Has(w, r.camEntity, component.CameraComponent.Kind()) {
			r.camEntity = 0
		}
	}

	if !r.camEntity.Valid() || !ecs.IsAlive(w, r.camEntity) {
		if camEntity, ok := ecs.First(w, component.CameraComponent.Kind()); ok {
			r.camEntity = camEntity
		}
	}

	camX, camY := 0.0, 0.0
	zoom := 1.0
	// Fetch the camera entity's transform
	if camTransform, ok := ecs.Get(w, r.camEntity, component.TransformComponent.Kind()); ok {
		camX, camY, _, _, _ = resolvedTransform(camTransform)
	}
	if camComp, ok := ecs.Get(w, r.camEntity, component.CameraComponent.Kind()); ok {
		if camComp.Zoom > 0 {
			zoom = camComp.Zoom
		}
	}

	screenW := float64(screen.Bounds().Dx())
	screenH := float64(screen.Bounds().Dy())
	viewLeft := camX
	viewTop := camY
	viewRight := camX + (screenW / zoom)
	viewBottom := camY + (screenH / zoom)
	levelBounds := activeLevelBounds(w)
	viewLeft, viewTop, viewRight, viewBottom = clampViewToLevelBounds(levelBounds, viewLeft, viewTop, viewRight, viewBottom)
	worldTarget, ok := worldRenderTarget(screen, levelBounds, camX, camY, zoom)
	if !ok {
		worldTarget = nil
	}

	r.ensureStaticTileBatch(w)
	visibleChunks := r.visibleStaticChunks(viewLeft, viewTop, viewRight, viewBottom)
	visibleChunksByLayer, visibleLayerOrder := groupChunksByLayer(visibleChunks)
	drawnStaticLayers := make(map[int]bool, len(visibleChunksByLayer))

	allEntities := ecs.Query2(w, component.TransformComponent.Kind(), component.SpriteComponent.Kind())
	r.drawEntities = r.drawEntities[:0]
	for _, e := range allEntities {
		if e == r.camEntity {
			continue
		}
		if ecs.Has(w, e, component.StaticTileComponent.Kind()) && !ecs.Has(w, e, component.SpriteShakeComponent.Kind()) {
			continue
		}

		t, ok := ecs.Get(w, e, component.TransformComponent.Kind())
		if !ok {
			continue
		}
		s, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
		if !ok || s.Image == nil {
			continue
		}

		if tr, ok := ecs.Get(w, e, component.TransitionComponent.Kind()); ok {
			if transitionVisible(t, tr, viewLeft, viewTop, viewRight, viewBottom) {
				r.drawEntities = append(r.drawEntities, e)
			}
			continue
		}

		// Keep dynamic entities always drawable. Aggressive culling can reject
		// animated/offset sprites incorrectly and make entities disappear.
		r.drawEntities = append(r.drawEntities, e)
	}

	sort.Slice(r.drawEntities, func(i, j int) bool {
		li := drawLayerIndex(w, r.drawEntities[i])
		lj := drawLayerIndex(w, r.drawEntities[j])
		if li != lj {
			return li < lj
		}
		oi := renderOrderIndex(w, r.drawEntities[i])
		oj := renderOrderIndex(w, r.drawEntities[j])
		if oi != oj {
			return oi < oj
		}
		return uint64(r.drawEntities[i]) < uint64(r.drawEntities[j])
	})

	for _, e := range r.drawEntities {
		layer := drawLayerIndex(w, e)
		r.drawStaticChunksUpToLayer(worldTarget, visibleChunksByLayer, visibleLayerOrder, drawnStaticLayers, layer, camX, camY, zoom)

		screenSpace := ecs.Has(w, e, component.ScreenSpaceComponent.Kind())

		line, ok := ecs.Get(w, e, component.LineRenderComponent.Kind())
		if ok && line.Width > 0 && line.BehindEntities {
			target := screen
			if !screenSpace {
				if worldTarget == nil {
					continue
				}
				target = worldTarget
			}
			drawLine(target, line, screenSpace, camX, camY, zoom)
		}

		if ok && line.Width > 0 && !line.BehindEntities {
			target := screen
			if !screenSpace {
				if worldTarget == nil {
					continue
				}
				target = worldTarget
			}
			drawLine(target, line, screenSpace, camX, camY, zoom)
		}

		t, ok := ecs.Get(w, e, component.TransformComponent.Kind())
		if !ok {
			continue
		}

		circle, ok := ecs.Get(w, e, component.CircleRenderComponent.Kind())
		if ok && !circle.Disabled && circle.Radius > 0 && circle.Width > 0 {
			target := screen
			cx, cy, _, _, _ := resolvedTransform(t)
			cx += circle.OffsetX
			cy += circle.OffsetY
			radius := circle.Radius
			if !screenSpace {
				if worldTarget == nil {
					continue
				}
				target = worldTarget
				cx = (cx - camX) * zoom
				cy = (cy - camY) * zoom
				radius *= zoom
			}
			vector.StrokeCircle(target, float32(cx), float32(cy), float32(radius), circle.Width, circle.Color, circle.AntiAlias)
		}

		s, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
		if !ok || s.Image == nil || s.Disabled {
			continue
		}

		img := r.spriteImage(s)
		if img == nil {
			continue
		}

		if stamp, ok := ecs.Get(w, e, component.AreaTileStampComponent.Kind()); ok && stamp != nil {
			if !screenSpace && worldTarget == nil {
				continue
			}

			target := screen
			if !screenSpace {
				target = worldTarget
			}

			if r.drawAreaTileStamp(w, e, target, t, s, stamp, camX, camY, zoom, screenSpace) {
				continue
			}
		}

		if !screenSpace && worldTarget == nil {
			continue
		}

		tx, ty, tsx, tsy, trot := resolvedTransform(t)
		op := &ebiten.DrawImageOptions{}
		op.GeoM = spriteGeoM(w, e, t, s, img)

		target := screen
		if screenSpace {
		} else {
			target = worldTarget
			op.GeoM.Scale(zoom, zoom)
			op.GeoM.Translate(-camX*zoom, -camY*zoom)
		}

		if c, ok := ecs.Get(w, e, component.ColorComponent.Kind()); ok {
			op.ColorM.Scale(c.R, c.G, c.B, c.A)
		}

		if ecs.Has(w, e, component.SpriteBlackoutComponent.Kind()) {
			op.ColorM.Scale(0, 0, 0, 1)
		}

		// If the entity has an active white-flash Component.Kind(), apply a color transform
		// that turns the sprite fully white while `On` is true.
		if wf, ok := ecs.Get(w, e, component.WhiteFlashComponent.Kind()); ok {
			if wf.On {
				op.ColorM.Scale(0, 0, 0, 1)
				op.ColorM.Translate(1, 1, 1, 0)
			}
		}

		if s.TileX || s.TileY {
			r.drawTiledSprite(target, img, s, tx, ty, tsx, tsy, trot, camX, camY, zoom, screenSpace, &op.ColorM)
			continue
		}

		target.DrawImage(img, op)
	}
	r.drawStaticChunksUpToLayer(worldTarget, visibleChunksByLayer, visibleLayerOrder, drawnStaticLayers, int(^uint(0)>>1), camX, camY, zoom)

	// Draw transition fade overlay if a runtime exists.
	if rtEnt, ok := ecs.First(w, component.TransitionRuntimeComponent.Kind()); ok {
		rt, _ := ecs.Get(w, rtEnt, component.TransitionRuntimeComponent.Kind())
		if rt.Alpha > 0 {
			a := rt.Alpha
			if a < 0 {
				a = 0
			}
			if a > 1 {
				a = 1
			}
			vector.FillRect(screen, 0, 0, float32(screenW), float32(screenH), color.RGBA{A: uint8(a * 255)}, false)
		}
	}
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

func (r *RenderSystem) ensureStaticTileBatch(w *ecs.World) {
	if r == nil || w == nil {
		return
	}

	loadSeq := uint64(0)
	if loadedEnt, ok := ecs.First(w, component.LevelLoadedComponent.Kind()); ok {
		if loaded, ok := ecs.Get(w, loadedEnt, component.LevelLoadedComponent.Kind()); ok && loaded != nil {
			loadSeq = loaded.Sequence
		}
	}

	// Only rebuild the static tile batch when the level load sequence changes
	// or when a `StaticTileBatchState` component on the level bounds entity
	// has its `Dirty` flag set by systems that mutate static-tile-related state.
	var st *component.StaticTileBatchState
	if b, ok := ecs.First(w, component.LevelGridComponent.Kind()); ok {
		st, ok = ecs.Get(w, b, component.StaticTileBatchStateComponent.Kind())
	}
	dirty := st != nil && st.Dirty
	staticSig := uint64(0)
	if st == nil {
		staticSig = staticTileBatchSignature(w)
	}

	if r.batch.world == w {
		if (loadSeq != 0 && loadSeq != r.lastLoadSeq) || dirty || (st == nil && staticSig != r.lastStaticSig) {
			chunkSize := r.batch.chunkSize
			if chunkSize <= 0 {
				chunkSize = 512
			}
			r.batch = staticTileBatch{world: w, chunkSize: chunkSize}
			r.buildStaticTileBatch(w)
			if st != nil {
				st.Dirty = false
			}
			r.lastLoadSeq = loadSeq
			r.lastStaticSig = staticSig
		}
		return
	}
	r.batch = staticTileBatch{world: w, chunkSize: 512}
	r.buildStaticTileBatch(w)
	if st != nil {
		st.Dirty = false
	}
	r.lastLoadSeq = loadSeq
	r.lastStaticSig = staticSig
}

func (r *RenderSystem) buildStaticTileBatch(w *ecs.World) {
	if r == nil || w == nil {
		return
	}

	chunkSize := r.batch.chunkSize
	if chunkSize <= 0 {
		chunkSize = 512
		r.batch.chunkSize = chunkSize
	}

	chunkTiles := make(map[staticChunkKey][]staticTileDraw)
	ecs.ForEach4(w,
		component.StaticTileComponent.Kind(),
		component.TransformComponent.Kind(),
		component.SpriteComponent.Kind(),
		component.RenderLayerComponent.Kind(),
		func(e ecs.Entity, _ *component.StaticTile, t *component.Transform, s *component.Sprite, layer *component.RenderLayer) {
			if t == nil || s == nil || s.Image == nil || s.Disabled || layer == nil {
				return
			}
			if ecs.Has(w, e, component.SpriteShakeComponent.Kind()) {
				return
			}
			tx, ty, _, _, _ := resolvedTransform(t)
			cx := int(math.Floor(tx / chunkSize))
			cy := int(math.Floor(ty / chunkSize))
			k := staticChunkKey{layer: layer.Index, cx: cx, cy: cy}
			chunkTiles[k] = append(chunkTiles[k], staticTileDraw{t: t, s: s})
		})

	chunks := make([]staticTileChunk, 0, len(chunkTiles))
	for k, tiles := range chunkTiles {
		img := ebiten.NewImage(int(chunkSize), int(chunkSize))
		for _, d := range tiles {
			if d.t == nil || d.s == nil {
				continue
			}
			tileImg := r.spriteImage(d.s)
			if tileImg == nil {
				continue
			}

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-d.s.OriginX, -d.s.OriginY)
			tx, ty, tsx, tsy, trot := resolvedTransform(d.t)

			sx := tsx
			if sx == 0 {
				sx = 1
			}
			if d.s.FacingLeft {
				sx = -sx
				op.GeoM.Translate(float64(-tileImg.Bounds().Dx()), 0)
			}

			sy := tsy
			if sy == 0 {
				sy = 1
			}

			op.GeoM.Scale(sx, sy)
			op.GeoM.Rotate(trot)

			chunkBaseX := float64(k.cx) * chunkSize
			chunkBaseY := float64(k.cy) * chunkSize
			op.GeoM.Translate(tx-chunkBaseX, ty-chunkBaseY)
			img.DrawImage(tileImg, op)
		}

		chunks = append(chunks, staticTileChunk{
			layer: k.layer,
			x:     float64(k.cx) * chunkSize,
			y:     float64(k.cy) * chunkSize,
			w:     chunkSize,
			h:     chunkSize,
			img:   img,
		})
	}

	sort.Slice(chunks, func(i, j int) bool {
		if chunks[i].layer != chunks[j].layer {
			return chunks[i].layer < chunks[j].layer
		}
		if chunks[i].y != chunks[j].y {
			return chunks[i].y < chunks[j].y
		}
		return chunks[i].x < chunks[j].x
	})

	r.batch.chunks = chunks
}

func staticTileBatchSignature(w *ecs.World) uint64 {
	if w == nil {
		return 0
	}
	var sig uint64 = 1469598103934665603
	ecs.ForEach4(w,
		component.StaticTileComponent.Kind(),
		component.TransformComponent.Kind(),
		component.SpriteComponent.Kind(),
		component.RenderLayerComponent.Kind(),
		func(e ecs.Entity, _ *component.StaticTile, t *component.Transform, s *component.Sprite, layer *component.RenderLayer) {
			sig ^= uint64(e)
			sig *= 1099511628211
			if layer != nil {
				sig ^= uint64(uint32(layer.Index + 1))
				sig *= 1099511628211
			}
			if s != nil {
				if s.Disabled {
					sig ^= 0xff
				} else {
					sig ^= 0x7f
				}
				sig *= 1099511628211
			}
			if ecs.Has(w, e, component.SpriteShakeComponent.Kind()) {
				sig ^= 0x53
				sig *= 1099511628211
			}
			if t != nil {
				tx, ty, _, _, _ := resolvedTransform(t)
				sig ^= uint64(int32(tx)) | (uint64(int32(ty)) << 32)
				sig *= 1099511628211
			}
		})
	return sig
}

func (r *RenderSystem) visibleStaticChunks(left, top, right, bottom float64) []staticTileChunk {
	if r == nil || len(r.batch.chunks) == 0 {
		return nil
	}
	visible := make([]staticTileChunk, 0, len(r.batch.chunks))
	for _, ch := range r.batch.chunks {
		x2 := ch.x + ch.w
		y2 := ch.y + ch.h
		if x2 < left || ch.x > right || y2 < top || ch.y > bottom {
			continue
		}
		visible = append(visible, ch)
	}
	return visible
}

func activeLevelBounds(w *ecs.World) *component.LevelBounds {
	if w == nil {
		return nil
	}
	boundsEntity, ok := ecs.First(w, component.LevelBoundsComponent.Kind())
	if !ok {
		return nil
	}
	bounds, ok := ecs.Get(w, boundsEntity, component.LevelBoundsComponent.Kind())
	if !ok || bounds == nil {
		return nil
	}
	return bounds
}

func clampViewToLevelBounds(bounds *component.LevelBounds, left, top, right, bottom float64) (float64, float64, float64, float64) {
	if bounds == nil {
		return left, top, right, bottom
	}
	if bounds.Width > 0 {
		left = math.Max(left, 0)
		right = math.Min(right, bounds.Width)
	}
	if bounds.Height > 0 {
		top = math.Max(top, 0)
		bottom = math.Min(bottom, bounds.Height)
	}
	return left, top, right, bottom
}

func worldRenderTarget(screen *ebiten.Image, bounds *component.LevelBounds, camX, camY, zoom float64) (*ebiten.Image, bool) {
	if screen == nil {
		return nil, false
	}
	clipRect, ok := worldClipRect(screen.Bounds(), bounds, camX, camY, zoom)
	if !ok {
		return nil, false
	}
	if clipRect == screen.Bounds() {
		return screen, true
	}
	sub, ok := screen.SubImage(clipRect).(*ebiten.Image)
	if !ok {
		return screen, true
	}
	return sub, true
}

func worldClipRect(screenBounds image.Rectangle, bounds *component.LevelBounds, camX, camY, zoom float64) (image.Rectangle, bool) {
	if screenBounds.Empty() {
		return image.Rectangle{}, false
	}
	if zoom <= 0 || bounds == nil {
		return screenBounds, true
	}
	minX := 0.0
	minY := 0.0
	maxX := float64(screenBounds.Dx())
	maxY := float64(screenBounds.Dy())
	if bounds.Width > 0 {
		minX = math.Max(minX, (-camX)*zoom)
		maxX = math.Min(maxX, (bounds.Width-camX)*zoom)
	}
	if bounds.Height > 0 {
		minY = math.Max(minY, (-camY)*zoom)
		maxY = math.Min(maxY, (bounds.Height-camY)*zoom)
	}
	clip := image.Rect(
		screenBounds.Min.X+int(math.Floor(minX)),
		screenBounds.Min.Y+int(math.Floor(minY)),
		screenBounds.Min.X+int(math.Ceil(maxX)),
		screenBounds.Min.Y+int(math.Ceil(maxY)),
	)
	clip = clip.Intersect(screenBounds)
	if clip.Empty() {
		return image.Rectangle{}, false
	}
	return clip, true
}

func (r *RenderSystem) drawStaticChunksUpToLayer(screen *ebiten.Image, chunksByLayer map[int][]staticTileChunk, layerOrder []int, drawn map[int]bool, maxLayer int, camX, camY, zoom float64) {
	if screen == nil {
		return
	}
	if len(chunksByLayer) == 0 || len(layerOrder) == 0 {
		return
	}
	for _, layer := range layerOrder {
		if layer > maxLayer {
			break
		}
		if drawn[layer] {
			continue
		}
		chunks := chunksByLayer[layer]
		for _, ch := range chunks {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(zoom, zoom)
			op.GeoM.Translate((ch.x-camX)*zoom, (ch.y-camY)*zoom)
			screen.DrawImage(ch.img, op)
		}
		drawn[layer] = true
	}
}

func groupChunksByLayer(chunks []staticTileChunk) (map[int][]staticTileChunk, []int) {
	if len(chunks) == 0 {
		return nil, nil
	}
	byLayer := make(map[int][]staticTileChunk)
	for _, ch := range chunks {
		byLayer[ch.layer] = append(byLayer[ch.layer], ch)
	}
	layers := make([]int, 0, len(byLayer))
	for layer := range byLayer {
		layers = append(layers, layer)
	}
	sort.Ints(layers)
	return byLayer, layers
}

func drawLayerIndex(w *ecs.World, e ecs.Entity) int {
	if layer, ok := ecs.Get(w, e, component.EntityLayerComponent.Kind()); ok && layer != nil {
		return layer.Index
	}
	if layer, ok := ecs.Get(w, e, component.RenderLayerComponent.Kind()); ok {
		return layer.Index
	}
	return 0
}

func renderOrderIndex(w *ecs.World, e ecs.Entity) int {
	if !ecs.Has(w, e, component.EntityLayerComponent.Kind()) {
		return 0
	}
	if layer, ok := ecs.Get(w, e, component.RenderLayerComponent.Kind()); ok && layer != nil {
		return layer.Index
	}
	return 0
}

func (r *RenderSystem) spriteImage(s *component.Sprite) *ebiten.Image {
	if s == nil || s.Image == nil {
		return nil
	}
	if !s.UseSource {
		return s.Image
	}
	if r.sourceCache == nil {
		r.sourceCache = make(map[spriteSourceKey]*ebiten.Image)
	}
	key := spriteSourceKey{img: s.Image, src: s.Source}
	if cached, ok := r.sourceCache[key]; ok {
		return cached
	}
	sub, ok := s.Image.SubImage(s.Source).(*ebiten.Image)
	if !ok {
		return s.Image
	}
	r.sourceCache[key] = sub
	return sub
}

func (r *RenderSystem) drawTiledSprite(target *ebiten.Image, img *ebiten.Image, s *component.Sprite, tx, ty, scaleX, scaleY, rotation, camX, camY, zoom float64, screenSpace bool, colorM *ebiten.ColorM) {
	if target == nil || img == nil || s == nil {
		return
	}

	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	if imgW <= 0 || imgH <= 0 {
		return
	}

	absScaleX := math.Abs(scaleX)
	absScaleY := math.Abs(scaleY)
	if absScaleX == 0 {
		absScaleX = 1
	}
	if absScaleY == 0 {
		absScaleY = 1
	}

	totalW := imgW
	totalH := imgH
	if s.TileX {
		totalW = imgW * absScaleX
	}
	if s.TileY {
		totalH = imgH * absScaleY
	}

	flipX := scaleX < 0
	if s.FacingLeft {
		flipX = !flipX
	}
	flipY := scaleY < 0

	baseScaleX := absScaleX
	if s.TileX {
		baseScaleX = 1
	}
	baseScaleY := absScaleY
	if s.TileY {
		baseScaleY = 1
	}

	for drawY := 0.0; drawY < totalH; drawY += imgH {
		remainingH := totalH - drawY
		tileH := math.Min(imgH, remainingH)
		srcH := int(math.Round(tileH))
		if srcH <= 0 {
			continue
		}

		for drawX := 0.0; drawX < totalW; drawX += imgW {
			remainingW := totalW - drawX
			tileW := math.Min(imgW, remainingW)
			srcW := int(math.Round(tileW))
			if srcW <= 0 {
				continue
			}

			tileImg := img
			if srcW != int(imgW) || srcH != int(imgH) {
				sub, ok := img.SubImage(image.Rect(0, 0, srcW, srcH)).(*ebiten.Image)
				if !ok {
					continue
				}
				tileImg = sub
			}

			op := &ebiten.DrawImageOptions{}
			if colorM != nil {
				op.ColorM = *colorM
			}
			op.GeoM.Translate(-s.OriginX, -s.OriginY)
			op.GeoM.Translate(drawX, drawY)

			tileScaleX := baseScaleX
			if s.TileX {
				tileScaleX = tileW / float64(srcW)
			}
			if flipX {
				op.GeoM.Scale(-tileScaleX, 1)
				op.GeoM.Translate(-tileW, 0)
			} else {
				op.GeoM.Scale(tileScaleX, 1)
			}

			tileScaleY := baseScaleY
			if s.TileY {
				tileScaleY = tileH / float64(srcH)
			}
			if flipY {
				op.GeoM.Scale(1, -tileScaleY)
				op.GeoM.Translate(0, -tileH)
			} else {
				op.GeoM.Scale(1, tileScaleY)
			}

			op.GeoM.Rotate(rotation)
			op.GeoM.Translate(tx, ty)
			if !screenSpace {
				op.GeoM.Scale(zoom, zoom)
				op.GeoM.Translate(-camX*zoom, -camY*zoom)
			}

			target.DrawImage(tileImg, op)
		}
	}
}

func spriteVisibleFast(t *component.Transform, s *component.Sprite, left, top, right, bottom float64) bool {
	if t == nil || s == nil || s.Image == nil {
		return false
	}
	tx, ty, tsx, tsy, _ := resolvedTransform(t)

	sx := tsx
	if sx == 0 {
		sx = 1
	}
	sy := tsy
	if sy == 0 {
		sy = 1
	}

	w := 0.0
	h := 0.0
	if s.UseSource {
		srcW := s.Source.Dx()
		srcH := s.Source.Dy()
		if srcW <= 0 || srcH <= 0 {
			return false
		}
		w = math.Abs(sx) * float64(srcW)
		h = math.Abs(sy) * float64(srcH)
	} else {
		w = math.Abs(sx) * float64(s.Image.Bounds().Dx())
		h = math.Abs(sy) * float64(s.Image.Bounds().Dy())
	}

	x1 := tx - s.OriginX*math.Abs(sx)
	y1 := ty - s.OriginY*math.Abs(sy)
	x2 := x1 + w
	y2 := y1 + h

	return x2 >= left && x1 <= right && y2 >= top && y1 <= bottom
}

func transitionVisible(t *component.Transform, tr *component.Transition, left, top, right, bottom float64) bool {
	if t == nil || tr == nil {
		return false
	}
	tx, ty, _, _, _ := resolvedTransform(t)
	x1 := tx + tr.Bounds.X
	y1 := ty + tr.Bounds.Y
	w := tr.Bounds.W
	h := tr.Bounds.H
	if w <= 0 {
		w = 32
	}
	if h <= 0 {
		h = 32
	}
	x2 := x1 + w
	y2 := y1 + h
	return x2 >= left && x1 <= right && y2 >= top && y1 <= bottom
}

func (r *RenderSystem) drawAreaTileStamp(w *ecs.World, entity ecs.Entity, target *ebiten.Image, transform *component.Transform, sprite *component.Sprite, stamp *component.AreaTileStamp, camX, camY, zoom float64, screenSpace bool) bool {
	if r == nil || w == nil || target == nil || transform == nil || sprite == nil || sprite.Image == nil || sprite.Disabled || stamp == nil {
		return false
	}
	areaBounds, ok := ecs.Get(w, entity, component.AreaBoundsComponent.Kind())
	if !ok || areaBounds == nil {
		return false
	}
	tileW := stamp.TileWidth
	tileH := stamp.TileHeight
	if tileW <= 0 {
		tileW = tileSize
	}
	if tileH <= 0 {
		tileH = tileSize
	}
	bounds := areaBounds.Bounds
	if bounds.W <= 0 {
		bounds.W = tileW
	}
	if bounds.H <= 0 {
		bounds.H = tileH
	}
	tx, ty, _, _, _ := resolvedTransform(transform)
	areaX := tx + bounds.X
	areaY := ty + bounds.Y
	cols := int(math.Ceil(bounds.W / tileW))
	rows := int(math.Ceil(bounds.H / tileH))
	if cols <= 0 {
		cols = 1
	}
	if rows <= 0 {
		rows = 1
	}
	img := r.spriteImage(sprite)
	if img == nil {
		return false
	}
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cellX := areaX + float64(col)*tileW
			cellY := areaY + float64(row)*tileH
			rotation := stamp.RotationOffset + areaTileStampCellRotation(w, entity, stamp, areaX, areaY, areaBounds.Bounds, col, row)
			r.drawAreaTile(w, entity, target, img, sprite, cellX, cellY, tileW, tileH, rotation, camX, camY, zoom, screenSpace)
		}
	}
	return true
}

func (r *RenderSystem) drawAreaTile(w *ecs.World, entity ecs.Entity, target *ebiten.Image, img *ebiten.Image, sprite *component.Sprite, x, y, width, height, rotation, camX, camY, zoom float64, screenSpace bool) {
	if target == nil || img == nil || sprite == nil {
		return
	}
	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	if imgW <= 0 || imgH <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	scaleX := width / imgW
	if sprite.FacingLeft {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(-imgW, 0)
		scaleX = -scaleX
	}
	op.GeoM.Translate(-imgW/2, -imgH/2)
	op.GeoM.Scale(scaleX, height/imgH)
	op.GeoM.Rotate(rotation)
	centerX, centerY := areaTileCenter(w, entity, x, y, width, height)
	if !screenSpace {
		op.GeoM.Scale(zoom, zoom)
		op.GeoM.Translate((centerX-camX)*zoom, (centerY-camY)*zoom)
	} else {
		op.GeoM.Translate(centerX, centerY)
	}

	// Apply color transforms similar to the main sprite renderer so
	// area-tile-stamped visuals respect Color, Blackout, and WhiteFlash.
	if w != nil {
		if c, ok := ecs.Get(w, entity, component.ColorComponent.Kind()); ok {
			op.ColorM.Scale(c.R, c.G, c.B, c.A)
		}

		if ecs.Has(w, entity, component.SpriteBlackoutComponent.Kind()) {
			op.ColorM.Scale(0, 0, 0, 1)
		}

		if wf, ok := ecs.Get(w, entity, component.WhiteFlashComponent.Kind()); ok {
			if wf.On {
				op.ColorM.Scale(0, 0, 0, 1)
				op.ColorM.Translate(1, 1, 1, 0)
			}
		}
	}

	target.DrawImage(img, op)
}

func areaTileCenter(w *ecs.World, entity ecs.Entity, x, y, width, height float64) (float64, float64) {
	centerX := x + width/2
	centerY := y + height/2
	shakeOffsetX, shakeOffsetY := spriteShakeOffset(w, entity)
	return centerX + shakeOffsetX, centerY + shakeOffsetY
}

func areaTileStampCellRotation(w *ecs.World, entity ecs.Entity, stamp *component.AreaTileStamp, areaX, areaY float64, bounds component.AABB, col, row int) float64 {
	if stamp == nil {
		return 0
	}
	switch stamp.RotationMode {
	case component.AreaTileStampRotationTransitionEnter:
		transition, ok := ecs.Get(w, entity, component.TransitionComponent.Kind())
		if !ok || transition == nil {
			return 0
		}
		switch transition.EnterDir {
		case component.TransitionDirRight:
			return math.Pi / 2
		case component.TransitionDirDown:
			return math.Pi
		case component.TransitionDirLeft:
			return 3 * math.Pi / 2
		default:
			return 0
		}
	case component.AreaTileStampRotationOpenNeighbor:
		return openNeighborCellRotation(w, areaX, areaY, bounds, col, row)
	default:
		return 0
	}
}

func openNeighborCellRotation(w *ecs.World, areaX, areaY float64, bounds component.AABB, col, row int) float64 {
	if w == nil {
		return 0
	}
	gridEntity, ok := ecs.First(w, component.LevelGridComponent.Kind())
	if !ok {
		return 0
	}
	grid, ok := ecs.Get(w, gridEntity, component.LevelGridComponent.Kind())
	if !ok || grid == nil || grid.TileSize <= 0 {
		return 0
	}
	leftCell := int(math.Floor(areaX / grid.TileSize))
	topCell := int(math.Floor(areaY / grid.TileSize))
	widthCells := int(math.Round(bounds.W / grid.TileSize))
	heightCells := int(math.Round(bounds.H / grid.TileSize))
	if widthCells < 1 {
		widthCells = 1
	}
	if heightCells < 1 {
		heightCells = 1
	}
	cellX := leftCell + col
	cellY := topCell + row
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
		inside := check.nextX >= leftCell && check.nextX <= rightCell && check.nextY >= topCell && check.nextY <= bottomCell
		if inside {
			continue
		}
		if !grid.InBounds(check.nextX, check.nextY) || !grid.CellSolid(check.nextX, check.nextY) {
			return check.rotation
		}
	}
	return 0
}

func resolvedTransform(t *component.Transform) (x, y, scaleX, scaleY, rotation float64) {
	if t == nil {
		return 0, 0, 1, 1, 0
	}

	if t.Parent != 0 {
		x = t.WorldX
		y = t.WorldY
		scaleX = t.WorldScaleX
		scaleY = t.WorldScaleY
		rotation = t.WorldRotation
	} else {
		x = t.X
		y = t.Y
		scaleX = t.ScaleX
		scaleY = t.ScaleY
		rotation = t.Rotation
	}

	if scaleX == 0 {
		scaleX = 1
	}
	if scaleY == 0 {
		scaleY = 1
	}

	return x, y, scaleX, scaleY, rotation
}
