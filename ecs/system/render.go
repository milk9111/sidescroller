package system

import (
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type RenderSystem struct {
	camEntity    ecs.Entity
	sourceCache  map[spriteSourceKey]*ebiten.Image
	drawEntities []ecs.Entity
	batch        staticTileBatch
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

func (r *RenderSystem) Draw(w *ecs.World, screen *ebiten.Image) {
	if r == nil {
		return
	}

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
		camX = camTransform.X
		camY = camTransform.Y
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
		if ecs.Has(w, e, component.StaticTileComponent.Kind()) {
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
		li := 0
		if layer, ok := ecs.Get(w, r.drawEntities[i], component.RenderLayerComponent.Kind()); ok {
			li = layer.Index
		}
		lj := 0
		if layer, ok := ecs.Get(w, r.drawEntities[j], component.RenderLayerComponent.Kind()); ok {
			lj = layer.Index
		}
		if li != lj {
			return li < lj
		}
		return uint64(r.drawEntities[i]) < uint64(r.drawEntities[j])
	})

	for _, e := range r.drawEntities {
		layer := renderLayerIndex(w, e)
		r.drawStaticChunksUpToLayer(screen, visibleChunksByLayer, visibleLayerOrder, drawnStaticLayers, layer, camX, camY, zoom)

		line, ok := ecs.Get(w, e, component.LineRenderComponent.Kind())
		if ok && line.Width > 0 {
			startX := (line.StartX - camX) * zoom
			startY := (line.StartY - camY) * zoom
			endX := (line.EndX - camX) * zoom
			endY := (line.EndY - camY) * zoom
			vector.StrokeLine(screen, float32(startX), float32(startY), float32(endX), float32(endY), line.Width, line.Color, line.AntiAlias)
		}

		t, ok := ecs.Get(w, e, component.TransformComponent.Kind())
		if !ok {
			continue
		}

		s, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
		if !ok || s.Image == nil {
			continue
		}

		img := r.spriteImage(s)
		if img == nil {
			continue
		}

		// Special-case Transitions: draw animated sprite duplicated across the
		// transition "base" (the row/column opposite the enter direction) and
		// stretched to fill the remainder of the transition area. Also rotate to
		// match the enter direction.
		if tr, ok := ecs.Get(w, e, component.TransitionComponent.Kind()); ok {
			if img != nil {
				// Transition bounds are in pixels; assume tile size 32.
				tileSize := 32.0
				areaX := t.X + tr.Bounds.X
				areaY := t.Y + tr.Bounds.Y
				areaW := tr.Bounds.W
				areaH := tr.Bounds.H
				if areaW <= 0 {
					areaW = tileSize
				}
				if areaH <= 0 {
					areaH = tileSize
				}
				cols := int((areaW-1)/tileSize) + 1
				rows := int((areaH-1)/tileSize) + 1
				angle := 0.0
				switch tr.EnterDir {
				case component.TransitionDirLeft:
					angle = -1.5707963267948966
				case component.TransitionDirRight:
					angle = 1.5707963267948966
				case component.TransitionDirUp:
					angle = 0
				case component.TransitionDirDown:
					angle = 3.141592653589793
				default:
					angle = 0
				}

				imgW := float64(img.Bounds().Dx())
				imgH := float64(img.Bounds().Dy())

				// Helper to draw the image stretched to (dw,dh) at (dx,dy) with rotation
				drawSprite := func(dx, dy, dw, dh float64) {
					op := &ebiten.DrawImageOptions{}
					sx := dw / imgW
					sy := dh / imgH
					// translate so rotation/scaling happens about image center
					op.GeoM.Translate(-imgW/2, -imgH/2)
					op.GeoM.Scale(sx, sy)
					op.GeoM.Rotate(angle)
					// apply camera zoom
					op.GeoM.Scale(zoom, zoom)
					// final translate to place center at desired position (translated in screen space)
					cx := dw / 2
					cy := dh / 2
					op.GeoM.Translate((dx+cx-camX)*zoom, (dy+cy-camY)*zoom)
					screen.DrawImage(img, op)
				}

				// For each base tile, draw one stretched sprite covering the strip
				switch tr.EnterDir {
				case component.TransitionDirLeft:
					// base is rightmost column; for each row draw a strip across full width
					for r := 0; r < rows; r++ {
						dy := areaY + float64(r)*tileSize
						dx := areaX
						dw := areaW
						dh := tileSize
						drawSprite(dx, dy, dw, dh)
					}
				case component.TransitionDirRight:
					// base is leftmost column
					for r := 0; r < rows; r++ {
						dy := areaY + float64(r)*tileSize
						dx := areaX
						dw := areaW
						dh := tileSize
						drawSprite(dx, dy, dw, dh)
					}
				case component.TransitionDirUp:
					// base is bottom row; for each column draw a vertical strip
					for c := 0; c < cols; c++ {
						dx := areaX + float64(c)*tileSize
						dy := areaY
						dw := tileSize
						dh := areaH
						drawSprite(dx, dy, dw, dh)
					}
				case component.TransitionDirDown:
					// base is top row
					for c := 0; c < cols; c++ {
						dx := areaX + float64(c)*tileSize
						dy := areaY
						dw := tileSize
						dh := areaH
						drawSprite(dx, dy, dw, dh)
					}
				default:
					// fallback: draw single sprite at entity transform
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(-s.OriginX, -s.OriginY)
					op.GeoM.Scale(t.ScaleX, t.ScaleY)
					op.GeoM.Rotate(t.Rotation)
					op.GeoM.Scale(zoom, zoom)
					op.GeoM.Translate((t.X-camX)*zoom, (t.Y-camY)*zoom)
					screen.DrawImage(img, op)
				}
			}
			continue
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-s.OriginX, -s.OriginY)

		sx := t.ScaleX
		if sx == 0 {
			sx = 1
		}

		if s.FacingLeft {
			sx = -sx
			op.GeoM.Translate(float64(-img.Bounds().Dx()), 0)
		}

		sy := t.ScaleY
		if sy == 0 {
			sy = 1
		}

		op.GeoM.Scale(sx, sy)
		op.GeoM.Rotate(t.Rotation)
		op.GeoM.Scale(zoom, zoom)
		op.GeoM.Translate((t.X-camX)*zoom, (t.Y-camY)*zoom)

		// If the entity has an active white-flash Component.Kind(), apply a color transform
		// that turns the sprite fully white while `On` is true.
		if wf, ok := ecs.Get(w, e, component.WhiteFlashComponent.Kind()); ok {
			if wf.On {
				op.ColorM.Scale(0, 0, 0, 1)
				op.ColorM.Translate(1, 1, 1, 0)
			}
		}

		screen.DrawImage(img, op)
	}
	r.drawStaticChunksUpToLayer(screen, visibleChunksByLayer, visibleLayerOrder, drawnStaticLayers, int(^uint(0)>>1), camX, camY, zoom)

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

func (r *RenderSystem) ensureStaticTileBatch(w *ecs.World) {
	if r == nil || w == nil {
		return
	}
	if r.batch.world == w {
		return
	}
	r.batch = staticTileBatch{world: w, chunkSize: 512}
	r.buildStaticTileBatch(w)
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
		func(_ ecs.Entity, _ *component.StaticTile, t *component.Transform, s *component.Sprite, layer *component.RenderLayer) {
			if t == nil || s == nil || s.Image == nil || layer == nil {
				return
			}
			cx := int(math.Floor(t.X / chunkSize))
			cy := int(math.Floor(t.Y / chunkSize))
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

			sx := d.t.ScaleX
			if sx == 0 {
				sx = 1
			}
			if d.s.FacingLeft {
				sx = -sx
				op.GeoM.Translate(float64(-tileImg.Bounds().Dx()), 0)
			}

			sy := d.t.ScaleY
			if sy == 0 {
				sy = 1
			}

			op.GeoM.Scale(sx, sy)
			op.GeoM.Rotate(d.t.Rotation)

			chunkBaseX := float64(k.cx) * chunkSize
			chunkBaseY := float64(k.cy) * chunkSize
			op.GeoM.Translate(d.t.X-chunkBaseX, d.t.Y-chunkBaseY)
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

func renderLayerIndex(w *ecs.World, e ecs.Entity) int {
	if layer, ok := ecs.Get(w, e, component.RenderLayerComponent.Kind()); ok {
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

func spriteVisibleFast(t *component.Transform, s *component.Sprite, left, top, right, bottom float64) bool {
	if t == nil || s == nil || s.Image == nil {
		return false
	}

	sx := t.ScaleX
	if sx == 0 {
		sx = 1
	}
	sy := t.ScaleY
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

	x1 := t.X - s.OriginX*math.Abs(sx)
	y1 := t.Y - s.OriginY*math.Abs(sy)
	x2 := x1 + w
	y2 := y1 + h

	return x2 >= left && x1 <= right && y2 >= top && y1 <= bottom
}

func transitionVisible(t *component.Transform, tr *component.Transition, left, top, right, bottom float64) bool {
	if t == nil || tr == nil {
		return false
	}
	x1 := t.X + tr.Bounds.X
	y1 := t.Y + tr.Bounds.Y
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
