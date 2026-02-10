package system

import (
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	collisionTypePlayer cp.CollisionType = iota + 1
	collisionTypePlayerGround
	collisionTypeSolid
)

const (
	wallNone  = 0
	wallLeft  = 1
	wallRight = 2
)

const groundGraceFrames = 6

type PhysicsSystem struct {
	space         *cp.Space
	handlersReady bool

	entities     map[ecs.Entity]*bodyInfo
	playerShapes map[*cp.Shape]ecs.Entity
	groundShapes map[*cp.Shape]ecs.Entity
	playerStates map[ecs.Entity]*playerContactState
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
		playerStates: make(map[ecs.Entity]*playerContactState),
	}
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

	// Process any anchors marked for pending destroy: remove their physics
	// constraints from the space, then destroy the entity. This avoids
	// injecting other systems into physics; systems can mark anchors for
	// removal via the AnchorPendingDestroy component.
	for _, e := range w.Query(component.AnchorPendingDestroyComponent.Kind()) {
		if !w.IsAlive(e) {
			continue
		}
		if jc, ok := ecs.Get(w, e, component.AnchorJointComponent); ok {
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
		w.DestroyEntity(e)
	}

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

	ps.space.Step(1.0)

	ps.syncTransforms(w)
	ps.flushPlayerContacts(w)
}

func (ps *PhysicsSystem) processAnchorConstraints(w *ecs.World) {
	if ps == nil || ps.space == nil || w == nil {
		return
	}

	playerEnt, ok := w.First(component.PlayerTagComponent.Kind())
	if !ok {
		return
	}
	playerBodyComp, ok := ecs.Get(w, playerEnt, component.PhysicsBodyComponent)
	if !ok || playerBodyComp.Body == nil {
		return
	}

	anchors := w.Query(component.AnchorConstraintRequestComponent.Kind(), component.AnchorTagComponent.Kind())
	for _, e := range anchors {
		req, ok := ecs.Get(w, e, component.AnchorConstraintRequestComponent)
		if !ok {
			continue
		}
		if req.Applied {
			continue
		}

		jointComp, _ := ecs.Get(w, e, component.AnchorJointComponent)

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
			if jointComp.Slide == nil {
				minLen := req.MinLen
				maxLen := req.MaxLen
				if maxLen <= 0 {
					maxLen = 100000.0
				}
				slide := cp.NewSlideJoint(playerBodyComp.Body, ps.space.StaticBody, cp.Vector{X: 0, Y: 0}, cp.Vector{X: req.AnchorX, Y: req.AnchorY}, minLen, maxLen)
				ps.space.AddConstraint(slide)
				jointComp.Slide = slide
			}
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
				pivot := cp.NewPivotJoint(playerBodyComp.Body, ps.space.StaticBody, cp.Vector{X: req.AnchorX, Y: req.AnchorY})
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
			if jointComp.Pin == nil {
				pin := cp.NewPinJoint(playerBodyComp.Body, ps.space.StaticBody, cp.Vector{X: 0, Y: 0}, cp.Vector{X: req.AnchorX, Y: req.AnchorY})
				ps.space.AddConstraint(pin)
				jointComp.Pin = pin
			}
		default:
			continue
		}

		if err := ecs.Add(w, e, component.AnchorJointComponent, jointComp); err != nil {
			panic("physics system: update anchor joint: " + err.Error())
		}
		req.Applied = true
		if err := ecs.Add(w, e, component.AnchorConstraintRequestComponent, req); err != nil {
			panic("physics system: update anchor request: " + err.Error())
		}
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
}

func (ps *PhysicsSystem) syncEntities(w *ecs.World) {
	if ps.space == nil {
		return
	}

	ps.cleanupEntities(w)

	entities := w.Query(component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind())
	for _, e := range entities {
		bodyComp, ok := ecs.Get(w, e, component.PhysicsBodyComponent)
		if !ok {
			continue
		}
		transform, ok := ecs.Get(w, e, component.TransformComponent)
		if !ok {
			continue
		}

		isPlayer := ecs.Has(w, e, component.PlayerTagComponent)
		isAnchor := ecs.Has(w, e, component.AnchorTagComponent)

		info := ps.entities[e]
		if info != nil && info.mainShape != nil {
			if isPlayer {
				if info.mainShape != nil {
					ps.playerShapes[info.mainShape] = e
				}
				if info.groundShape != nil {
					ps.groundShapes[info.groundShape] = e
				}
			}
			if bodyComp.Body == nil || bodyComp.Shape == nil {
				bodyComp.Body = info.body
				bodyComp.Shape = info.mainShape
				_ = ecs.Add(w, e, component.PhysicsBodyComponent, bodyComp)
			}
			continue
		}

		info = ps.createBodyInfo(transform, bodyComp, isPlayer, isAnchor)
		if info == nil || info.mainShape == nil {
			continue
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
		bodyComp.Body = info.body
		bodyComp.Shape = info.mainShape
		_ = ecs.Add(w, e, component.PhysicsBodyComponent, bodyComp)
	}
}

func (ps *PhysicsSystem) createBodyInfo(transform component.Transform, bodyComp component.PhysicsBody, isPlayer bool, isAnchor bool) *bodyInfo {
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

	topLeftX := transform.X + bodyComp.OffsetX
	topLeftY := transform.Y + bodyComp.OffsetY
	if !bodyComp.AlignTopLeft {
		topLeftX = transform.X + bodyComp.OffsetX - sizeW/2
		topLeftY = transform.Y + bodyComp.OffsetY - sizeH/2
	}

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
		shape.SetCollisionType(collisionTypeSolid)
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

	var moment float64
	if radius > 0 {
		moment = cp.MomentForCircle(mass, 0, radius, cp.Vector{})
	} else {
		moment = cp.MomentForBox(mass, width, height)
	}

	body := cp.NewBody(mass, moment)
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
	shape.SetCollisionType(collisionTypeSolid)
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

func (ps *PhysicsSystem) createGroundSensor(bodyComp component.PhysicsBody, body *cp.Body) *cp.Shape {
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
	boundsEntity, ok := w.First(component.LevelBoundsComponent.Kind())
	if !ok {
		return
	}
	if _, exists := ps.entities[boundsEntity]; exists {
		return
	}
	bounds, ok := ecs.Get(w, boundsEntity, component.LevelBoundsComponent)
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

	ps.entities[boundsEntity] = info
}

func (ps *PhysicsSystem) resetPlayerContacts(w *ecs.World) {
	if w == nil {
		return
	}
	players := w.Query(component.PlayerCollisionComponent.Kind())
	seen := make(map[ecs.Entity]struct{}, len(players))
	for _, e := range players {
		seen[e] = struct{}{}
		pc, ok := ecs.Get(w, e, component.PlayerCollisionComponent)
		if !ok {
			continue
		}
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
	}

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
		if !w.IsAlive(e) {
			continue
		}
		pc, ok := ecs.Get(w, e, component.PlayerCollisionComponent)
		if !ok {
			continue
		}
		pc.Grounded = st.grounded
		pc.GroundGrace = st.groundGrace
		pc.Wall = st.wall
		_ = ecs.Add(w, e, component.PlayerCollisionComponent, pc)
	}
}

func (ps *PhysicsSystem) syncTransforms(w *ecs.World) {
	if w == nil {
		return
	}
	entities := w.Query(component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind())
	for _, e := range entities {
		bodyComp, ok := ecs.Get(w, e, component.PhysicsBodyComponent)
		if !ok || bodyComp.Body == nil {
			continue
		}
		if bodyComp.Static {
			continue
		}
		transform, ok := ecs.Get(w, e, component.TransformComponent)
		if !ok {
			continue
		}
		pos := bodyComp.Body.Position()
		if bodyComp.AlignTopLeft {
			transform.X = pos.X - bodyComp.Width/2.0 - bodyComp.OffsetX
			transform.Y = pos.Y - bodyComp.Height/2.0 - bodyComp.OffsetY
		} else {
			transform.X = pos.X - bodyComp.OffsetX
			transform.Y = pos.Y - bodyComp.OffsetY
		}
		transform.Rotation = bodyComp.Body.Angle()
		_ = ecs.Add(w, e, component.TransformComponent, transform)
	}
}

func (ps *PhysicsSystem) cleanupEntities(w *ecs.World) {
	for e, info := range ps.entities {
		keep := false
		if w.IsAlive(e) {
			if ecs.Has(w, e, component.PhysicsBodyComponent) || ecs.Has(w, e, component.LevelBoundsComponent) {
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
		}
		if info.body != nil && !info.static && ps.space != nil {
			ps.space.RemoveBody(info.body)
		}

		delete(ps.entities, e)
		delete(ps.playerStates, e)
	}

	for shape, entity := range ps.playerShapes {
		if !w.IsAlive(entity) {
			delete(ps.playerShapes, shape)
		}
	}
	for shape, entity := range ps.groundShapes {
		if !w.IsAlive(entity) {
			delete(ps.groundShapes, shape)
		}
	}
}
