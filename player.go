package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
	if p.Input.Jump && !p.prevJump {
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
	if p.Input.Jump && !p.prevJump {
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
	if p.Input.Jump && !p.prevJump {
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
	if p.Input.Jump && !p.prevJump {
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
	if p.Input.Jump && !p.prevJump {
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
	Rect
	StartX, StartY float32
	VelocityX      float32
	VelocityY      float32
	Input          *Input
	CollisionWorld *CollisionWorld

	frames          int
	state           playerState
	doubleJumped    bool
	prevJump        bool
	jumpBuffer      bool
	jumpBufferTimer int
	coyoteTimer     int
	img             *ebiten.Image
	anim            *Animation
	animIdle        *Animation
	animRun         *Animation
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
		Rect: Rect{
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
	// default render size matches the collision AABB; can be changed independently
	p.RenderWidth = 64
	p.RenderHeight = 64
	if PlayerSheet != nil {
		p.animIdle = NewAnimationRow(PlayerSheet, 128, 128, 0, 9, 12, true)
		p.animRun = NewAnimationRow(PlayerSheet, 128, 128, 1, 7, 12, true)
		p.anim = p.animIdle
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
		p.Input.Jump = true
		p.prevJump = false
		p.jumpBuffer = false
		// re-handle input now that we're grounded
		p.state.HandleInput(p)
	}

	// Let the state react to physics (velocity, grounded)
	p.state.OnPhysics(p)

	p.prevJump = p.Input.Jump
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
		if tileVal == 2 || p.Y > float32(baseHeight)-p.Height {
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

	if p.X+float32(p.Width) > float32(baseWidth) {
		p.X = float32(baseWidth) - float32(p.Width)
		p.VelocityX = 0
	}

	if p.Y < 0 {
		p.Y = 0
		p.VelocityY = 0
	}

	// if p.Y > float32(baseHeight)-p.Height {
	// 	p.Y = float32(baseHeight) - p.Height
	// 	p.VelocityY = 0
	// 	p.setState(stateIdle)
	// 	// fmt.Println("hit the ground")
	// 	p.doubleJumped = false
	// }

	// if p.CollisionWorld.IsGrounded(p.Rect) {p
	// 	p.setState(stateIdle)
	// 	// fmt.Println("landed")
	// 	p.doubleJumped = false
	// }
}

func (p *Player) Draw(screen *ebiten.Image) {
	// center sprite within the collision AABB when render and collision sizes differ
	offsetX := (float64(p.RenderWidth) - float64(p.Width)) / 2.0
	offsetY := (float64(p.RenderHeight) - float64(p.Height)) / 2.0
	drawX := float64(p.X) - offsetX
	drawY := float64(p.Y) - offsetY

	if p.anim != nil {
		// scale frame to render size and flip when facing left
		op := &ebiten.DrawImageOptions{}
		fw, fh := p.anim.Size()
		sx := float64(p.RenderWidth) / float64(fw)
		sy := float64(p.RenderHeight) / float64(fh)
		if p.facingRight {
			op.GeoM.Scale(sx, sy)
			op.GeoM.Translate(drawX, drawY)
		} else {
			op.GeoM.Scale(-sx, sy)
			// when flipped horizontally, translate by frame width * scale to align
			op.GeoM.Translate(drawX+float64(fw)*sx, drawY)
		}
		p.anim.Draw(screen, 0, 0, op)
	} else {
		op := &ebiten.DrawImageOptions{}
		if p.facingRight {
			op.GeoM.Translate(drawX, drawY)
		} else {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(drawX+float64(p.RenderWidth), drawY)
		}
		screen.DrawImage(p.img, op)
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("State: %s, jumped: %g, doubleJumped: %g", p.state.Name(), p.Input.Jump, p.doubleJumped), 0, 20)
}
