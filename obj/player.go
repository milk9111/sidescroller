package obj

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
	"golang.org/x/image/colornames"
)

// playerState is the interface each concrete player state implements.
type playerState interface {
	Enter(p *Player)
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
}

// singletons for each state to avoid allocating on every transition
var (
	stateIdle          playerState = &idleState{}
	stateRunning       playerState = &runningState{}
	stateJumping       playerState = &jumpingState{}
	stateDoubleJumping playerState = &doubleJumpingState{}
	stateFalling       playerState = &fallingState{}
)

type Player struct {
	common.Rect
	StartX, StartY float32
	VelocityX      float32
	VelocityY      float32
	Input          *Input
	CollisionWorld *CollisionWorld

	frames          int
	state           playerState
	doubleJumped    bool
	jumpBuffer      bool
	jumpBufferTimer int
	coyoteTimer     int
	prevJumpHeld    bool
	img             *ebiten.Image
	anim            *component.Animation
	animIdle        *component.Animation
	animRun         *component.Animation
	facingRight     bool
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
	// pre-scaling removed: frames will be scaled at draw-time
	p.anim = p.animIdle

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
	p.VelocityY += 0.5 // gravity

	p.X += p.VelocityX
	p.Y += p.VelocityY
}

func (p *Player) checkCollisions() {
	// Use the rect position before movement when querying collisions.
	// applyPhysics already modified p.Rect, so compute the rects before movement.
	preX := p.Rect
	preX.X -= p.VelocityX
	if resolved, hit, tileVal := p.CollisionWorld.MoveX(preX, p.VelocityX); hit {
		if tileVal == 2 || p.Y > float32(common.BaseHeight)-p.Height {
			// collided with triangle or fell out of world -> reset player
			p.Rect.X = p.StartX
			p.Rect.Y = p.StartY
			p.VelocityX = 0
			p.VelocityY = 0
			p.setState(stateIdle)
			p.doubleJumped = false
			if p.anim != nil {
				p.anim.Reset()
			}
			return
		}
		p.Rect.X = resolved.X
		p.VelocityX = 0
	}

	preY := p.Rect
	preY.Y -= p.VelocityY
	if resolved, hit, tileVal := p.CollisionWorld.MoveY(preY, p.VelocityY); hit {
		if tileVal == 2 {
			// collided with triangle -> reset player
			p.Rect.X = p.StartX
			p.Rect.Y = p.StartY
			p.VelocityX = 0
			p.VelocityY = 0
			p.setState(stateIdle)
			p.doubleJumped = false
			return
		}
		p.Rect.Y = resolved.Y
		p.VelocityY = 0
	}

	if p.X < 0 {
		p.X = 0
		p.VelocityX = 0
	}

	if p.X+float32(p.Width) > float32(common.BaseWidth) {
		p.X = float32(common.BaseWidth) - float32(p.Width)
		p.VelocityX = 0
	}

	if p.Y < 0 {
		p.Y = 0
		p.VelocityY = 0
	}
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
