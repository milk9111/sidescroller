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
)

// setState helper switches states and calls Enter.
func (p *Player) setState(s playerState) {
	p.state = s
	p.state.Enter(p)
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
		if !p.doubleJumped {
			p.doubleJumped = true
			p.VelocityY = jumpHeight
			p.setState(stateDoubleJumping)
			// fmt.Println("double jump from falling")
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
	img             *ebiten.Image
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
			Height: 32,
		},
		StartX:         x,
		StartY:         y,
		Input:          input,
		CollisionWorld: collisionWorld,
		state:          stateIdle,
	}
	p.state.Enter(p)
	p.img = ebiten.NewImage(int(p.Width), int(p.Height))
	p.img.Fill(colornames.Crimson)
	return p
}

func (p *Player) Update() {
	p.frames++
	p.VelocityX = 5 * p.Input.MoveX
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

	// if p.CollisionWorld.IsGrounded(p.Rect) {
	// 	p.setState(stateIdle)
	// 	// fmt.Println("landed")
	// 	p.doubleJumped = false
	// }
}

func (p *Player) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(p.X), float64(p.Y))
	screen.DrawImage(p.img, op)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("State: %s, jumped: %g, doubleJumped: %g", p.state.Name(), p.Input.Jump, p.doubleJumped), 0, 20)
}
