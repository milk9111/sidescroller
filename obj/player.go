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
	jumpHeight            = -6
	jumpCutVelocity       = -4
	jumpBufferTimerAmount = 10 // frames
	coyoteTimeFrames      = 6  // allow jump within this many frames after leaving ground
	// physics forces/impulses
	moveForce           = 0.2 // stronger horizontal force for snappier response
	swingMoveForce      = 0.45
	jumpImpulse         = -8.0
	brakeForceFactor    = 0.3 // much stronger braking when no input
	nonPhysicsDecelMult = 0.3 // faster deceleration when not using physics body
	// velocity caps
	maxSpeedX      = 6.0
	maxSwingSpeedX = 16.0
	maxSpeedY      = 14.0
	// rope adjust speed (pixels per physics step)
	ropeAdjustSpeed = 8.0
	// double-jump tuned values: set a consistent upward velocity and
	// a small horizontal impulse so the double-jump feels impactful at speed
	doubleJumpVelocity     = -14.0
	doubleJumpHorizImpulse = 2.0
	// dash settings
	dashSpeed  = 16.0
	dashFrames = 12
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
	case stateWallGrab:
		if p.animWallGrab != nil {
			p.anim = p.animWallGrab
			p.anim.Reset()
		}
	case stateFalling, stateJumping, stateDoubleJumping:
		if p.animIdle != nil {
			p.anim = p.animIdle
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
	// Zero out vertical velocity first so the impulse cleanly reverses
	// downward motion and feels weighty, then apply a vertical impulse.
	if p.body != nil {
		v := p.body.Velocity()
		// zero Y
		p.body.SetVelocity(v.X, 0)
		// apply vertical impulse (use existing jumpImpulse constant)
		p.body.ApplyImpulseAtLocalPoint(cp.Vector{X: 0, Y: jumpImpulse}, cp.Vector{})
		// small horizontal nudge based on input direction
		if p.Input != nil && p.Input.MoveX != 0 {
			p.body.ApplyImpulseAtLocalPoint(cp.Vector{X: doubleJumpHorizImpulse * float64(p.Input.MoveX), Y: 0}, cp.Vector{})
		}
		p.VelocityY = float32(jumpImpulse)
	} else {
		// non-physics path: zero then set velocity to impulse
		p.VelocityY = 0
		p.VelocityY = float32(jumpImpulse)
	}
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
	// stop the wall grab if the player is moving away from the wall or not pressing any horizontal input
	if p.Input.MoveX == 0 || (p.Input.MoveX < 0 && w.wallS == WALL_RIGHT) || (p.Input.MoveX > 0 && w.wallS == WALL_LEFT) {
		p.setState(stateFalling)
		return
	}

	if p.Input.JumpPressed {
		// horizontal push-off impulse
		if w.wallS == WALL_LEFT {
			p.body.ApplyImpulseAtLocalPoint(cp.Vector{X: 4, Y: 0}, cp.Vector{})
		} else if w.wallS == WALL_RIGHT {
			p.body.ApplyImpulseAtLocalPoint(cp.Vector{X: -4, Y: 0}, cp.Vector{})
		}
		p.setState(stateJumping)
		return
	}
}
func (w *wallGrabState) OnPhysics(p *Player) {
	// while grabbing, clamp vertical movement; if using physics body adjust body velocity
	if float64(w.elapsed) < ebiten.ActualTPS()/2 {
		v := p.body.Velocity()
		p.body.SetVelocity(v.X, 0)
		p.VelocityY = 0
	} else {
		v := p.body.Velocity()
		// gently slide down
		p.body.SetVelocity(v.X, v.Y+0.025*float64(1/ebiten.ActualTPS()))
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

	// hide OS cursor while aiming; we'll draw our own target sprite
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	// initialize aim reticle from last remembered angle (if available)
	if p.Input != nil && p.Input.LastAimValid {
		aimRadius := 100000.0
		if p.CollisionWorld != nil && p.CollisionWorld.level != nil {
			lw := float64(p.CollisionWorld.level.Width * common.TileSize)
			lh := float64(p.CollisionWorld.level.Height * common.TileSize)
			maxDim := math.Max(lw, lh)
			aimRadius = maxDim * 2.0
		}
		cx := float64(p.X + float32(p.Width)/2.0)
		cy := float64(p.Y + float32(p.Height)/2.0)
		nx := math.Cos(p.Input.LastAimAngle)
		ny := math.Sin(p.Input.LastAimAngle)
		p.Input.MouseWorldX = cx + nx*aimRadius
		p.Input.MouseWorldY = cy + ny*aimRadius
	}

	if !p.CollisionWorld.IsGrounded(p.Rect) {
		p.PhysicsTimeScale = 0.05
		p.PhysicsSlowTimer = int(2.0 * ebiten.ActualTPS())
	}
}
func (aimingState) Exit(p *Player) {
	p.PhysicsTimeScale = 1.0
}
func (aimingState) HandleInput(p *Player) {
	// Left click to attach to a physics tile
	if p.Input.MouseLeftPressed {
		mx := p.Input.MouseWorldX
		my := p.Input.MouseWorldY
		if p.Anchor != nil {
			ax, ay, hit := p.AimCollisionPoint(mx, my)
			if hit {
				if p.Anchor.Attach(ax, ay) {
					p.setState(stateIdle)
					return
				}
			}
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
		if p.Anchor != nil {
			p.Anchor.Detach()
			if p.body != nil {
				p.body.SetAngle(0)
				p.body.SetAngularVelocity(0)
			}
		}
		p.setState(stateFalling)
		return
	}
}
func (swingingState) OnPhysics(p *Player) {
	p.applyMoveXForce()
}

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
	stateDashing       playerState = &dashingState{}
)

type dashingState struct{}

func (dashingState) Name() string { return "dashing" }
func (dashingState) Enter(p *Player) {
	p.DashTimer = dashFrames
	// set velocity to dash speed immediately
	if p.body != nil {
		v := p.body.Velocity()
		p.body.SetVelocity(p.DashDir*dashSpeed, v.Y)
		p.VelocityX = float32(p.DashDir * dashSpeed)
	} else {
		p.VelocityX = float32(p.DashDir * dashSpeed)
	}
}
func (dashingState) Exit(p *Player) {}
func (dashingState) HandleInput(p *Player) {
	// ignore player input while dashing
}
func (dashingState) OnPhysics(p *Player) {
	// maintain dash horizontal velocity
	if p.body != nil {
		v := p.body.Velocity()
		p.body.SetVelocity(p.DashDir*dashSpeed, v.Y)
		p.VelocityX = float32(p.DashDir * dashSpeed)
	} else {
		p.VelocityX = float32(p.DashDir * dashSpeed)
	}

	// transition when dash timer expires (timer is decremented in Update)
	if p.DashTimer <= 0 {
		if p.CollisionWorld != nil && p.CollisionWorld.IsGrounded(p.Rect) {
			if p.Input.MoveX != 0 {
				p.setState(stateRunning)
			} else {
				p.setState(stateIdle)
			}
		} else {
			p.setState(stateFalling)
		}
	}
}

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
	// ColliderOffset shifts the physics body relative to the sprite/rect.
	ColliderOffsetX float32
	ColliderOffsetY float32

	// SpriteOffset shifts the drawn sprite relative to the collision AABB/body.
	SpriteOffsetX float32
	SpriteOffsetY float32

	// Aiming / swing anchor (moved to its own type and file)
	Anchor *Anchor

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

	// PhysicsTimeScale scales how much time physics advances for this player
	// (1.0 = normal). Used for slow-to-crawl effect when aiming in-air.
	PhysicsTimeScale float64
	// PhysicsSlowTimer counts down frames while slow is active
	PhysicsSlowTimer int
	// DashTimer counts frames remaining while dash is active
	DashTimer int
	// DashDir is the horizontal direction of the current dash (-1 or +1)
	DashDir float64
}

// ApplyTransitionJumpImpulse applies the standard jump impulse to the player and
// transitions the player into the jumping state. This is intended for use
// when the player is teleported into a new level and should immediately hop.
func (p *Player) ApplyTransitionJumpImpulse() {
	if p == nil {
		return
	}
	if p.body != nil {

		xForce := 1.5
		if !p.facingRight {
			xForce = -xForce
		}

		p.body.ApplyImpulseAtLocalPoint(cp.Vector{X: xForce, Y: jumpImpulse}, cp.Vector{})
		// sync velocities
		v := p.body.Velocity()
		p.VelocityY = float32(v.Y)
	} else {
		p.VelocityY = float32(jumpImpulse)
		p.setState(stateJumping)
	}
}

func (p *Player) IsFacingRight() bool {
	return p.facingRight
}

func NewPlayer(
	x, y float32,
	input *Input,
	collisionWorld *CollisionWorld,
	anchor *Anchor,
	facingRight bool,
) *Player {
	p := &Player{
		Rect: common.Rect{
			X:      x,
			Y:      y,
			Width:  16,
			Height: 40,
		},
		StartX:          x,
		StartY:          y,
		GravityEnabled:  true,
		Input:           input,
		CollisionWorld:  collisionWorld,
		state:           stateIdle,
		facingRight:     facingRight,
		ColliderOffsetX: 0,
		ColliderOffsetY: 0,
		SpriteOffsetX:   0,
		SpriteOffsetY:   -8,
		Anchor:          anchor,
	}
	p.PhysicsTimeScale = 1.0
	p.state.Enter(p)
	p.img = ebiten.NewImage(int(p.Width), int(p.Height))
	p.img.Fill(colornames.Crimson)
	// default render size matches the sprite frame size to avoid scaling artifacts
	// Temporarily use the source frame size (64) to check whether scaling
	// is the source of the artifact. If this removes the artifact we will
	// investigate better downscaling strategies.
	p.RenderWidth = 64
	p.RenderHeight = 64

	p.animIdle = component.NewAnimationRow(assets.PlayerV2Sheet, 64, 64, 0, 8, 12, true)
	p.animRun = component.NewAnimationRow(assets.PlayerV2Sheet, 64, 64, 1, 4, 12, true)
	p.animWallGrab = component.NewAnimationRow(assets.PlayerV2Sheet, 64, 64, 2, 1, 12, false)

	// pre-scaling removed: frames will be scaled at draw-time
	p.anim = p.animIdle
	if p.CollisionWorld != nil {
		p.CollisionWorld.AttachPlayer(p)
	}

	return p
}

func (p *Player) Update() {
	p.frames++

	// advance anchor extension animation if any
	if p.Anchor != nil {
		p.Anchor.Update()
	}

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
		} else if p.Anchor == nil || !p.Anchor.Active {
			p.setState(stateAiming)
		} else if p.Anchor != nil && p.Anchor.Active {
			p.Anchor.Detach()
			if p.body != nil {
				p.body.SetAngle(0)
				p.body.SetAngularVelocity(0)
			}
		}
	}

	// Dash: LeftShift / gamepad X triggers a short burst. Do not dash while aiming.
	if p.Input.DashPressed {
		if p.state != stateAiming {
			dir := 1.0
			if p.Input.MoveX != 0 {
				dir = float64(p.Input.MoveX)
			} else if !p.facingRight {
				dir = -1.0
			}
			p.DashDir = dir
			p.setState(stateDashing)
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

	// decrement physics slow timer and restore normal time scale when expired
	if p.PhysicsSlowTimer > 0 {
		p.PhysicsSlowTimer--
		if p.PhysicsSlowTimer <= 0 {
			p.PhysicsTimeScale = 1.0
		}
	}

	// decrement dash timer
	if p.DashTimer > 0 {
		p.DashTimer--
	}

}

// AimCollisionPoint samples along a ray from the player's center toward the
// provided mouse world coords and returns the first world point that lies on a
// physics tile. If no tile is hit, returns the original mouse coords and
// hit==false.
func (p *Player) AimCollisionPoint(mx, my float64) (float64, float64, bool) {
	if p == nil {
		return mx, my, false
	}
	cxWorld := float64(p.X + float32(p.Width)/2.0)
	cyWorld := float64(p.Y + float32(p.Height)/2.0)
	dx := mx - cxWorld
	dy := my - cyWorld
	distToMouse := math.Hypot(dx, dy)
	if distToMouse == 0 {
		return mx, my, false
	}
	nx := dx / distToMouse
	ny := dy / distToMouse
	if p.CollisionWorld == nil || p.CollisionWorld.level == nil {
		return mx, my, false
	}
	step := 4.0
	for s := step; s <= distToMouse; s += step {
		px := cxWorld + nx*s
		py := cyWorld + ny*s
		tileX := int(math.Floor(px / float64(common.TileSize)))
		tileY := int(math.Floor(py / float64(common.TileSize)))
		if p.CollisionWorld.level.physicsTileAt(tileX, tileY) {
			return px, py, true
		}
	}
	return mx, my, false
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
		// apply braking when no horizontal input and NOT anchored (don't kill swing momentum)
		if p.Input != nil && p.Input.MoveX == 0 && p.state != stateFalling {
			v := p.body.Velocity()
			brake := -v.X * brakeForceFactor
			p.body.ApplyForceAtLocalPoint(cp.Vector{X: brake, Y: 0}, cp.Vector{})
		}
		// adjust rope length while grounded and NOT falling (lock once falling)
		if p.Anchor != nil && p.Anchor.Active && p.Input != nil && p.state != stateFalling && p.CollisionWorld.IsGrounded(p.Rect) {
			if p.Anchor.Joint != nil {
				if sj, ok := p.Anchor.Joint.Class.(*cp.SlideJoint); ok {
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
		if p.Anchor != nil && p.Anchor.Active && p.state == stateFalling && p.Anchor.Joint != nil {
			if _, ok := p.Anchor.Joint.Class.(*cp.SlideJoint); ok {
				// compute anchors for pin joint so its Dist equals current distance
				// anchor on body: use body's local point corresponding to its world position
				anchorA := p.body.WorldToLocal(p.body.Position())
				anchorB := p.CollisionWorld.space.StaticBody.WorldToLocal(p.Anchor.Pos)
				// remove slide joint
				p.CollisionWorld.space.RemoveConstraint(p.Anchor.Joint)
				// clear angular velocity to avoid large impulses on swap
				p.body.SetAngularVelocity(0)
				// create pin joint that locks the current distance
				newJoint := cp.NewPinJoint(p.body, p.CollisionWorld.space.StaticBody, anchorA, anchorB)
				// limit max force to avoid explosive impulses
				newJoint.SetMaxForce(1000)
				p.CollisionWorld.space.AddConstraint(newJoint)
				p.Anchor.Joint = newJoint
			}
		}

		if p.CollisionWorld.IsGrounded(p.Rect) && p.Anchor != nil && p.Anchor.Active && p.Anchor.Joint != nil {
			p.CollisionWorld.space.RemoveConstraint(p.Anchor.Joint)
			// recreate joint immediately without animation when grounded
			p.Anchor.CreateJointAt(p.Anchor.Pos.X, p.Anchor.Pos.Y)
		}

		// physics-driven integration: the physics step is performed centrally
		v := p.body.Velocity()
		maxX := maxSpeedX
		if p.DashTimer > 0 {
			maxX = dashSpeed
		}
		if p.Anchor != nil && p.Anchor.Active && p.state == stateFalling {
			maxX = maxSwingSpeedX
		}

		clampedX := clampFloat64(v.X, -maxX, maxX)
		clampedY := clampFloat64(v.Y, -maxSpeedY, maxSpeedY)
		if clampedX != v.X || clampedY != v.Y {
			p.body.SetVelocity(clampedX, clampedY)
			v.X = clampedX
			v.Y = clampedY
		}

		p.VelocityX = float32(v.X)
		p.VelocityY = float32(v.Y)

		// keep body rotation locked: also lock while anchored so the
		// player doesn't spin during swinging
		p.body.SetAngle(0)
		p.body.SetAngularVelocity(0)

		pos := p.body.Position()
		p.Rect.X = float32(pos.X - float64(p.Width)/2.0 - float64(p.ColliderOffsetX))
		p.Rect.Y = float32(pos.Y - float64(p.Height)/2.0 - float64(p.ColliderOffsetY))
		if math.IsNaN(float64(p.Rect.X)) || math.IsNaN(float64(p.Rect.Y)) || math.IsInf(float64(p.Rect.X), 0) || math.IsInf(float64(p.Rect.Y), 0) {
			p.resetToSpawn()
		}

		return
	}

	p.VelocityX = float32(clampFloat64(float64(p.VelocityX), -maxSpeedX, maxSpeedX))
	p.VelocityY = float32(clampFloat64(float64(p.VelocityY), -maxSpeedY, maxSpeedY))
	p.X += p.VelocityX
	p.Y += p.VelocityY
}

func (p *Player) checkCollisions() {
	if p.CollisionWorld != nil && p.CollisionWorld.HitTriangle() {
		p.resetToSpawn()
		return
	}
	if p.Y > float32(p.CollisionWorld.level.Height*common.TileSize)-p.Height {
		p.resetToSpawn()
		return
	}

	clamped := false
	if p.X < 0 {
		p.X = 0
		p.VelocityX = 0
		clamped = true
	}

	if p.X+float32(p.Width) > float32(p.CollisionWorld.level.Width*common.TileSize) {
		p.X = float32(p.CollisionWorld.level.Width*common.TileSize) - float32(p.Width)
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
	p.body.SetPosition(cp.Vector{X: float64(p.X + p.Width/2 + p.ColliderOffsetX), Y: float64(p.Y + p.Height/2 + p.ColliderOffsetY)})
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
// Anchor logic moved to obj/anchor.go (Anchor.Attach/Detach)

func (p *Player) applyMoveXForce() {
	if p.Input.MoveX == 0 {
		return
	}

	force := moveForce
	if p.state == stateSwinging {
		force = swingMoveForce
	}
	fx := float64(p.Input.MoveX) * force
	p.body.ApplyForceAtLocalPoint(cp.Vector{X: fx, Y: 0}, cp.Vector{})
}

func clampFloat64(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (p *Player) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if p.Anchor != nil {
		p.Anchor.Draw(screen, camX, camY, zoom)
	}

	// center sprite within the collision AABB when render and collision sizes differ
	// apply SpriteOffset so the sprite is drawn offset relative to the physics body
	offsetX := (float64(p.RenderWidth) - float64(p.Width)) / 2.0
	offsetY := (float64(p.RenderHeight) - float64(p.Height)) / 2.0
	if zoom <= 0 {
		zoom = 1
	}
	// Add SpriteOffset so that a positive SpriteOffsetX/Y moves the sprite to
	// the right/down relative to the collision AABB/body.
	drawX := float64(p.X) - offsetX - camX + float64(p.SpriteOffsetX)
	drawY := float64(p.Y) - offsetY - camY + float64(p.SpriteOffsetY)

	if p.anim != nil {
		// If animation has been pre-scaled to RenderWidth/Height, draw it directly
		// without additional scaling. Otherwise, scale at draw-time.
		op := &ebiten.DrawImageOptions{}
		fw, fh := p.anim.Size()
		// scale at draw-time
		sx := float64(p.RenderWidth) / float64(fw)
		sy := float64(p.RenderHeight) / float64(fh)
		if p.facingRight {
			op.GeoM.Scale(sx*zoom, sy*zoom)
			tx := math.Round(drawX * sx * zoom)
			ty := math.Round(drawY * sy * zoom)
			op.GeoM.Translate(tx, ty)
		} else {
			op.GeoM.Scale(-sx*zoom, sy*zoom)
			tx := math.Round((drawX + float64(fw)) * sx * zoom)
			ty := math.Round(drawY * sy * zoom)
			op.GeoM.Translate(tx, ty)
		}
		op.Filter = ebiten.FilterNearest
		p.anim.Draw(screen, op)
	} else {
		op := &ebiten.DrawImageOptions{}
		if p.facingRight {
			op.GeoM.Scale(zoom, zoom)
			op.GeoM.Translate(math.Round(drawX*zoom), math.Round(drawY*zoom))
		} else {
			op.GeoM.Scale(-zoom, zoom)
			op.GeoM.Translate(math.Round((drawX+float64(p.RenderWidth))*zoom), math.Round(drawY*zoom))
		}
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(p.img, op)
	}

	// draw aiming ray when in aiming state: from player center toward mouse,
	// stop if it hits a physics tile.
	if p.state == stateAiming && p.Input != nil {
		cxWorld := float64(p.X + float32(p.Width)/2.0)
		cyWorld := float64(p.Y + float32(p.Height)/2.0)
		tx := p.Input.MouseWorldX
		ty := p.Input.MouseWorldY
		dx := tx - cxWorld
		dy := ty - cyWorld
		distToMouse := math.Hypot(dx, dy)
		endX := tx
		endY := ty

		// compute unit direction; if zero, skip
		if distToMouse > 0 {
			nx := dx / distToMouse
			ny := dy / distToMouse

			// limit sampling to the mouse position (do not extend past it)
			maxDist := distToMouse

			// sample along the ray until hitting a physics tile or reaching the mouse
			if p.CollisionWorld != nil && p.CollisionWorld.level != nil {
				step := 4.0
				for s := step; s <= maxDist; s += step {
					px := cxWorld + nx*s
					py := cyWorld + ny*s
					tileX := int(math.Floor(px / float64(common.TileSize)))
					tileY := int(math.Floor(py / float64(common.TileSize)))
					if p.CollisionWorld.level.physicsTileAt(tileX, tileY) {
						endX = px
						endY = py
						break
					}
				}
				// if no collision, endX/endY remain at mouse coords
			} else {
				// no collision world: end at mouse coords
				endX = tx
				endY = ty
			}
		}

		ebitenutil.DrawLine(screen, (cxWorld-camX)*zoom, (cyWorld-camY)*zoom, (endX-camX)*zoom, (endY-camY)*zoom, colornames.Red)

		// aim target is drawn in screen space by Game.Draw (not here)
	}

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("State: %s, jumpHeld: %v, doubleJumped: %v", p.state.Name(), p.Input.JumpHeld, p.doubleJumped), 0, 20)
}

// IsAiming reports whether the player is currently in the aiming state.
func (p *Player) IsAiming() bool {
	if p == nil {
		return false
	}
	return p.state == stateAiming
}
