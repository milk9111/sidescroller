package obj

import (
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/common"
)

const (
	collisionTypePlayer cp.CollisionType = iota + 1
	collisionTypePlayerGround
	collisionTypeSolid
	collisionTypeHazard
)

type CollisionWorld struct {
	level *Level
	space *cp.Space

	playerBody  *cp.Body
	playerShape *cp.Shape
	groundShape *cp.Shape

	grounded    bool
	wall        wallSide
	hitTriangle bool
	groundGrace int

	handlersReady bool
}

func NewCollisionWorld(level *Level) *CollisionWorld {
	space := cp.NewSpace()
	space.Iterations = 20
	space.SetGravity(cp.Vector{X: 0, Y: common.Gravity})
	cw := &CollisionWorld{level: level, space: space}
	cw.buildStaticShapes()
	return cw
}

func (cw *CollisionWorld) buildStaticShapes() {
	if cw == nil || cw.space == nil || cw.level == nil {
		return
	}
	if cw.level.Layers == nil || len(cw.level.Layers) == 0 {
		return
	}

	for layerIdx, layer := range cw.level.Layers {
		if layer == nil || len(layer) != cw.level.Width*cw.level.Height {
			continue
		}
		if cw.level.LayerMeta == nil || layerIdx >= len(cw.level.LayerMeta) || !cw.level.LayerMeta[layerIdx].HasPhysics {
			continue
		}
		// Merge contiguous solid tiles into larger rectangles so the physics
		// world uses fewer continuous static boxes instead of one box per tile.
		processed := make([]bool, cw.level.Width*cw.level.Height)
		for y := 0; y < cw.level.Height; y++ {
			for x := 0; x < cw.level.Width; x++ {
				idx := y*cw.level.Width + x
				if processed[idx] {
					continue
				}
				tileVal := layer[idx]
				if tileVal == 0 {
					processed[idx] = true
					continue
				}

				x0 := float64(x * common.TileSize)
				y0 := float64(y * common.TileSize)

				// Hazard triangles remain individual sensor shapes.
				if tileVal == 2 {
					size := float64(common.TileSize)
					verts := []cp.Vector{
						{X: x0, Y: y0 + size},
						{X: x0 + size, Y: y0 + size},
						{X: x0 + size/2.0, Y: y0},
					}
					shape := cp.NewPolyShapeRaw(cw.space.StaticBody, 3, verts, 0)
					shape.SetSensor(true)
					shape.SetCollisionType(collisionTypeHazard)
					cw.space.AddShape(shape)
					processed[idx] = true
					continue
				}

				// For solid tiles, greedily expand a rectangle to cover as many
				// contiguous solid tiles as possible (width then height).
				w := 1
				for x+w < cw.level.Width {
					idx2 := y*cw.level.Width + (x + w)
					v := layer[idx2]
					if processed[idx2] || v == 0 || v == 2 {
						break
					}
					w++
				}

				h := 1
			heightLoop:
				for y+h < cw.level.Height {
					for xi := x; xi < x+w; xi++ {
						idx2 := (y+h)*cw.level.Width + xi
						v := layer[idx2]
						if processed[idx2] || v == 0 || v == 2 {
							break heightLoop
						}
					}
					h++
				}

				// Create a single box covering the rectangle [x..x+w) x [y..y+h).
				widthF := float64(w * common.TileSize)
				heightF := float64(h * common.TileSize)
				bb := cp.BB{L: x0, B: y0, R: x0 + widthF, T: y0 + heightF}
				shape := cp.NewBox2(cw.space.StaticBody, bb, 0)
				shape.SetFriction(0.8)
				shape.SetCollisionType(collisionTypeSolid)
				cw.space.AddShape(shape)

				// Mark all tiles in the rectangle processed.
				for yy := y; yy < y+h; yy++ {
					for xx := x; xx < x+w; xx++ {
						processed[yy*cw.level.Width+xx] = true
					}
				}
			}
		}
	}

	// add world bounds matching the level size (pixels)
	worldW := float64(cw.level.Width * common.TileSize)
	worldH := float64(cw.level.Height * common.TileSize)
	if worldW > 0 && worldH > 0 {
		thickness := 1.0
		segments := []struct {
			a cp.Vector
			b cp.Vector
		}{
			{a: cp.Vector{X: 0, Y: 0}, b: cp.Vector{X: worldW, Y: 0}},           // top
			{a: cp.Vector{X: 0, Y: worldH}, b: cp.Vector{X: worldW, Y: worldH}}, // bottom
			{a: cp.Vector{X: 0, Y: 0}, b: cp.Vector{X: 0, Y: worldH}},           // left
			{a: cp.Vector{X: worldW, Y: 0}, b: cp.Vector{X: worldW, Y: worldH}}, // right
		}
		for _, seg := range segments {
			shape := cp.NewSegment(cw.space.StaticBody, seg.a, seg.b, thickness)
			shape.SetFriction(0.8)
			shape.SetCollisionType(collisionTypeSolid)
			cw.space.AddShape(shape)
		}
	}
}

func (cw *CollisionWorld) AttachPlayer(p *Player) {
	if cw == nil || cw.space == nil || p == nil {
		return
	}
	if cw.playerBody != nil {
		return
	}

	mass := 1.0
	moment := cp.MomentForBox(mass, float64(p.Width), float64(p.Height))
	body := cp.NewBody(mass, moment)
	body.SetAngle(0)
	body.SetAngularVelocity(0)
	body.SetPosition(cp.Vector{X: float64(p.X + p.Width/2), Y: float64(p.Y + p.Height/2)})
	shape := cp.NewBox(body, float64(p.Width), float64(p.Height), 0)
	shape.SetFriction(0.0)
	shape.SetCollisionType(collisionTypePlayer)
	groundBB := cp.BB{
		L: -float64(p.Width) * 0.45,
		B: float64(p.Height) / 2.0,
		R: float64(p.Width) * 0.45,
		T: float64(p.Height)/2.0 + 2,
	}
	groundShape := cp.NewBox2(body, groundBB, 0)
	groundShape.SetSensor(true)
	groundShape.SetCollisionType(collisionTypePlayerGround)

	cw.space.AddBody(body)
	cw.space.AddShape(shape)
	cw.space.AddShape(groundShape)

	cw.playerBody = body
	cw.playerShape = shape
	cw.groundShape = groundShape
	if p != nil {
		p.body = body
		p.shape = shape
	}

	cw.setupHandlers()
}

func (cw *CollisionWorld) setupHandlers() {
	if cw.handlersReady || cw.space == nil {
		return
	}
	handler := cw.space.NewCollisionHandler(collisionTypePlayer, collisionTypeSolid)
	handler.UserData = cw
	handler.PreSolveFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		world, ok := userData.(*CollisionWorld)
		if !ok || world == nil {
			return true
		}
		shapeA, shapeB := arb.Shapes()
		if world.playerShape == nil || (shapeA != world.playerShape && shapeB != world.playerShape) {
			return true
		}
		playerIsA := shapeA == world.playerShape
		n := arb.Normal()
		if !playerIsA {
			n = n.Neg()
		}
		if n.X < -0.5 {
			world.wall = WALL_LEFT
		} else if n.X > 0.5 {
			world.wall = WALL_RIGHT
		}
		return true
	}

	groundHandler := cw.space.NewCollisionHandler(collisionTypePlayerGround, collisionTypeSolid)
	groundHandler.UserData = cw
	groundHandler.PreSolveFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		world, ok := userData.(*CollisionWorld)
		if !ok || world == nil {
			return true
		}
		world.grounded = true
		world.groundGrace = 6
		return true
	}

	hazardHandler := cw.space.NewCollisionHandler(collisionTypePlayer, collisionTypeHazard)
	hazardHandler.UserData = cw
	hazardHandler.BeginFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		world, ok := userData.(*CollisionWorld)
		if ok && world != nil {
			world.hitTriangle = true
		}
		return true
	}

	cw.handlersReady = true
}

func (cw *CollisionWorld) BeginStep() {
	if cw == nil {
		return
	}
	if cw.groundGrace > 0 {
		cw.groundGrace--
	}
	cw.grounded = false
	cw.wall = WALL_NONE
	cw.hitTriangle = false
}

func (cw *CollisionWorld) Step(dt float64) {
	if cw == nil || cw.space == nil {
		return
	}
	cw.space.Step(dt)
}

func (cw *CollisionWorld) HitTriangle() bool {
	if cw == nil {
		return false
	}
	return cw.hitTriangle
}

// IsGrounded returns true when the player is grounded according to chipmunk contacts.
func (cw *CollisionWorld) IsGrounded(_ common.Rect) bool {
	if cw == nil {
		return false
	}
	return cw.grounded || cw.groundGrace > 0
}

// IsTouchingWall returns which side is touching a wall according to chipmunk contacts.
func (cw *CollisionWorld) IsTouchingWall(_ common.Rect) wallSide {
	if cw == nil {
		return WALL_NONE
	}
	return cw.wall
}
