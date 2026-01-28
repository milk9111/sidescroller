package obj

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
	"golang.org/x/image/colornames"
)

// playerState is the interface each concrete player state implements.
type playerState interface {
	Enter(p *Player)
	Exit(p *Player)
	HandleInput(p *Player)
	OnPhysics(p *Player)
	Name() string
}

const (
	jumpHeight            = -12
	jumpCutVelocity       = -4
	jumpBufferTimerAmount = 10 // frames
	coyoteTimeFrames      = 6  // allow jump within this many frames after leaving ground
	// physics forces/impulses
	moveForce           = 0.2 // stronger horizontal force for snappier response
	jumpImpulse         = -12.0
	brakeForceFactor    = 0.2 // much stronger braking when no input
	nonPhysicsDecelMult = 0.2 // faster deceleration when not using physics body
	// rope adjust speed (pixels per physics step)
	ropeAdjustSpeed = 8.0
)

// setState helper switches states and calls Enter.
func (p *Player) setState(s playerState) {
	p.state.Exit(p)
	p.state = s
	p.state.Enter(p)
	// switch animation based on state
	switch s {
	case stateIdle:
		if p.animIdle != nil {
			p.anim = p.animIdle
			p.anim.Reset()
		}
	case stateRunning:
		if p.animRun != nil {
			p.anim = p.animRun
			p.anim.Reset()
		}
	default:
		// keep current animation for other states
	}
}

// Concrete states
type idleState struct{}

func (idleState) Name() string { return "idle" }
func (idleState) Enter(p *Player) {
	fmt.Println("entered idle state")
}
func (idleState) Exit(p *Player) {}
func (idleState) HandleInput(p *Player) {
	if p.Input.JumpPressed {
		p.setState(stateJumping)
		return
	}

	if p.Input.MoveX != 0 {
		p.setState(stateRunning)
	}
}
func (idleState) OnPhysics(p *Player) {
	if !p.CollisionWorld.IsGrounded(p.Rect) {
		p.setState(stateFalling)
	}
}

type runningState struct{}

func (runningState) Name() string { return "running" }
func (runningState) Enter(p *Player) {
	fmt.Println("entered running state")
}
func (runningState) Exit(p *Player) {}
func (runningState) HandleInput(p *Player) {
	if p.Input.JumpPressed {
		p.setState(stateJumping)
		return
	}

	if p.Input.MoveX == 0 {
		p.setState(stateIdle)
	}
}
func (runningState) OnPhysics(p *Player) {
	p.applyMoveXForce()

	if !p.CollisionWorld.IsGrounded(p.Rect) {
		p.setState(stateFalling)
	}
}

type jumpingState struct{}

func (jumpingState) Name() string { return "jumping" }
func (jumpingState) Enter(p *Player) {
	fmt.Println("entered jumping state")
	p.body.ApplyImpulseAtLocalPoint(cp.Vector{X: 0, Y: jumpImpulse}, cp.Vector{})
}
func (jumpingState) Exit(p *Player) {}
func (jumpingState) HandleInput(p *Player) {
	if p.Input.JumpPressed {
		if !p.doubleJumped {
			p.doubleJumped = true
			p.setState(stateDoubleJumping)
			// fmt.Println("double jump from jumping")
			return
		}
		// already used double jump -> record buffer for next landing
		p.jumpBuffer = true
		p.jumpBufferTimer = jumpBufferTimerAmount
	}
}
func (jumpingState) OnPhysics(p *Player) {
	p.applyMoveXForce()

	if p.VelocityY > 0 {
		p.setState(stateFalling)
	}
}

type doubleJumpingState struct{}

func (doubleJumpingState) Name() string { return "doublejump" }
func (doubleJumpingState) Enter(p *Player) {
	fmt.Println("entered double jumping state")
	p.body.ApplyImpulseAtLocalPoint(cp.Vector{X: 0, Y: jumpImpulse}, cp.Vector{})
}
func (doubleJumpingState) Exit(p *Player) {}
func (doubleJumpingState) HandleInput(p *Player) {
	if p.Input.JumpPressed {
		// already double-jumped; record buffer for landing
		p.jumpBuffer = true
		p.jumpBufferTimer = jumpBufferTimerAmount
	}
}
func (doubleJumpingState) OnPhysics(p *Player) {
	p.applyMoveXForce()

	if p.VelocityY > 0 {
		p.setState(stateFalling)
	}
}

type fallingState struct{}

func (fallingState) Name() string { return "falling" }
func (fallingState) Enter(p *Player) {
	fmt.Println("entered falling state")
}
func (fallingState) Exit(p *Player) {}
func (fallingState) HandleInput(p *Player) {
	if p.Input.JumpPressed {
		// allow coyote jump shortly after leaving ground
		if p.coyoteTimer > 0 && !p.doubleJumped {
			p.coyoteTimer = 0
			p.setState(stateJumping)
			return
		}
		if !p.doubleJumped {
			p.doubleJumped = true
			p.setState(stateDoubleJumping)
			return
		}

		// already used double jump -> record buffer for next landing
		p.jumpBuffer = true
		p.jumpBufferTimer = jumpBufferTimerAmount
	}
}
func (fallingState) OnPhysics(p *Player) {
	p.applyMoveXForce()

	if p.CollisionWorld.IsGrounded(p.Rect) {
		// fmt.Println("landed from falling")
		if p.Input.MoveX != 0 {
			p.setState(stateRunning)
		} else {
			p.setState(stateIdle)
		}
		p.doubleJumped = false
	}

	if p.CollisionWorld.IsTouchingWall(p.Rect) != WALL_NONE {
		p.setState(stateWallGrab)
		p.doubleJumped = false
	}
}

type wallGrabState struct {
	wallS   wallSide
	elapsed int
}

func (w *wallGrabState) Name() string { return "wall grab" }
func (w *wallGrabState) Enter(p *Player) {
	p.GravityEnabled = false
	w.elapsed = 0
	w.wallS = p.CollisionWorld.IsTouchingWall(p.Rect)
	fmt.Println("entered wall grab state")
}
func (w *wallGrabState) Exit(p *Player) {
	p.GravityEnabled = true
}
func (w *wallGrabState) HandleInput(p *Player) {
	if p.Input.JumpPressed {
		// horizontal push-off impulse
		if w.wallS == WALL_LEFT {
			p.body.ApplyImpulseAtLocalPoint(cp.Vector{X: 2, Y: 0}, cp.Vector{})
		} else if w.wallS == WALL_RIGHT {
			p.body.ApplyImpulseAtLocalPoint(cp.Vector{X: -2, Y: 0}, cp.Vector{})
		}
		p.setState(stateJumping)
		return
	}
}
func (w *wallGrabState) OnPhysics(p *Player) {
	if w.wallS == WALL_LEFT {
		p.facingRight = true
	} else if w.wallS == WALL_RIGHT {
		p.facingRight = false
	}

	// while grabbing, clamp vertical movement; if using physics body adjust body velocity
	if float64(w.elapsed) < ebiten.ActualTPS()/2 {
		v := p.body.Velocity()
		p.body.SetVelocity(v.X, 0)
		p.VelocityY = 0
	} else {
		v := p.body.Velocity()
		// gently slide down
		p.body.SetVelocity(v.X, v.Y+1.5*float64(1/ebiten.ActualTPS()))
		p.VelocityY = float32(p.body.Velocity().Y)
	}
	w.elapsed++

	if p.CollisionWorld.IsGrounded(p.Rect) {
		if p.Input.MoveX != 0 {
			p.setState(stateRunning)
		} else {
			p.setState(stateIdle)
		}
		p.doubleJumped = false
		return
	}

	if p.CollisionWorld.IsTouchingWall(p.Rect) == WALL_NONE {
		p.setState(stateFalling)
	}
}

type aimingState struct{}

func (aimingState) Name() string { return "aiming" }
func (aimingState) Enter(p *Player) {
	fmt.Println("entered aiming state")
}
func (aimingState) Exit(p *Player) {}
func (aimingState) HandleInput(p *Player) {
	// Left click to attach to a physics tile
	if p.Input.MouseLeftPressed {
		mx := p.Input.MouseWorldX
		my := p.Input.MouseWorldY
		if p.tryAttachAnchor(mx, my) {
			p.setState(stateIdle)
			return
		}
	}
}
func (aimingState) OnPhysics(p *Player) {}

type swingingState struct{}

func (swingingState) Name() string { return "swinging" }
func (swingingState) Enter(p *Player) {
	fmt.Println("entered swinging state")
}
func (swingingState) Exit(p *Player) {}
func (swingingState) HandleInput(p *Player) {
	// pressing E again detaches
	if p.Input.AimPressed {
		p.detachAnchor()
		p.setState(stateFalling)
		return
	}
}
func (swingingState) OnPhysics(p *Player) {}

// singletons for each state to avoid allocating on every transition
var (
	stateIdle          playerState = &idleState{}
	stateRunning       playerState = &runningState{}
	stateJumping       playerState = &jumpingState{}
	stateDoubleJumping playerState = &doubleJumpingState{}
	stateFalling       playerState = &fallingState{}
	stateWallGrab      playerState = &wallGrabState{}
	stateAiming        playerState = &aimingState{}
	stateSwinging      playerState = &swingingState{}
)

type Player struct {
	common.Rect
	StartX, StartY float32
	VelocityX      float32
	VelocityY      float32
	GravityEnabled bool
	Input          *Input
	CollisionWorld *CollisionWorld
	body           *cp.Body
	shape          *cp.Shape

	// Aiming / swing fields
	anchorActive bool
	anchorPos    cp.Vector
	anchorJoint  *cp.Constraint

	frames          int
	state           playerState
	doubleJumped    bool
	jumpBuffer      bool
	jumpBufferTimer int
	coyoteTimer     int
	prevJumpHeld    bool
	img             *ebiten.Image

	anim         *component.Animation
	animIdle     *component.Animation
	animRun      *component.Animation
	animWallGrab *component.Animation

	facingRight bool
	// RenderWidth/RenderHeight control the drawn sprite size. They are
	// independent from the collision AABB (`Width`/`Height` in `Rect`).
	RenderWidth  float32
	RenderHeight float32
}

func NewPlayer(
	x, y float32,
	input *Input,
	collisionWorld *CollisionWorld,
) *Player {
	p := &Player{
		Rect: common.Rect{
			X:      x,
			Y:      y,
			Width:  32,
			Height: 64,
		},
		StartX:         x,
		StartY:         y,
		GravityEnabled: true,
		Input:          input,
		CollisionWorld: collisionWorld,
		state:          stateIdle,
		facingRight:    true,
	}
	p.state.Enter(p)
	p.img = ebiten.NewImage(int(p.Width), int(p.Height))
	p.img.Fill(colornames.Crimson)
	// default render size matches the sprite frame size to avoid scaling artifacts
	// Temporarily use the source frame size (64) to check whether scaling
	// is the source of the artifact. If this removes the artifact we will
	// investigate better downscaling strategies.
	p.RenderWidth = 64
	p.RenderHeight = 64

	p.animIdle = component.NewAnimationRow(assets.PlayerSheet, 64, 64, 0, 9, 12, true)
	p.animRun = component.NewAnimationRow(assets.PlayerSheet, 64, 64, 1, 7, 12, true)
	p.animWallGrab = component.NewAnimationRow(assets.PlayerSheet, 64, 64, 2, 1, 12, false)

	// pre-scaling removed: frames will be scaled at draw-time
	p.anim = p.animIdle
	if p.CollisionWorld != nil {
		p.CollisionWorld.AttachPlayer(p)
	}

	return p
}

func (p *Player) Update() {
	p.frames++

	// update facing direction when moving
	if p.Input.MoveX < 0 {
		p.facingRight = false
	} else if p.Input.MoveX > 0 {
		p.facingRight = true
	}
	// manage jump buffer timer
	if p.jumpBuffer {
		p.jumpBufferTimer--
		if p.jumpBufferTimer <= 0 {
			p.jumpBuffer = false
		}
	}
	// (jump buffer is handled by airborne states)
	// Let current state handle input-driven behavior/transitions.
	// toggle aiming state with E
	if p.Input.AimPressed {
		if p.state == stateAiming {
			p.setState(stateIdle)
		} else if !p.anchorActive {
			p.setState(stateAiming)
		} else if p.anchorActive {
			p.detachAnchor()
		}
	}

	p.state.HandleInput(p)

	if p.prevJumpHeld && !p.Input.JumpHeld {
		p.applyJumpCut()
	}
	p.prevJumpHeld = p.Input.JumpHeld

	p.state.OnPhysics(p)
	p.applyPhysics()
	p.checkCollisions()

	// update coyote timer: reset when grounded, count down when airborne
	if p.CollisionWorld.IsGrounded(p.Rect) {
		p.coyoteTimer = coyoteTimeFrames
	} else if p.coyoteTimer > 0 {
		p.coyoteTimer--
	}

	if p.anim != nil {
		p.anim.Update()
	}

	// Apply buffered jump if we landed this frame
	if p.jumpBuffer && p.CollisionWorld.IsGrounded(p.Rect) {
		p.Input.JumpPressed = true
		p.jumpBuffer = false
		// re-handle input now that we're grounded
		p.state.HandleInput(p)
		p.Input.JumpPressed = false
	}

	// Let the state react to physics (velocity, grounded)

}

func (p *Player) GetState() string {
	if p.state != nil {
		return p.state.Name()
	}
	return "nil"
}

func (p *Player) applyJumpCut() {
	if p.state == stateJumping || p.state == stateDoubleJumping {
		if p.body != nil {
			v := p.body.Velocity()
			if v.Y < float64(jumpCutVelocity) {
				p.body.SetVelocity(v.X, float64(jumpCutVelocity))
				p.VelocityY = jumpCutVelocity
			}
		} else if p.VelocityY < jumpCutVelocity {
			p.VelocityY = jumpCutVelocity
		}
	}
}

func (p *Player) applyPhysics() {
	if p.CollisionWorld != nil && p.body != nil {
		p.CollisionWorld.BeginStep()
		// apply braking when no horizontal input and NOT anchored (don't kill swing momentum)
		if p.Input != nil && p.Input.MoveX == 0 && p.state != stateFalling {
			v := p.body.Velocity()
			brake := -v.X * brakeForceFactor
			p.body.ApplyForceAtLocalPoint(cp.Vector{X: brake, Y: 0}, cp.Vector{})
		}
		// adjust rope length while grounded and NOT falling (lock once falling)
		if p.anchorActive && p.Input != nil && p.state != stateFalling && p.CollisionWorld.IsGrounded(p.Rect) {
			if p.anchorJoint != nil {
				if sj, ok := p.anchorJoint.Class.(*cp.SlideJoint); ok {
					// increase/decrease max length based on horizontal input
					delta := float64(p.Input.MoveX) * ropeAdjustSpeed
					newMax := sj.Max + delta
					if newMax < 0 {
						newMax = 0
					}
					sj.Max = newMax
				}
			}
		}
		// if we've entered falling state, replace the slide joint with a pin joint
		if p.anchorActive && p.state == stateFalling && p.anchorJoint != nil {
			if _, ok := p.anchorJoint.Class.(*cp.SlideJoint); ok {
				// compute anchors for pin joint so its Dist equals current distance
				// anchor on body: use body's local point corresponding to its world position
				anchorA := p.body.WorldToLocal(p.body.Position())
				anchorB := p.CollisionWorld.space.StaticBody.WorldToLocal(p.anchorPos)
				// remove slide joint
				p.CollisionWorld.space.RemoveConstraint(p.anchorJoint)
				// clear angular velocity to avoid large impulses on swap
				p.body.SetAngularVelocity(0)
				// create pin joint that locks the current distance
				newJoint := cp.NewPinJoint(p.body, p.CollisionWorld.space.StaticBody, anchorA, anchorB)
				// limit max force to avoid explosive impulses
				newJoint.SetMaxForce(1000)
				p.CollisionWorld.space.AddConstraint(newJoint)
				p.anchorJoint = newJoint
			}
		}
		// physics-driven integration: states apply forces/impulses to the body
		p.CollisionWorld.Step(1.0)
		v := p.body.Velocity()
		p.VelocityX = float32(v.X)
		p.VelocityY = float32(v.Y)
		// keep body rotation locked: also lock while anchored so the
		// player doesn't spin during swinging
		p.body.SetAngle(0)
		p.body.SetAngularVelocity(0)

		pos := p.body.Position()
		p.Rect.X = float32(pos.X - float64(p.Width)/2.0)
		p.Rect.Y = float32(pos.Y - float64(p.Height)/2.0)
		if math.IsNaN(float64(p.Rect.X)) || math.IsNaN(float64(p.Rect.Y)) || math.IsInf(float64(p.Rect.X), 0) || math.IsInf(float64(p.Rect.Y), 0) {
			p.resetToSpawn()
		}
		return
	}

	p.X += p.VelocityX
	p.Y += p.VelocityY
}

func (p *Player) checkCollisions() {
	if p.CollisionWorld != nil && p.CollisionWorld.HitTriangle() {
		p.resetToSpawn()
		return
	}
	if p.Y > float32(common.BaseHeight)-p.Height {
		p.resetToSpawn()
		return
	}

	clamped := false
	if p.X < 0 {
		p.X = 0
		p.VelocityX = 0
		clamped = true
	}

	if p.X+float32(p.Width) > float32(common.BaseWidth) {
		p.X = float32(common.BaseWidth) - float32(p.Width)
		p.VelocityX = 0
		clamped = true
	}

	if p.Y < 0 {
		p.Y = 0
		p.VelocityY = 0
		clamped = true
	}

	if clamped {
		p.syncBodyFromRect()
	}
}

func (p *Player) syncBodyFromRect() {
	if p.body == nil {
		return
	}
	p.body.SetPosition(cp.Vector{X: float64(p.X + p.Width/2), Y: float64(p.Y + p.Height/2)})
	p.body.SetVelocity(float64(p.VelocityX), float64(p.VelocityY))
}

func (p *Player) resetToSpawn() {
	p.Rect.X = p.StartX
	p.Rect.Y = p.StartY
	p.VelocityX = 0
	p.VelocityY = 0
	p.setState(stateIdle)
	p.doubleJumped = false
	if p.anim != nil {
		p.anim.Reset()
	}
	p.syncBodyFromRect()
}

// tryAttachAnchor attempts to attach a pivot joint from the player's body to
// the clicked tile at world coordinates mx,my. Returns true if attached.
func (p *Player) tryAttachAnchor(mx, my float64) bool {
	if p == nil || p.CollisionWorld == nil || p.CollisionWorld.level == nil || p.body == nil {
		return false
	}
	tx := int(math.Floor(mx / float64(common.TileSize)))
	ty := int(math.Floor(my / float64(common.TileSize)))
	if !p.CollisionWorld.level.physicsTileAt(tx, ty) {
		return false
	}
	// anchor at tile center
	ax := float64(tx*common.TileSize + common.TileSize/2)
	ay := float64(ty*common.TileSize + common.TileSize/2)
	anchorWorld := cp.Vector{X: ax, Y: ay}

	// compute initial length from body position to anchor
	bpos := p.body.Position()
	dx := bpos.X - anchorWorld.X
	dy := bpos.Y - anchorWorld.Y
	dist := math.Hypot(dx, dy)

	// anchors are specified in each body's local coordinates
	anchorA := p.body.WorldToLocal(anchorWorld)
	anchorB := p.CollisionWorld.space.StaticBody.WorldToLocal(anchorWorld)

	joint := cp.NewSlideJoint(p.body, p.CollisionWorld.space.StaticBody, anchorA, anchorB, 0, dist)
	p.CollisionWorld.space.AddConstraint(joint)
	p.anchorActive = true
	p.anchorPos = anchorWorld
	p.anchorJoint = joint
	return true
}

func (p *Player) detachAnchor() {
	if p == nil || p.CollisionWorld == nil || !p.anchorActive {
		return
	}
	if p.anchorJoint != nil {
		p.CollisionWorld.space.RemoveConstraint(p.anchorJoint)
	}
	p.anchorActive = false
	p.anchorJoint = nil
	if p.body != nil {
		p.body.SetAngle(0)
		p.body.SetAngularVelocity(0)
	}
}

func (p *Player) applyMoveXForce() {
	if p.Input.MoveX == 0 {
		return
	}

	fx := float64(p.Input.MoveX) * moveForce
	p.body.ApplyForceAtLocalPoint(cp.Vector{X: fx, Y: 0}, cp.Vector{})
}

func (p *Player) Draw(screen *ebiten.Image) {
	// center sprite within the collision AABB when render and collision sizes differ
	offsetX := (float64(p.RenderWidth) - float64(p.Width)) / 2.0
	offsetY := (float64(p.RenderHeight) - float64(p.Height)) / 2.0
	drawX := float64(p.X) - offsetX
	drawY := float64(p.Y) - offsetY

	if p.anim != nil {
		// If animation has been pre-scaled to RenderWidth/Height, draw it directly
		// without additional scaling. Otherwise, scale at draw-time.
		op := &ebiten.DrawImageOptions{}
		fw, fh := p.anim.Size()
		// scale at draw-time
		sx := float64(p.RenderWidth) / float64(fw)
		sy := float64(p.RenderHeight) / float64(fh)
		if p.facingRight {
			op.GeoM.Scale(sx, sy)
			tx := math.Round(drawX * sx)
			ty := math.Round(drawY * sy)
			op.GeoM.Translate(tx, ty)
		} else {
			op.GeoM.Scale(-sx, sy)
			tx := math.Round((drawX + float64(fw)) * sx)
			ty := math.Round(drawY * sy)
			op.GeoM.Translate(tx, ty)
		}
		op.Filter = ebiten.FilterNearest
		p.anim.Draw(screen, op)
	} else {
		op := &ebiten.DrawImageOptions{}
		if p.facingRight {
			op.GeoM.Translate(math.Round(drawX), math.Round(drawY))
		} else {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(math.Round(drawX+float64(p.RenderWidth)), math.Round(drawY))
		}
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(p.img, op)
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("State: %s, jumpHeld: %v, doubleJumped: %v", p.state.Name(), p.Input.JumpHeld, p.doubleJumped), 0, 20)

	// draw anchor line if attached
	if p.anchorActive {
		cx := float64(p.X + float32(p.Width)/2.0)
		cy := float64(p.Y + float32(p.Height)/2.0)
		ebitenutil.DrawLine(screen, cx, cy, p.anchorPos.X, p.anchorPos.Y, colornames.Red)
	}
}
