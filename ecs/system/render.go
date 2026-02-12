package system

import (
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type RenderSystem struct {
	camEntity ecs.Entity
}

func NewRenderSystem() *RenderSystem {
	return &RenderSystem{}
}

func (r *RenderSystem) Draw(w *ecs.World, screen *ebiten.Image) {
	if r == nil {
		return
	}

	// The world is recreated on level transitions. Entity IDs can be reused across
	// worlds, so a cached entity may still be "alive" but refer to the wrong thing.
	// Validate the required components before reusing the cached camera.
	if r.camEntity.Valid() && w.IsAlive(r.camEntity) {
		if !ecs.Has(w, r.camEntity, component.CameraComponent) {
			r.camEntity = 0
		}
	}

	if !r.camEntity.Valid() || !w.IsAlive(r.camEntity) {
		if camEntity, ok := w.First(component.CameraComponent.Kind()); ok {
			r.camEntity = camEntity
		}
	}

	camX, camY := 0.0, 0.0
	zoom := 1.0
	// Fetch the camera entity's transform
	if camTransform, ok := ecs.Get(w, r.camEntity, component.TransformComponent); ok {
		camX = camTransform.X
		camY = camTransform.Y
	}
	if camComp, ok := ecs.Get(w, r.camEntity, component.CameraComponent); ok {
		if camComp.Zoom > 0 {
			zoom = camComp.Zoom
		}
	}

	entities := w.Query(component.TransformComponent.Kind(), component.SpriteComponent.Kind())
	sort.SliceStable(entities, func(i, j int) bool {
		li := 0
		if layer, ok := ecs.Get(w, entities[i], component.RenderLayerComponent); ok {
			li = layer.Index
		}
		lj := 0
		if layer, ok := ecs.Get(w, entities[j], component.RenderLayerComponent); ok {
			lj = layer.Index
		}
		if li != lj {
			return li < lj
		}
		return uint64(entities[i]) < uint64(entities[j])
	})

	for _, e := range entities {
		if e == r.camEntity {
			continue
		}

		line, ok := ecs.Get(w, e, component.LineRenderComponent)
		if ok && line.Width > 0 {
			startX := (line.StartX - camX) * zoom
			startY := (line.StartY - camY) * zoom
			endX := (line.EndX - camX) * zoom
			endY := (line.EndY - camY) * zoom
			vector.StrokeLine(screen, float32(startX), float32(startY), float32(endX), float32(endY), line.Width, line.Color, line.AntiAlias)
		}

		t, ok := ecs.Get(w, e, component.TransformComponent)
		if !ok {
			continue
		}

		s, ok := ecs.Get(w, e, component.SpriteComponent)
		if !ok || s.Image == nil {
			continue
		}

		img := s.Image
		if s.UseSource {
			sub, ok := s.Image.SubImage(s.Source).(*ebiten.Image)
			if ok {
				img = sub
			}
		}

		// Special-case Transitions: draw animated sprite duplicated across the
		// transition "base" (the row/column opposite the enter direction) and
		// stretched to fill the remainder of the transition area. Also rotate to
		// match the enter direction.
		if tr, ok := ecs.Get(w, e, component.TransitionComponent); ok {
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
					angle = 3.141592653589793
				case component.TransitionDirDown:
					angle = 0
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

		// If the entity has an active white-flash component, apply a color transform
		// that turns the sprite fully white while `On` is true.
		if wf, ok := ecs.Get(w, e, component.WhiteFlashComponent); ok {
			if wf.On {
				op.ColorM.Scale(0, 0, 0, 1)
				op.ColorM.Translate(1, 1, 1, 0)
			}
		}

		screen.DrawImage(img, op)
	}

	// Draw transition fade overlay if a runtime exists.
	if rtEnt, ok := w.First(component.TransitionRuntimeComponent.Kind()); ok {
		rt, _ := ecs.Get(w, rtEnt, component.TransitionRuntimeComponent)
		if rt.Alpha > 0 {
			ww, hh := ebiten.Monitor().Size()
			a := rt.Alpha
			if a < 0 {
				a = 0
			}
			if a > 1 {
				a = 1
			}
			vector.FillRect(screen, 0, 0, float32(ww), float32(hh), color.RGBA{A: uint8(a * 255)}, false)
		}
	}
}
