package system

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	debugCircleSegments = 24
	debugDotSize        = 4
)

func DrawPhysicsDebug(space *cp.Space, w *ecs.World, screen *ebiten.Image) {
	if space == nil || w == nil || screen == nil {
		return
	}

	camX, camY, zoom := debugCameraTransform(w)
	drawer := &physicsDebugDrawer{
		screen: screen,
		camX:   camX,
		camY:   camY,
		zoom:   zoom,
	}
	cp.DrawSpace(space, drawer)

	// Draw hitboxes (red) and hurtboxes (blue) from components
	if w != nil && screen != nil {
		// Hitboxes: show active frames in red outline
		ecs.ForEach3(w, component.HitboxComponent.Kind(), component.TransformComponent.Kind(), component.AnimationComponent.Kind(), func(e ecs.Entity, hbSlice *[]component.Hitbox, t *component.Transform, anim *component.Animation) {
			for _, hb := range *hbSlice {
				active := true
				if hb.Anim != "" {
					active = false
					if anim.Current == hb.Anim {
						for _, f := range hb.Frames {
							if f == anim.Frame {
								active = true
								break
							}
						}
					}
				}
				if !active {
					continue
				}
				// compute world rect (flip horizontally when facing left)
				// compute world rect (flip horizontally when facing left)
				scaleX := t.ScaleX
				if scaleX == 0 {
					scaleX = 1
				}
				baseOff := hb.OffsetX * scaleX
				offX := baseOff
				if s, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && s.FacingLeft {
					// attempt to use animation frame width if available
					if animComp, ok2 := ecs.Get(w, e, component.AnimationComponent.Kind()); ok2 {
						if def, ok3 := animComp.Defs[animComp.Current]; ok3 {
							imgW := float64(def.FrameW)
							offX = imgW*scaleX - baseOff - hb.Width
						} else {
							offX = -baseOff - hb.Width
						}
					} else {
						offX = -baseOff - hb.Width
					}
				}
				// compute world rect
				x := (t.X + offX - camX) * zoom
				y := (t.Y + hb.OffsetY*scaleX - camY) * zoom
				wRect := hb.Width * zoom
				hRect := hb.Height * zoom
				// outline
				ebitenutil.DrawLine(screen, x, y, x+wRect, y, color.NRGBA{R: 220, G: 40, B: 40, A: 200})
				ebitenutil.DrawLine(screen, x+wRect, y, x+wRect, y+hRect, color.NRGBA{R: 220, G: 40, B: 40, A: 200})
				ebitenutil.DrawLine(screen, x+wRect, y+hRect, x, y+hRect, color.NRGBA{R: 220, G: 40, B: 40, A: 200})
				ebitenutil.DrawLine(screen, x, y+hRect, x, y, color.NRGBA{R: 220, G: 40, B: 40, A: 200})
			}
		})

		// Hurtboxes: show outlines in blue
		ecs.ForEach2(w, component.HurtboxComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, hbSlice *[]component.Hurtbox, t *component.Transform) {
			for _, hb := range *hbSlice {
				scaleX := t.ScaleX
				if scaleX == 0 {
					scaleX = 1
				}
				baseOff := hb.OffsetX * scaleX
				offX := baseOff
				if s, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && s.FacingLeft {
					if animComp, ok2 := ecs.Get(w, e, component.AnimationComponent.Kind()); ok2 {
						if def, ok3 := animComp.Defs[animComp.Current]; ok3 {
							imgW := float64(def.FrameW)
							offX = imgW*scaleX - baseOff - hb.Width
						} else {
							offX = -baseOff - hb.Width
						}
					} else {
						offX = -baseOff - hb.Width
					}
				}
				x := (t.X + offX - camX) * zoom
				y := (t.Y + hb.OffsetY*scaleX - camY) * zoom
				wRect := hb.Width * zoom
				hRect := hb.Height * zoom
				ebitenutil.DrawLine(screen, x, y, x+wRect, y, color.NRGBA{R: 60, G: 140, B: 220, A: 180})
				ebitenutil.DrawLine(screen, x+wRect, y, x+wRect, y+hRect, color.NRGBA{R: 60, G: 140, B: 220, A: 180})
				ebitenutil.DrawLine(screen, x+wRect, y+hRect, x, y+hRect, color.NRGBA{R: 60, G: 140, B: 220, A: 180})
				ebitenutil.DrawLine(screen, x, y+hRect, x, y, color.NRGBA{R: 60, G: 140, B: 220, A: 180})
			}
		})
	}

}

func DrawPlayerStateDebug(w *ecs.World, screen *ebiten.Image) {
	if w == nil || screen == nil {
		return
	}
	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}
	stateComp, ok := ecs.Get(w, player, component.PlayerStateMachineComponent.Kind())
	if !ok {
		return
	}
	stateName := "none"
	if stateComp.State != nil {
		stateName = stateComp.State.Name()
	}
	grounded := false
	wall := 0
	if pc, ok := ecs.Get(w, player, component.PlayerCollisionComponent.Kind()); ok {
		grounded = pc.Grounded || pc.GroundGrace > 0
		wall = pc.Wall
	}
	text := fmt.Sprintf("Player State: %s\nGrounded: %v\nWall: %d\nWallGrabTimer: %d\nJumpsUsed: %d", stateName, grounded, wall, stateComp.WallGrabTimer, stateComp.JumpsUsed)
	ebitenutil.DebugPrintAt(screen, text, 10, 10)
}

// DrawAIStateDebug draws each AI-controlled entity's current FSM state above it.
func DrawAIStateDebug(w *ecs.World, screen *ebiten.Image) {
	if w == nil || screen == nil {
		return
	}

	camX, camY, zoom := debugCameraTransform(w)

	ecs.ForEach2(w, component.AITagComponent.Kind(), component.AIStateComponent.Kind(), func(e ecs.Entity, aiTag *component.AITag, stateComp *component.AIState) {
		x, y := 0.0, 0.0
		if pb, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind()); ok && pb.Body != nil {
			pos := pb.Body.Position()
			x = pos.X
			y = pos.Y - pb.Height/2.0 - 8 // above top
		} else if t, ok := ecs.Get(w, e, component.TransformComponent.Kind()); ok {
			x = t.X
			y = t.Y - 16
		}

		sx := int((x - camX) * zoom)
		sy := int((y - camY) * zoom)

		stateName := string(stateComp.Current)
		if stateName == "" {
			stateName = "none"
		}
		ebitenutil.DebugPrintAt(screen, stateName, sx, sy)
	})
}

// DrawPathfindingDebug draws pathfinding nodes for entities with PathfindingComponent.
func DrawPathfindingDebug(w *ecs.World, screen *ebiten.Image) {
	if w == nil || screen == nil {
		return
	}

	camX, camY, zoom := debugCameraTransform(w)

	ecs.ForEach(w, component.PathfindingComponent.Kind(), func(e ecs.Entity, pf *component.Pathfinding) {
		size := pf.DebugNodeSize
		if size <= 0 {
			size = 3
		}

		drawNode := func(node component.PathNode, c color.Color) {
			x := (node.X-camX)*zoom - size/2
			y := (node.Y-camY)*zoom - size/2
			ebitenutil.DrawRect(screen, x, y, size, size, c)
		}

		for _, n := range pf.Visited {
			drawNode(n, color.NRGBA{R: 80, G: 160, B: 255, A: 140})
		}
		for _, n := range pf.Path {
			drawNode(n, color.NRGBA{R: 255, G: 220, B: 40, A: 200})
		}
	})
}

type physicsDebugDrawer struct {
	screen *ebiten.Image
	camX   float64
	camY   float64
	zoom   float64
}

func (d *physicsDebugDrawer) DrawCircle(pos cp.Vector, angle, radius float64, outline, fill cp.FColor, data interface{}) {
	if radius <= 0 {
		return
	}
	points := make([]cp.Vector, 0, debugCircleSegments)
	for i := 0; i < debugCircleSegments; i++ {
		t := (2 * math.Pi) * (float64(i) / float64(debugCircleSegments))
		points = append(points, cp.Vector{X: pos.X + math.Cos(t)*radius, Y: pos.Y + math.Sin(t)*radius})
	}
	d.drawPolygon(points, outline)
	end := cp.Vector{X: pos.X + math.Cos(angle)*radius, Y: pos.Y + math.Sin(angle)*radius}
	d.drawLine(pos, end, outline)
}

func (d *physicsDebugDrawer) DrawSegment(a, b cp.Vector, fill cp.FColor, data interface{}) {
	d.drawLine(a, b, fill)
}

func (d *physicsDebugDrawer) DrawFatSegment(a, b cp.Vector, radius float64, outline, fill cp.FColor, data interface{}) {
	d.drawLine(a, b, outline)
	if radius > 0 {
		d.drawCircle(a, radius, outline)
		d.drawCircle(b, radius, outline)
	}
}

func (d *physicsDebugDrawer) DrawPolygon(count int, verts []cp.Vector, radius float64, outline, fill cp.FColor, data interface{}) {
	if count <= 0 {
		return
	}
	d.drawPolygon(verts[:count], outline)
}

func (d *physicsDebugDrawer) DrawDot(size float64, pos cp.Vector, fill cp.FColor, data interface{}) {
	if size <= 0 {
		size = debugDotSize
	}
	half := size / 2
	left := cp.Vector{X: pos.X - half, Y: pos.Y}
	right := cp.Vector{X: pos.X + half, Y: pos.Y}
	up := cp.Vector{X: pos.X, Y: pos.Y - half}
	down := cp.Vector{X: pos.X, Y: pos.Y + half}
	d.drawLine(left, right, fill)
	d.drawLine(up, down, fill)
}

func (d *physicsDebugDrawer) Flags() uint {
	return cp.DRAW_SHAPES
}

func (d *physicsDebugDrawer) OutlineColor() cp.FColor {
	return cp.FColor{R: 0.2, G: 1, B: 0.2, A: 0.9}
}

func (d *physicsDebugDrawer) ShapeColor(shape *cp.Shape, data interface{}) cp.FColor {
	return cp.FColor{R: 0.1, G: 0.6, B: 0.1, A: 0.5}
}

func (d *physicsDebugDrawer) ConstraintColor() cp.FColor {
	return cp.FColor{R: 1, G: 0.5, B: 0.1, A: 0.9}
}

func (d *physicsDebugDrawer) CollisionPointColor() cp.FColor {
	return cp.FColor{R: 1, G: 0.2, B: 0.2, A: 0.9}
}

func (d *physicsDebugDrawer) Data() interface{} {
	return nil
}

func (d *physicsDebugDrawer) drawLine(a, b cp.Vector, color cp.FColor) {
	x1, y1 := d.toScreen(a)
	x2, y2 := d.toScreen(b)
	ebitenutil.DrawLine(d.screen, x1, y1, x2, y2, toNRGBA(color))
}

func (d *physicsDebugDrawer) drawPolygon(verts []cp.Vector, color cp.FColor) {
	if len(verts) == 0 {
		return
	}
	for i := 0; i < len(verts); i++ {
		a := verts[i]
		b := verts[(i+1)%len(verts)]
		d.drawLine(a, b, color)
	}
}

func (d *physicsDebugDrawer) drawCircle(center cp.Vector, radius float64, color cp.FColor) {
	if radius <= 0 {
		return
	}
	points := make([]cp.Vector, 0, debugCircleSegments)
	for i := 0; i < debugCircleSegments; i++ {
		t := (2 * math.Pi) * (float64(i) / float64(debugCircleSegments))
		points = append(points, cp.Vector{X: center.X + math.Cos(t)*radius, Y: center.Y + math.Sin(t)*radius})
	}
	d.drawPolygon(points, color)
}

func (d *physicsDebugDrawer) toScreen(v cp.Vector) (float64, float64) {
	return (v.X - d.camX) * d.zoom, (v.Y - d.camY) * d.zoom
}

func toNRGBA(c cp.FColor) color.NRGBA {
	return color.NRGBA{
		R: uint8(clamp01(c.R) * 255),
		G: uint8(clamp01(c.G) * 255),
		B: uint8(clamp01(c.B) * 255),
		A: uint8(clamp01(c.A) * 255),
	}
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func debugCameraTransform(w *ecs.World) (float64, float64, float64) {
	camX, camY := 0.0, 0.0
	zoom := 1.0
	camEntity, ok := ecs.First(w, component.CameraComponent.Kind())
	if !ok {
		return camX, camY, zoom
	}
	if camTransform, ok := ecs.Get(w, camEntity, component.TransformComponent.Kind()); ok {
		camX = camTransform.X
		camY = camTransform.Y
	}
	if camComp, ok := ecs.Get(w, camEntity, component.CameraComponent.Kind()); ok {
		if camComp.Zoom > 0 {
			zoom = camComp.Zoom
		}
	}
	return camX, camY, zoom
}
