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
		p.VelocityY = jumpHeight
		// fmt.Println("jump from idle")
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
		p.VelocityY = jumpHeight
		p.setState(stateJumping)
		return
	}
	if p.Input.MoveX == 0 {
		p.setState(stateIdle)
	}
}
func (runningState) OnPhysics(p *Player) {
	if !p.CollisionWorld.IsGrounded(p.Rect) {
		p.setState(stateFalling)
	}
}

type jumpingState struct{}

func (jumpingState) Name() string { return "jumping" }
func (jumpingState) Enter(p *Player) {
	fmt.Println("entered jumping state")
}
func (jumpingState) Exit(p *Player) {}
func (jumpingState) HandleInput(p *Player) {
	if p.Input.JumpPressed {
		if !p.doubleJumped {
			p.doubleJumped = true
			p.VelocityY = jumpHeight
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
	if p.VelocityY > 0 {
		p.setState(stateFalling)
	}
}

type doubleJumpingState struct{}

func (doubleJumpingState) Name() string { return "doublejump" }
func (doubleJumpingState) Enter(p *Player) {
	fmt.Println("entered double jumping state")
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
			p.VelocityY = jumpHeight
			p.setState(stateJumping)
			return
		}
		if !p.doubleJumped {
			p.doubleJumped = true
			p.VelocityY = jumpHeight
			p.setState(stateDoubleJumping)
			return
		}
		// already used double jump -> record buffer for next landing
		p.jumpBuffer = true
		p.jumpBufferTimer = jumpBufferTimerAmount
	}
}
func (fallingState) OnPhysics(p *Player) {
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
		p.VelocityY = jumpHeight
		if w.wallS == WALL_LEFT {
			p.VelocityX = 20 // jump right
		} else if w.wallS == WALL_RIGHT {
			p.VelocityX = -20 // jump left
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

	if float64(w.elapsed) < ebiten.ActualTPS()/2 {
		p.VelocityY = 0
	} else {
		p.VelocityY += 1.5 * float32(1/ebiten.ActualTPS())
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

// singletons for each state to avoid allocating on every transition
var (
	stateIdle          playerState = &idleState{}
	stateRunning       playerState = &runningState{}
	stateJumping       playerState = &jumpingState{}
	stateDoubleJumping playerState = &doubleJumpingState{}
	stateFalling       playerState = &fallingState{}
	stateWallGrab      playerState = &wallGrabState{}
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
	p.VelocityX = 5 * p.Input.MoveX
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
	p.state.HandleInput(p)

	if p.prevJumpHeld && !p.Input.JumpHeld {
		p.applyJumpCut()
	}
	p.prevJumpHeld = p.Input.JumpHeld

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
	p.state.OnPhysics(p)

}

func (p *Player) applyJumpCut() {
	if (p.state == stateJumping || p.state == stateDoubleJumping) && p.VelocityY < jumpCutVelocity {
		p.VelocityY = jumpCutVelocity
	}
}

func (p *Player) applyPhysics() {
	if p.GravityEnabled {
		p.VelocityY += common.Gravity
	}
	if p.CollisionWorld != nil && p.body != nil {
		p.CollisionWorld.BeginStep()
		p.body.SetVelocity(float64(p.VelocityX), float64(p.VelocityY))
		p.CollisionWorld.Step(1.0)
		v := p.body.Velocity()
		p.VelocityX = float32(v.X)
		p.VelocityY = float32(v.Y)
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
}
