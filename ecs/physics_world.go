package ecs

import (
	"log"
	"math"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs/components"
	"github.com/milk9111/sidescroller/obj"
)

const (
	collisionTypeSolid cp.CollisionType = iota + 1
	collisionTypeHazard
	collisionTypeDynamic
	collisionTypeGroundSensor
	collisionTypePlayer
	collisionTypeEnemy
)

// PhysicsWorld owns the Chipmunk space and static collision shapes.
type PhysicsWorld struct {
	level         *obj.Level
	space         *cp.Space
	handlersReady bool

	shapeToEntity  map[*cp.Shape]int
	groundToEntity map[*cp.Shape]int
	entityStates   map[int]*components.CollisionState
}

// NewPhysicsWorld creates a physics world for a level.
func NewPhysicsWorld(level *obj.Level) *PhysicsWorld {
	space := cp.NewSpace()
	space.Iterations = 20
	space.SetGravity(cp.Vector{X: 0, Y: common.Gravity})

	pw := &PhysicsWorld{
		level:          level,
		space:          space,
		shapeToEntity:  make(map[*cp.Shape]int),
		groundToEntity: make(map[*cp.Shape]int),
		entityStates:   make(map[int]*components.CollisionState),
	}
	pw.buildStaticShapes()
	pw.setupHandlers()
	return pw
}

// Space returns the underlying Chipmunk space.
func (pw *PhysicsWorld) Space() *cp.Space {
	if pw == nil {
		return nil
	}
	return pw.space
}

// SetEntityState registers a collision state for an entity.
func (pw *PhysicsWorld) SetEntityState(id int, state *components.CollisionState) {
	if pw == nil || id <= 0 {
		return
	}
	if pw.entityStates == nil {
		pw.entityStates = make(map[int]*components.CollisionState)
	}
	if state == nil {
		delete(pw.entityStates, id)
		return
	}
	pw.entityStates[id] = state
}

// EnsureBody creates a dynamic body for an entity if needed.
func (pw *PhysicsWorld) EnsureBody(id int, t *components.Transform, c *components.Collider, g *components.GroundSensor, grav *components.Gravity, body *components.PhysicsBody) *components.PhysicsBody {
	if pw == nil || pw.space == nil || id <= 0 || t == nil || c == nil {
		return body
	}
	if body != nil && body.Body != nil {
		return body
	}

	mass := 1.0
	moment := cp.MomentForBox(mass, float64(c.Width), float64(c.Height))
	if c != nil && c.FixedRotation {
		moment = math.Inf(1)
	}
	cpBody := cp.NewBody(mass, moment)
	cpBody.SetAngle(0)
	cpBody.SetAngularVelocity(0)
	posX := float64(t.X + c.Width/2 + c.OffsetX)
	posY := float64(t.Y + c.Height/2 + c.OffsetY)
	cpBody.SetPosition(cp.Vector{X: posX, Y: posY})
	shape := cp.NewBox(cpBody, float64(c.Width), float64(c.Height), 0)
	shape.SetFriction(0.8)
	// set collision type based on collider role
	if c != nil && c.IsEnemy {
		shape.SetCollisionType(collisionTypeEnemy)
		log.Printf("PhysicsWorld: EnsureBody set entity %d collisionType=Enemy", id)
	} else if c != nil && c.IsPlayer {
		shape.SetCollisionType(collisionTypePlayer)
		log.Printf("PhysicsWorld: EnsureBody set entity %d collisionType=Player", id)
	} else {
		shape.SetCollisionType(collisionTypeDynamic)
		log.Printf("PhysicsWorld: EnsureBody set entity %d collisionType=Dynamic", id)
	}
	shape.SetSensor(c.Sensor)

	if grav != nil && !grav.Enabled {
		cpBody.SetVelocityUpdateFunc(func(body *cp.Body, gravity cp.Vector, damping float64, dt float64) {
			cp.BodyUpdateVelocity(body, cp.Vector{}, damping, dt)
		})
	}

	pw.space.AddBody(cpBody)
	pw.space.AddShape(shape)

	if pw.shapeToEntity != nil {
		pw.shapeToEntity[shape] = id
	}

	var groundShape *cp.Shape
	if g != nil {
		gw := g.Width
		gh := g.Height
		if gw <= 0 {
			gw = c.Width * 0.9
		}
		if gh <= 0 {
			gh = 2
		}
		gx := g.OffsetX
		gy := g.OffsetY
		if gx == 0 && gy == 0 {
			gy = c.Height / 2.0
		}
		bb := cp.BB{
			L: float64(gx) - float64(gw)/2.0,
			B: float64(gy),
			R: float64(gx) + float64(gw)/2.0,
			T: float64(gy) + float64(gh),
		}
		groundShape = cp.NewBox2(cpBody, bb, 0)
		groundShape.SetSensor(true)
		groundShape.SetCollisionType(collisionTypeGroundSensor)
		pw.space.AddShape(groundShape)
		if pw.groundToEntity != nil {
			pw.groundToEntity[groundShape] = id
		}
	}

	return &components.PhysicsBody{Body: cpBody, Shape: shape, GroundShape: groundShape}
}

// Step advances the physics simulation.
func (pw *PhysicsWorld) Step(dt float64) {
	if pw == nil || pw.space == nil {
		return
	}
	pw.space.Step(dt)
}

func (pw *PhysicsWorld) buildStaticShapes() {
	if pw == nil || pw.space == nil || pw.level == nil {
		return
	}

	if pw.level.PhysicsLayers != nil && len(pw.level.PhysicsLayers) > 0 {
		for _, ly := range pw.level.PhysicsLayers {
			if ly == nil || ly.Tiles == nil || len(ly.Tiles) != pw.level.Width*pw.level.Height {
				continue
			}
			pw.processLayerTiles(ly.Tiles)
		}
	} else {
		if pw.level.Layers == nil || len(pw.level.Layers) == 0 {
			return
		}
		for layerIdx, layer := range pw.level.Layers {
			if layer == nil || len(layer) != pw.level.Width*pw.level.Height {
				continue
			}
			if pw.level.LayerMeta == nil || layerIdx >= len(pw.level.LayerMeta) || !pw.level.LayerMeta[layerIdx].HasPhysics {
				continue
			}
			pw.processLayerTiles(layer)
		}
	}

	worldW := float64(pw.level.Width * common.TileSize)
	worldH := float64(pw.level.Height * common.TileSize)
	if worldW > 0 && worldH > 0 {
		thickness := 1.0
		segments := []struct {
			a cp.Vector
			b cp.Vector
		}{
			{a: cp.Vector{X: 0, Y: 0}, b: cp.Vector{X: worldW, Y: 0}},
			{a: cp.Vector{X: 0, Y: worldH}, b: cp.Vector{X: worldW, Y: worldH}},
			{a: cp.Vector{X: 0, Y: 0}, b: cp.Vector{X: 0, Y: worldH}},
			{a: cp.Vector{X: worldW, Y: 0}, b: cp.Vector{X: worldW, Y: worldH}},
		}
		for _, seg := range segments {
			shape := cp.NewSegment(pw.space.StaticBody, seg.a, seg.b, thickness)
			shape.SetFriction(0.8)
			shape.SetCollisionType(collisionTypeSolid)
			pw.space.AddShape(shape)
		}
	}
}

func (pw *PhysicsWorld) processLayerTiles(layer []int) {
	if pw == nil || layer == nil || pw.level == nil {
		return
	}
	processed := make([]bool, pw.level.Width*pw.level.Height)
	for y := 0; y < pw.level.Height; y++ {
		for x := 0; x < pw.level.Width; x++ {
			idx := y*pw.level.Width + x
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

			if tileVal == 2 {
				size := float64(common.TileSize)
				verts := []cp.Vector{
					{X: x0, Y: y0 + size},
					{X: x0 + size, Y: y0 + size},
					{X: x0 + size/2.0, Y: y0},
				}
				shape := cp.NewPolyShapeRaw(pw.space.StaticBody, 3, verts, 0)
				shape.SetSensor(true)
				shape.SetCollisionType(collisionTypeHazard)
				pw.space.AddShape(shape)
				processed[idx] = true
				continue
			}

			w := 1
			for x+w < pw.level.Width {
				idx2 := y*pw.level.Width + (x + w)
				v := layer[idx2]
				if processed[idx2] || v == 0 || v == 2 {
					break
				}
				w++
			}

			h := 1
		heightLoop:
			for y+h < pw.level.Height {
				for xi := x; xi < x+w; xi++ {
					idx2 := (y+h)*pw.level.Width + xi
					v := layer[idx2]
					if processed[idx2] || v == 0 || v == 2 {
						break heightLoop
					}
				}
				h++
			}

			widthF := float64(w * common.TileSize)
			heightF := float64(h * common.TileSize)
			bb := cp.BB{L: x0, B: y0, R: x0 + widthF, T: y0 + heightF}
			shape := cp.NewBox2(pw.space.StaticBody, bb, 0)
			shape.SetFriction(0.8)
			shape.SetCollisionType(collisionTypeSolid)
			pw.space.AddShape(shape)

			for yy := y; yy < y+h; yy++ {
				for xx := x; xx < x+w; xx++ {
					processed[yy*pw.level.Width+xx] = true
				}
			}
		}
	}
}

func (pw *PhysicsWorld) setupHandlers() {
	if pw == nil || pw.handlersReady || pw.space == nil {
		return
	}

	wallHandler := pw.space.NewCollisionHandler(collisionTypeDynamic, collisionTypeSolid)
	wallHandler.UserData = pw
	wallHandler.PreSolveFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		world, ok := userData.(*PhysicsWorld)
		if !ok || world == nil {
			return true
		}
		shapeA, shapeB := arb.Shapes()
		idA, okA := world.shapeToEntity[shapeA]
		idB, okB := world.shapeToEntity[shapeB]
		var id int
		playerIsA := okA
		if okA {
			id = idA
		} else if okB {
			id = idB
			playerIsA = false
		} else {
			return true
		}
		state := world.entityStates[id]
		if state == nil {
			return true
		}
		n := arb.Normal()
		if !playerIsA {
			n = n.Neg()
		}
		if n.X < -0.5 {
			state.Wall = components.WallLeft
		} else if n.X > 0.5 {
			state.Wall = components.WallRight
		}
		return true
	}

	groundHandler := pw.space.NewCollisionHandler(collisionTypeGroundSensor, collisionTypeSolid)
	groundHandler.UserData = pw
	groundHandler.PreSolveFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		world, ok := userData.(*PhysicsWorld)
		if !ok || world == nil {
			return true
		}
		shapeA, shapeB := arb.Shapes()
		idA, okA := world.groundToEntity[shapeA]
		idB, okB := world.groundToEntity[shapeB]
		var id int
		if okA {
			id = idA
		} else if okB {
			id = idB
		} else {
			return true
		}
		state := world.entityStates[id]
		if state == nil {
			return true
		}
		state.Grounded = true
		state.GroundGrace = 6
		return true
	}

	hazardHandler := pw.space.NewCollisionHandler(collisionTypeDynamic, collisionTypeHazard)
	hazardHandler.UserData = pw
	hazardHandler.BeginFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		world, ok := userData.(*PhysicsWorld)
		if !ok || world == nil {
			return true
		}

		// prevent physics collisions between player and enemy bodies
		playerEnemyHandler := pw.space.NewCollisionHandler(collisionTypePlayer, collisionTypeEnemy)
		playerEnemyHandler.UserData = pw
		playerEnemyHandler.PreSolveFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
			log.Println("PhysicsWorld: PreSolve player-enemy - ignoring collision")
			// ignore player-enemy collisions at solve time
			return false
		}
		shapeA, shapeB := arb.Shapes()
		idA, okA := world.shapeToEntity[shapeA]
		idB, okB := world.shapeToEntity[shapeB]
		var id int
		if okA {
			id = idA
		} else if okB {
			id = idB
		}
		if id > 0 {
			if state := world.entityStates[id]; state != nil {
				state.HitHazard = true
			}
		}
		return true
	}

	pw.handlersReady = true
}
