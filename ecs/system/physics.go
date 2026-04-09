package system

import (
	"math"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const TerminalVelocity = 15.0

const (
	collisionTypePlayer cp.CollisionType = iota + 1
	collisionTypePlayerGround
	collisionTypeSolid
	collisionTypeAI
)

const (
	wallNone  = 0
	wallLeft  = 1
	wallRight = 2
)

const groundGraceFrames = 6

const (
	clamberWallTolerance   = 2.5
	clamberStandClearance  = 0.1
	clamberCandidateMargin = 0.1
)

type PhysicsSystem struct {
	space         *cp.Space
	handlersReady bool

	entities     map[ecs.Entity]*bodyInfo
	playerShapes map[*cp.Shape]ecs.Entity
	groundShapes map[*cp.Shape]ecs.Entity
	aiShapes     map[*cp.Shape]ecs.Entity
	// shapeEntity maps any shape to its owning entity when available.
	shapeEntity  map[*cp.Shape]ecs.Entity
	playerAIColl map[ecs.Entity]bool
	playerStates map[ecs.Entity]*playerContactState
	// world is set at Update time so collision handlers can query components.
	world *ecs.World
}

type bodyInfo struct {
	body        *cp.Body
	mainShape   *cp.Shape
	groundShape *cp.Shape
	shapes      []*cp.Shape
	static      bool
}

type playerContactState struct {
	grounded    bool
	groundGrace int
	wall        int
}

func NewPhysicsSystem() *PhysicsSystem {
	space := cp.NewSpace()
	space.Iterations = 20
	space.SetGravity(cp.Vector{X: 0, Y: common.Gravity})
	return &PhysicsSystem{
		space:        space,
		entities:     make(map[ecs.Entity]*bodyInfo),
		playerShapes: make(map[*cp.Shape]ecs.Entity),
		groundShapes: make(map[*cp.Shape]ecs.Entity),
		aiShapes:     make(map[*cp.Shape]ecs.Entity),
		shapeEntity:  make(map[*cp.Shape]ecs.Entity),
		playerAIColl: make(map[ecs.Entity]bool),
		playerStates: make(map[ecs.Entity]*playerContactState),
	}
}

// Reset clears internal physics state and creates a fresh space. Call this when
// reloading the world to avoid leftover bodies/shapes from the previous world.
func (ps *PhysicsSystem) Reset() {
	if ps == nil {
		return
	}
	ps.space = cp.NewSpace()
	ps.space.Iterations = 20
	ps.space.SetGravity(cp.Vector{X: 0, Y: common.Gravity})
	ps.handlersReady = false
	ps.entities = make(map[ecs.Entity]*bodyInfo)
	ps.playerShapes = make(map[*cp.Shape]ecs.Entity)
	ps.groundShapes = make(map[*cp.Shape]ecs.Entity)
	ps.aiShapes = make(map[*cp.Shape]ecs.Entity)
	ps.shapeEntity = make(map[*cp.Shape]ecs.Entity)
	ps.playerAIColl = make(map[ecs.Entity]bool)
	ps.playerStates = make(map[ecs.Entity]*playerContactState)
}

func (ps *PhysicsSystem) Space() *cp.Space {
	if ps == nil {
		return nil
	}
	return ps.space
}

func (ps *PhysicsSystem) Update(w *ecs.World) {
	if ps == nil || w == nil {
		return
	}

	// store current world for use inside collision handlers
	ps.world = w

	// Process any anchors marked for pending destroy: remove their physics
	// constraints from the space, then destroy the entity. This avoids
	// injecting other systems into physics; systems can mark anchors for
	// removal via the AnchorPendingDestroy component.
	// When detaching anchors we zero the player's angular velocity first so
	// that any rotational momentum doesn't immediately convert to a large
	// translational impulse (observed as the player being launched).
	playerEnt, _ := ecs.First(w, component.PlayerTagComponent.Kind())
	var playerBody *cp.Body
	if playerEnt != 0 {
		if bodyComp, ok := ecs.Get(w, playerEnt, component.PhysicsBodyComponent.Kind()); ok && bodyComp.Body != nil {
			playerBody = bodyComp.Body
		}
	}

	ecs.ForEach(w, component.AnchorPendingDestroyComponent.Kind(), func(e ecs.Entity, anchorPendingDestroy *component.AnchorPendingDestroy) {
		if !ecs.IsAlive(w, e) {
			return
		}

		// clear rotational momentum on the player before removing constraints
		if playerBody != nil {
			playerBody.SetAngularVelocity(0)
		}

		if jc, ok := ecs.Get(w, e, component.AnchorJointComponent.Kind()); ok {
			if jc.Slide != nil && ps.space != nil {
				ps.space.RemoveConstraint(jc.Slide)
			}
			if jc.Pivot != nil && ps.space != nil {
				ps.space.RemoveConstraint(jc.Pivot)
			}
			if jc.Pin != nil && ps.space != nil {
				ps.space.RemoveConstraint(jc.Pin)
			}
		}

		ecs.DestroyEntity(w, e)
	})

	ecs.ForEach(w, component.AnchorDetachRequestComponent.Kind(), func(e ecs.Entity, _ *component.AnchorDetachRequest) {
		if !ecs.IsAlive(w, e) {
			return
		}

		targetBody := ps.anchorTargetBody(w, e)
		if targetBody != nil {
			targetBody.SetAngularVelocity(0)
		}

		ps.removeAnchorJoints(w, e)
		_ = ecs.Remove(w, e, component.AnchorJointComponent.Kind())
		_ = ecs.Remove(w, e, component.AnchorConstraintRequestComponent.Kind())
		_ = ecs.Remove(w, e, component.AnchorDetachRequestComponent.Kind())
	})

	if ps.space == nil {
		ps.space = cp.NewSpace()
		ps.space.Iterations = 20
		ps.space.SetGravity(cp.Vector{X: 0, Y: common.Gravity})
		ps.handlersReady = false
	}

	ps.ensureHandlers()
	ps.syncEntities(w)
	ps.syncWorldBounds(w)
	ps.resetPlayerContacts(w)
	ps.processAnchorConstraints(w)
	ps.applyGravityScale(w)
	ps.applyTerminalVelocity(w)

	ps.space.Step(1.0)

	ps.syncTransforms(w)
	ps.flushPlayerContacts(w)
}

func (ps *PhysicsSystem) applyTerminalVelocity(w *ecs.World) {
	ecs.ForEach(w, component.PhysicsBodyComponent.Kind(), func(e ecs.Entity, bodyComp *component.PhysicsBody) {
		if bodyComp == nil || bodyComp.Static || bodyComp.Body == nil {
			return
		}

		v := bodyComp.Body.Velocity()
		if v.Y > TerminalVelocity {
			v.Y = TerminalVelocity
			bodyComp.Body.SetVelocityVector(v)
		}
	})
}

func (ps *PhysicsSystem) applyGravityScale(w *ecs.World) {
	if ps == nil || w == nil {
		return
	}

	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.GravityScaleComponent.Kind(), func(_ ecs.Entity, bodyComp *component.PhysicsBody, grav *component.GravityScale) {
		if bodyComp == nil || bodyComp.Static || bodyComp.Body == nil || grav == nil {
			return
		}

		v := bodyComp.Body.Velocity()
		v.Y += common.Gravity * (grav.Scale - 1)
		bodyComp.Body.SetVelocityVector(v)
	})
}

func (ps *PhysicsSystem) processAnchorConstraints(w *ecs.World) {
	if ps == nil || ps.space == nil || w == nil {
		return
	}

	ecs.ForEach(w, component.AnchorConstraintRequestComponent.Kind(), func(e ecs.Entity, req *component.AnchorConstraintRequest) {
		if req.Applied {
			return
		}

		targetEnt := e
		if req.TargetEntity != 0 {
			targetEnt = ecs.Entity(req.TargetEntity)
		}
		bodyComp, ok := ecs.Get(w, targetEnt, component.PhysicsBodyComponent.Kind())
		if !ok || bodyComp == nil || bodyComp.Body == nil {
			return
		}

		jointComp, ok := ecs.Get(w, e, component.AnchorJointComponent.Kind())
		if !ok {
			jointComp = &component.AnchorJoint{}
			if err := ecs.Add(w, e, component.AnchorJointComponent.Kind(), jointComp); err != nil {
				panic("physics system: add anchor joint: " + err.Error())
			}
		}

		switch req.Mode {
		case component.AnchorConstraintSlide:
			if jointComp.Pivot != nil {
				ps.space.RemoveConstraint(jointComp.Pivot)
				jointComp.Pivot = nil
			}
			if jointComp.Pin != nil {
				ps.space.RemoveConstraint(jointComp.Pin)
				jointComp.Pin = nil
			}
			// Recreate slide joint to ensure min/max length updates are applied.
			// This avoids leaving an old slide with a fixed-length that behaves
			// like a pin when we intend to allow extension.
			minLen := req.MinLen
			maxLen := req.MaxLen
			if maxLen < 0 {
				maxLen = 100000.0
			}
			// Add a tiny slack when maxLen is effectively the current distance to
			// avoid numerical pin-like behavior that prevents small extensions.
			pPos := bodyComp.Body.Position()
			currDist := math.Hypot(pPos.X-req.AnchorX, pPos.Y-req.AnchorY)
			if math.Abs(maxLen-currDist) < 1e-6 {
				maxLen = currDist + 0.1
			}
			if jointComp.Slide != nil {
				if slideJoint, ok := jointComp.Slide.Class.(*cp.SlideJoint); ok {
					slideJoint.AnchorA = cp.Vector{}
					slideJoint.AnchorB = cp.Vector{X: req.AnchorX, Y: req.AnchorY}
					slideJoint.Min = minLen
					slideJoint.Max = maxLen

					break
				}

				ps.space.RemoveConstraint(jointComp.Slide)
				jointComp.Slide = nil
			}
			bodyLocal := cp.Vector{}
			slide := cp.NewSlideJoint(bodyComp.Body, ps.space.StaticBody, bodyLocal, cp.Vector{X: req.AnchorX, Y: req.AnchorY}, minLen, maxLen)
			ps.space.AddConstraint(slide)
			jointComp.Slide = slide
		case component.AnchorConstraintPivot:
			if jointComp.Slide != nil {
				ps.space.RemoveConstraint(jointComp.Slide)
				jointComp.Slide = nil
			}
			if jointComp.Pin != nil {
				ps.space.RemoveConstraint(jointComp.Pin)
				jointComp.Pin = nil
			}
			if jointComp.Pivot == nil {
				pivot := cp.NewPivotJoint(bodyComp.Body, ps.space.StaticBody, cp.Vector{X: req.AnchorX, Y: req.AnchorY})
				ps.space.AddConstraint(pivot)
				jointComp.Pivot = pivot
			}
		case component.AnchorConstraintPin:
			if jointComp.Slide != nil {
				ps.space.RemoveConstraint(jointComp.Slide)
				jointComp.Slide = nil
			}
			if jointComp.Pivot != nil {
				ps.space.RemoveConstraint(jointComp.Pivot)
				jointComp.Pivot = nil
			}
			if jointComp.Pin != nil {
				if pinJoint, ok := jointComp.Pin.Class.(*cp.PinJoint); ok {
					pinJoint.AnchorA = cp.Vector{}
					pinJoint.AnchorB = cp.Vector{X: req.AnchorX, Y: req.AnchorY}
					pinJoint.Dist = req.MaxLen

					break
				}

				ps.space.RemoveConstraint(jointComp.Pin)
				jointComp.Pin = nil
			}
			bodyLocal := cp.Vector{}
			pin := cp.NewPinJoint(bodyComp.Body, ps.space.StaticBody, bodyLocal, cp.Vector{X: req.AnchorX, Y: req.AnchorY})
			if pinJoint, ok := pin.Class.(*cp.PinJoint); ok {
				pinJoint.Dist = req.MaxLen
			}
			ps.space.AddConstraint(pin)
			jointComp.Pin = pin
		default:
			return
		}

		req.Applied = true
	})
}

func (ps *PhysicsSystem) anchorTargetBody(w *ecs.World, owner ecs.Entity) *cp.Body {
	if ps == nil || w == nil {
		return nil
	}

	targetEnt := owner
	if req, ok := ecs.Get(w, owner, component.AnchorConstraintRequestComponent.Kind()); ok && req != nil && req.TargetEntity != 0 {
		targetEnt = ecs.Entity(req.TargetEntity)
	}

	bodyComp, ok := ecs.Get(w, targetEnt, component.PhysicsBodyComponent.Kind())
	if !ok || bodyComp == nil || bodyComp.Body == nil {
		return nil
	}

	return bodyComp.Body
}

func (ps *PhysicsSystem) removeAnchorJoints(w *ecs.World, owner ecs.Entity) {
	if ps == nil || ps.space == nil || w == nil {
		return
	}

	jc, ok := ecs.Get(w, owner, component.AnchorJointComponent.Kind())
	if !ok || jc == nil {
		return
	}

	if jc.Slide != nil {
		ps.space.RemoveConstraint(jc.Slide)
		jc.Slide = nil
	}
	if jc.Pivot != nil {
		ps.space.RemoveConstraint(jc.Pivot)
		jc.Pivot = nil
	}
	if jc.Pin != nil {
		ps.space.RemoveConstraint(jc.Pin)
		jc.Pin = nil
	}
}

func (ps *PhysicsSystem) ensureHandlers() {
	if ps.handlersReady || ps.space == nil {
		return
	}

	wallHandler := ps.space.NewCollisionHandler(collisionTypePlayer, collisionTypeSolid)
	wallHandler.UserData = ps
	wallHandler.PreSolveFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		sys, ok := userData.(*PhysicsSystem)
		if !ok || sys == nil {
			return true
		}
		shapeA, shapeB := arb.Shapes()
		playerEntity, playerIsA := sys.playerShapes[shapeA]
		if !playerIsA {
			var okB bool
			playerEntity, okB = sys.playerShapes[shapeB]
			if !okB {
				return true
			}
		}

		st := sys.playerStates[playerEntity]
		if st == nil {
			st = &playerContactState{}
			sys.playerStates[playerEntity] = st
		}

		n := arb.Normal()
		if !playerIsA {
			n = n.Neg()
		}

		// find the other shape and ignore hazard/spike shapes for wall contact
		var otherShape *cp.Shape
		if playerIsA {
			otherShape = shapeB
		} else {
			otherShape = shapeA
		}

		if otherShape != nil && sys != nil && sys.world != nil {
			if otherEnt, ok := sys.shapeEntity[otherShape]; ok && otherEnt != 0 {
				if ecs.Has(sys.world, otherEnt, component.HazardComponent.Kind()) {
					return true
				}
			}
		}

		if n.X < -0.5 {
			st.wall = wallLeft
		} else if n.X > 0.5 {
			st.wall = wallRight
		}

		return true
	}

	groundHandler := ps.space.NewCollisionHandler(collisionTypePlayerGround, collisionTypeSolid)
	groundHandler.UserData = ps
	groundHandler.PreSolveFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		sys, ok := userData.(*PhysicsSystem)
		if !ok || sys == nil {
			return true
		}
		shapeA, shapeB := arb.Shapes()
		playerEntity, okA := sys.groundShapes[shapeA]
		if !okA {
			var okB bool
			playerEntity, okB = sys.groundShapes[shapeB]
			if !okB {
				return true
			}
		}

		n := arb.Normal()
		if !okA {
			n = n.Neg()
		}
		// Only count as grounded when the contact normal points upward from the ground
		// toward the player (positive Y in screen-down coordinates).
		if n.Y <= 0.5 {
			return true
		}

		// find the other shape and ignore hazard/spike shapes for grounding
		var otherShape *cp.Shape
		if okA {
			otherShape = shapeB
		} else {
			otherShape = shapeA
		}

		if otherShape != nil && sys != nil && sys.world != nil {
			if otherEnt, ok := sys.shapeEntity[otherShape]; ok && otherEnt != 0 {
				if ecs.Has(sys.world, otherEnt, component.HazardComponent.Kind()) {
					return true
				}
			}
		}

		st := sys.playerStates[playerEntity]
		if st == nil {
			st = &playerContactState{}
			sys.playerStates[playerEntity] = st
		}
		st.grounded = true
		st.groundGrace = groundGraceFrames
		return true
	}

	ps.handlersReady = true

	// Player vs AI: detect overlaps but do not solve (player should pass through AI)
	aiHandler := ps.space.NewCollisionHandler(collisionTypePlayer, collisionTypeAI)
	aiHandler.UserData = ps
	aiHandler.PreSolveFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		sys, ok := userData.(*PhysicsSystem)
		if !ok || sys == nil {
			return false
		}
		shapeA, shapeB := arb.Shapes()
		playerEntity, playerIsA := sys.playerShapes[shapeA]
		if !playerIsA {
			var okB bool
			playerEntity, okB = sys.playerShapes[shapeB]
			if !okB {
				return false
			}
		}
		var aiEntity ecs.Entity
		if e, ok := sys.aiShapes[shapeA]; ok {
			aiEntity = e
		} else if e, ok := sys.aiShapes[shapeB]; ok {
			aiEntity = e
		}
		if playerEntity == 0 {
			return false
		}
		if aiEntity != 0 {
			sys.playerAIColl[playerEntity] = true
		}
		// Return false to skip collision solving (allow player to pass through AI)
		return false
	}

	// AI vs AI: skip collision solving so enemies do not push each other.
	aiVsAIHandler := ps.space.NewCollisionHandler(collisionTypeAI, collisionTypeAI)
	aiVsAIHandler.PreSolveFunc = func(_ *cp.Arbiter, _ *cp.Space, _ interface{}) bool {
		return false
	}
}

func (ps *PhysicsSystem) syncEntities(w *ecs.World) {
	if ps.space == nil {
		return
	}

	ps.cleanupEntities(w)

	// Ensure collision filters are applied for entities that already have
	// physics bodies but may have had a CollisionLayer component added
	// after their shapes were created (e.g. spawned at runtime).
	ecs.ForEach2(w, component.CollisionLayerComponent.Kind(), component.PhysicsBodyComponent.Kind(), func(e ecs.Entity, cl *component.CollisionLayer, _ *component.PhysicsBody) {
		if cl == nil {
			return
		}
		info := ps.entities[e]
		if info == nil {
			return
		}
		cat := cl.Category
		mask := cl.Mask
		if cat == 0 {
			cat = 1
		}
		if mask == 0 {
			mask = ^uint32(0)
		}
		for _, s := range info.shapes {
			if s == nil {
				continue
			}
			s.SetFilter(cp.ShapeFilter{Categories: uint(cat), Mask: uint(mask)})
		}
	})

	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, bodyComp *component.PhysicsBody, transform *component.Transform) {
		if bodyComp == nil || bodyComp.Disabled {
			return
		}

		isPlayer := ecs.Has(w, e, component.PlayerTagComponent.Kind())
		isAnchor := ecs.Has(w, e, component.AnchorTagComponent.Kind())
		isAI := ecs.Has(w, e, component.AITagComponent.Kind())

		info := ps.entities[e]
		if info != nil && info.mainShape != nil {
			if !bodyComp.Static && bodyComp.Body != nil {
				ps.applyBodyRotationLock(bodyComp, isAI)
				desiredCenterX := bodyCenterX(w, e, transform, bodyComp)
				desiredCenterY := bodyCenterY(transform, bodyComp)
				currPos := bodyComp.Body.Position()
				if math.Abs(currPos.X-desiredCenterX) > 1e-6 || math.Abs(currPos.Y-desiredCenterY) > 1e-6 {
					bodyComp.Body.SetPosition(cp.Vector{X: desiredCenterX, Y: desiredCenterY})
				}
			}

			if isPlayer {
				if info.mainShape != nil {
					ps.playerShapes[info.mainShape] = e
				}
				if info.groundShape != nil {
					ps.groundShapes[info.groundShape] = e
				}
			}
			if ecs.Has(w, e, component.AITagComponent.Kind()) {
				ps.aiShapes[info.mainShape] = e
			}

			// map all shapes to their owning entity for handler lookup
			for _, s := range info.shapes {
				if s != nil {
					ps.shapeEntity[s] = e
					// apply collision layer filters if present
					if cl, ok := ecs.Get(w, e, component.CollisionLayerComponent.Kind()); ok && cl != nil {
						cat := cl.Category
						mask := cl.Mask
						if cat == 0 {
							cat = 1
						}
						if mask == 0 {
							mask = ^uint32(0)
						}
						s.SetFilter(cp.ShapeFilter{Categories: uint(cat), Mask: uint(mask)})
					}
				}
			}

			if bodyComp.Body == nil || bodyComp.Shape == nil {
				bodyComp.Body = info.body
				bodyComp.Shape = info.mainShape
				_ = ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), bodyComp)
			}
			return
		}

		info = ps.createBodyInfo(w, e, transform, bodyComp, isPlayer, isAnchor, isAI)
		if info == nil || info.mainShape == nil {
			return
		}

		ps.entities[e] = info
		if isPlayer {
			if info.mainShape != nil {
				ps.playerShapes[info.mainShape] = e
			}
			if info.groundShape != nil {
				ps.groundShapes[info.groundShape] = e
			}
		}
		if isAI {
			if info.mainShape != nil {
				ps.aiShapes[info.mainShape] = e
			}
		}

		// map all shapes to their owning entity for handler lookup and
		// apply collision layer filters if present on the entity.
		for _, s := range info.shapes {
			if s != nil {
				ps.shapeEntity[s] = e
				if cl, ok := ecs.Get(w, e, component.CollisionLayerComponent.Kind()); ok && cl != nil {
					cat := cl.Category
					mask := cl.Mask
					if cat == 0 {
						cat = 1
					}
					if mask == 0 {
						mask = ^uint32(0)
					}
					s.SetFilter(cp.ShapeFilter{Categories: uint(cat), Mask: uint(mask)})
				}
			}
		}
		bodyComp.Body = info.body
		bodyComp.Shape = info.mainShape
		// _ = ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), bodyComp)
	})
}

func (ps *PhysicsSystem) applyBodyRotationLock(bodyComp *component.PhysicsBody, isAI bool) {
	if bodyComp == nil || bodyComp.Body == nil || bodyComp.Static {
		return
	}

	bodyComp.Body.SetMoment(physicsBodyMoment(bodyComp, isAI))
	if bodyComp.LockRotation || isAI {
		bodyComp.Body.SetAngularVelocity(0)
	}
}

func physicsBodyMoment(bodyComp *component.PhysicsBody, isAI bool) float64 {
	if bodyComp == nil {
		return 0
	}
	if bodyComp.LockRotation || isAI {
		return math.Inf(1)
	}

	mass := bodyComp.Mass
	if mass <= 0 {
		mass = 1
	}
	if bodyComp.Radius > 0 {
		return cp.MomentForCircle(mass, 0, bodyComp.Radius, cp.Vector{})
	}
	return cp.MomentForBox(mass, bodyComp.Width, bodyComp.Height)
}

func (ps *PhysicsSystem) createBodyInfo(w *ecs.World, e ecs.Entity, transform *component.Transform, bodyComp *component.PhysicsBody, isPlayer bool, isAnchor bool, isAI bool) *bodyInfo {
	if ps.space == nil {
		return nil
	}

	width := bodyComp.Width
	height := bodyComp.Height
	radius := bodyComp.Radius

	if radius <= 0 && (width <= 0 || height <= 0) {
		width = 32
		height = 32
	}

	sizeW, sizeH := width, height
	if radius > 0 {
		sizeW = radius * 2
		sizeH = radius * 2
	}

	topLeftX := aabbTopLeftX(w, e, transform.X, bodyComp.OffsetX, sizeW, bodyComp.AlignTopLeft)
	topLeftY := aabbTopLeftY(transform.Y, bodyComp.OffsetY, sizeH, bodyComp.AlignTopLeft)

	centerX := topLeftX + sizeW/2
	centerY := topLeftY + sizeH/2

	info := &bodyInfo{static: bodyComp.Static}

	if bodyComp.Static {
		var shape *cp.Shape
		if radius > 0 {
			shape = cp.NewCircle(ps.space.StaticBody, radius, cp.Vector{X: centerX, Y: centerY})
		} else {
			bb := cp.BB{L: topLeftX, B: topLeftY, R: topLeftX + sizeW, T: topLeftY + sizeH}
			shape = cp.NewBox2(ps.space.StaticBody, bb, 0)
		}
		shape.SetFriction(bodyComp.Friction)
		shape.SetElasticity(bodyComp.Elasticity)
		if isAI {
			shape.SetCollisionType(collisionTypeAI)
		} else {
			shape.SetCollisionType(collisionTypeSolid)
		}
		if isAnchor {
			shape.SetSensor(true)
		}
		ps.space.AddShape(shape)

		info.body = ps.space.StaticBody
		info.mainShape = shape
		info.shapes = []*cp.Shape{shape}
		return info
	}

	mass := bodyComp.Mass
	if mass <= 0 {
		mass = 1
	}

	body := cp.NewBody(mass, physicsBodyMoment(bodyComp, isAI))
	body.SetPosition(cp.Vector{X: centerX, Y: centerY})
	body.SetAngle(transform.Rotation)
	body.SetAngularVelocity(0)

	var shape *cp.Shape
	if radius > 0 {
		shape = cp.NewCircle(body, radius, cp.Vector{})
	} else {
		shape = cp.NewBox(body, width, height, 0)
	}

	shape.SetFriction(bodyComp.Friction)
	shape.SetElasticity(bodyComp.Elasticity)
	if isAI {
		shape.SetCollisionType(collisionTypeAI)
	} else {
		shape.SetCollisionType(collisionTypeSolid)
	}
	if isAnchor {
		shape.SetSensor(true)
	}

	if isPlayer {
		shape.SetCollisionType(collisionTypePlayer)
	}

	ps.space.AddBody(body)
	ps.space.AddShape(shape)

	info.body = body
	info.mainShape = shape
	info.shapes = []*cp.Shape{shape}

	if isPlayer {
		groundShape := ps.createGroundSensor(bodyComp, body)
		if groundShape != nil {
			ps.space.AddShape(groundShape)
			info.groundShape = groundShape
			info.shapes = append(info.shapes, groundShape)
		}
	}

	return info
}

func (ps *PhysicsSystem) createGroundSensor(bodyComp *component.PhysicsBody, body *cp.Body) *cp.Shape {
	if body == nil {
		return nil
	}
	width := bodyComp.Width
	height := bodyComp.Height
	if width <= 0 || height <= 0 {
		return nil
	}

	groundBB := cp.BB{
		L: -width * 0.45,
		B: height / 2.0,
		R: width * 0.45,
		T: height/2.0 + 2,
	}

	groundShape := cp.NewBox2(body, groundBB, 0)
	groundShape.SetSensor(true)
	groundShape.SetCollisionType(collisionTypePlayerGround)
	return groundShape
}

func (ps *PhysicsSystem) syncWorldBounds(w *ecs.World) {
	if ps.space == nil || w == nil {
		return
	}
	boundsEntity, ok := ecs.First(w, component.LevelBoundsComponent.Kind())
	if !ok {
		return
	}
	if _, exists := ps.entities[boundsEntity]; exists {
		return
	}
	bounds, ok := ecs.Get(w, boundsEntity, component.LevelBoundsComponent.Kind())
	if !ok {
		return
	}

	worldW := bounds.Width
	worldH := bounds.Height
	if worldW <= 0 || worldH <= 0 {
		return
	}

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

	info := &bodyInfo{static: true, body: ps.space.StaticBody}
	for _, seg := range segments {
		shape := cp.NewSegment(ps.space.StaticBody, seg.a, seg.b, thickness)
		shape.SetFriction(0.8)
		shape.SetCollisionType(collisionTypeSolid)
		ps.space.AddShape(shape)
		info.shapes = append(info.shapes, shape)
	}

	// map shapes to the bounds entity
	for _, s := range info.shapes {
		if s != nil {
			ps.shapeEntity[s] = boundsEntity
		}
	}

	ps.entities[boundsEntity] = info
}

func (ps *PhysicsSystem) resetPlayerContacts(w *ecs.World) {
	if w == nil {
		return
	}

	// TODO - get count of entities with PlayerCollisionComponent and use that to pre-size the seen map
	seen := make(map[ecs.Entity]struct{})
	ecs.ForEach(w, component.PlayerCollisionComponent.Kind(), func(e ecs.Entity, pc *component.PlayerCollision) {
		seen[e] = struct{}{}

		st := ps.playerStates[e]
		if st == nil {
			st = &playerContactState{}
			ps.playerStates[e] = st
		}

		st.groundGrace = pc.GroundGrace
		if st.groundGrace > 0 {
			st.groundGrace--
		}

		st.grounded = false
		st.wall = wallNone
		ps.playerAIColl[e] = false
	})

	for e := range ps.playerStates {
		if _, ok := seen[e]; !ok {
			delete(ps.playerStates, e)
		}
	}
}

func (ps *PhysicsSystem) flushPlayerContacts(w *ecs.World) {
	if w == nil {
		return
	}
	for e, st := range ps.playerStates {
		if !ecs.IsAlive(w, e) {
			continue
		}
		pc, ok := ecs.Get(w, e, component.PlayerCollisionComponent.Kind())
		if !ok {
			continue
		}
		pc.Grounded = st.grounded
		pc.GroundGrace = st.groundGrace
		pc.Wall = st.wall
		pc.Clamber = false
		pc.ClamberTargetX = 0
		pc.ClamberTargetY = 0
		if !st.grounded {
			if bodyComp, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind()); ok && bodyComp != nil {
				if targetX, targetY, ok := ps.findBestPlayerClamberTarget(w, e, bodyComp, st.wall); ok {
					pc.Clamber = true
					pc.ClamberTargetX = targetX
					pc.ClamberTargetY = targetY
				}
			}
		}
		// set collision-with-AI flag from physics handler
		if collided, ok := ps.playerAIColl[e]; ok {
			pc.CollidedAI = collided
		} else {
			pc.CollidedAI = false
		}
		_ = ecs.Add(w, e, component.PlayerCollisionComponent.Kind(), pc)
	}
}

func (ps *PhysicsSystem) findBestPlayerClamberTarget(w *ecs.World, playerEntity ecs.Entity, bodyComp *component.PhysicsBody, preferredWall int) (float64, float64, bool) {
	if w == nil || bodyComp == nil {
		return 0, 0, false
	}
	transform, ok := ecs.Get(w, playerEntity, component.TransformComponent.Kind())
	if !ok || transform == nil {
		return 0, 0, false
	}
	centerX, _, ok := physicsBodyCenter(w, playerEntity, transform, bodyComp)
	if !ok {
		return 0, 0, false
	}

	sides := []int{wallLeft, wallRight}
	if preferredWall == wallRight {
		sides = []int{wallRight, wallLeft}
	}

	bestDistance := math.Inf(1)
	bestX := 0.0
	bestY := 0.0
	found := false
	for _, side := range sides {
		targetX, targetY, ok := ps.findPlayerClamberTarget(w, playerEntity, bodyComp, side)
		if !ok {
			continue
		}
		distance := math.Abs(targetX - centerX)
		if distance < bestDistance {
			bestDistance = distance
			bestX = targetX
			bestY = targetY
			found = true
		}
	}
	if !found {
		return 0, 0, false
	}
	return bestX, bestY, true
}

func (ps *PhysicsSystem) findPlayerClamberTarget(w *ecs.World, playerEntity ecs.Entity, bodyComp *component.PhysicsBody, wallSide int) (float64, float64, bool) {
	if w == nil || bodyComp == nil || wallSide == wallNone {
		return 0, 0, false
	}
	player, ok := ecs.Get(w, playerEntity, component.PlayerComponent.Kind())
	if !ok || player == nil {
		return 0, 0, false
	}
	transform, ok := ecs.Get(w, playerEntity, component.TransformComponent.Kind())
	if !ok || transform == nil {
		return 0, 0, false
	}
	centerX, centerY, ok := physicsBodyCenter(w, playerEntity, transform, bodyComp)
	if !ok {
		return 0, 0, false
	}
	playerWidth := bodyComp.Width
	playerHeight := bodyComp.Height
	if playerWidth <= 0 {
		playerWidth = 32
	}
	if playerHeight <= 0 {
		playerHeight = 32
	}
	halfW := playerWidth / 2
	halfH := playerHeight / 2
	playerMinX := centerX - halfW
	playerMaxX := centerX + halfW
	playerClamberMinY := centerY - playerHeight*0.25
	playerMaxY := centerY + halfH

	bestScore := math.Inf(1)
	bestX := 0.0
	bestY := 0.0

	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(other ecs.Entity, otherBody *component.PhysicsBody, otherTransform *component.Transform) {
		if other == playerEntity || otherBody == nil || otherTransform == nil || otherBody.Disabled || !otherBody.Static {
			return
		}
		if ecs.Has(w, other, component.AnchorTagComponent.Kind()) || ecs.Has(w, other, component.HazardComponent.Kind()) || ecs.Has(w, other, component.PlayerTagComponent.Kind()) || ecs.Has(w, other, component.AITagComponent.Kind()) {
			return
		}
		minX, minY, maxX, _, ok := physicsBodyBounds(w, other, otherTransform, otherBody)
		if !ok {
			return
		}
		ledgeY := minY
		if ledgeY < playerClamberMinY-clamberCandidateMargin || ledgeY >= playerMaxY-clamberCandidateMargin {
			return
		}

		targetX := 0.0
		switch wallSide {
		case wallRight:
			if math.Abs(playerMaxX-minX) > clamberWallTolerance {
				return
			}
			targetX = minX + halfW + playerClamberInset(player)
		case wallLeft:
			if math.Abs(playerMinX-maxX) > clamberWallTolerance {
				return
			}
			targetX = maxX - halfW - playerClamberInset(player)
		default:
			return
		}

		targetY := ledgeY - halfH - clamberStandClearance
		if ps.clamberTargetBlocked(w, playerEntity, targetX, targetY, playerWidth, playerHeight) {
			return
		}

		score := math.Abs(ledgeY - playerClamberMinY)
		if score < bestScore {
			bestScore = score
			bestX = targetX
			bestY = targetY
		}
	})

	if math.IsInf(bestScore, 1) {
		return 0, 0, false
	}
	return bestX, bestY, true
}

func (ps *PhysicsSystem) clamberTargetBlocked(w *ecs.World, playerEntity ecs.Entity, targetX, targetY, playerWidth, playerHeight float64) bool {
	targetMinX := targetX - playerWidth/2
	targetMinY := targetY - playerHeight/2
	targetMaxX := targetX + playerWidth/2
	targetMaxY := targetY + playerHeight/2
	blocked := false
	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(other ecs.Entity, otherBody *component.PhysicsBody, otherTransform *component.Transform) {
		if blocked || other == playerEntity || otherBody == nil || otherTransform == nil || otherBody.Disabled || !otherBody.Static {
			return
		}
		if ecs.Has(w, other, component.AnchorTagComponent.Kind()) || ecs.Has(w, other, component.HazardComponent.Kind()) || ecs.Has(w, other, component.PlayerTagComponent.Kind()) || ecs.Has(w, other, component.AITagComponent.Kind()) {
			return
		}
		minX, minY, maxX, maxY, ok := physicsBodyBounds(w, other, otherTransform, otherBody)
		if !ok {
			return
		}
		if targetMaxX > minX+clamberCandidateMargin && targetMinX < maxX-clamberCandidateMargin && targetMaxY > minY+clamberCandidateMargin && targetMinY < maxY-clamberCandidateMargin {
			blocked = true
		}
	})
	return blocked
}

func playerClamberInset(player *component.Player) float64 {
	if player == nil || player.ClamberInset <= 0 {
		return 4
	}
	return player.ClamberInset
}

func (ps *PhysicsSystem) syncTransforms(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, bodyComp *component.PhysicsBody, transform *component.Transform) {
		if bodyComp == nil || bodyComp.Disabled || bodyComp.Static || bodyComp.Body == nil {
			return
		}

		pos := bodyComp.Body.Position()
		effectiveOffsetX := facingAdjustedOffsetX(w, e, bodyComp.OffsetX, bodyComp.Width, bodyComp.AlignTopLeft)
		if bodyComp.AlignTopLeft {
			transform.X = pos.X - bodyComp.Width/2.0 - effectiveOffsetX
			transform.Y = pos.Y - bodyComp.Height/2.0 - bodyComp.OffsetY
		} else {
			transform.X = pos.X - effectiveOffsetX
			transform.Y = pos.Y - bodyComp.OffsetY
		}
		transform.Rotation = bodyComp.Body.Angle()
		// _ = ecs.Add(w, e, component.TransformComponent.Kind(), transform)
	})
}

func (ps *PhysicsSystem) cleanupEntities(w *ecs.World) {
	for e, info := range ps.entities {
		keep := false
		if ecs.IsAlive(w, e) {
			if body, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind()); ok && body != nil && !body.Disabled {
				keep = true
			}
			if ecs.Has(w, e, component.LevelBoundsComponent.Kind()) {
				keep = true
			}
		}
		if keep {
			continue
		}

		for _, shape := range info.shapes {
			if shape == nil || ps.space == nil {
				continue
			}
			ps.space.RemoveShape(shape)
			delete(ps.playerShapes, shape)
			delete(ps.groundShapes, shape)
			delete(ps.aiShapes, shape)
			delete(ps.shapeEntity, shape)
		}
		if info.body != nil && !info.static && ps.space != nil {
			ps.space.RemoveBody(info.body)
		}

		delete(ps.entities, e)
		delete(ps.playerStates, e)
	}

	for shape, entity := range ps.playerShapes {
		if !ecs.IsAlive(w, entity) {
			delete(ps.playerShapes, shape)
		}
	}
	for shape, entity := range ps.groundShapes {
		if !ecs.IsAlive(w, entity) {
			delete(ps.groundShapes, shape)
		}
	}
	for shape, entity := range ps.aiShapes {
		if !ecs.IsAlive(w, entity) {
			delete(ps.aiShapes, shape)
		}
	}
}
