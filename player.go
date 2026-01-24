package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/colornames"
)

const (
	jumpHeight            = -12
	jumpBufferTimerAmount = 10 // frames

)

type Player struct {
	Rect
	StartX, StartY float32
	VelocityX      float32
	VelocityY      float32
	Input          *Input
	CollisionWorld *CollisionWorld

	frames          int
	grounded        bool
	jumping         bool
	doubleJumping   bool
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
	}
	p.img = ebiten.NewImage(int(p.Width), int(p.Height))
	p.img.Fill(colornames.Crimson)
	return p
}

func (p *Player) Update() {
	p.frames++
	p.VelocityX = 5 * p.Input.MoveX

	// reset grounded; checkCollisions will set it if we land
	p.grounded = false

	// manage jump buffer timer
	if p.jumpBuffer {
		p.jumpBufferTimer--
		if p.jumpBufferTimer <= 0 {
			p.jumpBuffer = false
		}
	}

	// record a buffer when jump is pressed mid-air
	if !p.grounded && p.Input.Jump {
		p.jumpBuffer = true
		p.jumpBufferTimer = jumpBufferTimerAmount
	}

	p.applyPhysics()
	p.checkCollisions()

	// Apply buffered jump if we landed this frame
	if p.jumpBuffer && p.grounded {
		p.Input.Jump = true
		p.prevJump = false
		p.jumpBuffer = false
	}

	// Handle jump on press (rising edge) to support single and double jump.
	if p.Input.Jump && !p.prevJump {
		if !p.jumping {
			p.jumping = true
			p.VelocityY = jumpHeight
		} else if !p.doubleJumping {
			p.doubleJumping = true
			p.VelocityY = jumpHeight
		}
	}

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
		if tileVal == 2 {
			// collided with triangle -> reset player
			p.Rect.X = p.StartX
			p.Rect.Y = p.StartY
			p.VelocityX = 0
			p.VelocityY = 0
			p.jumping = false
			p.doubleJumping = false
			p.grounded = false
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
			p.jumping = false
			p.doubleJumping = false
			p.grounded = false
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

	if p.Y > float32(baseHeight)-p.Height {
		p.Y = float32(baseHeight) - p.Height
		p.VelocityY = 0
		p.jumping = false
		p.doubleJumping = false
		p.grounded = true
	}

	if p.CollisionWorld.IsGrounded(p.Rect) {
		p.jumping = false
		p.doubleJumping = false
		p.grounded = true
	}
}

func (p *Player) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(p.X), float64(p.Y))
	screen.DrawImage(p.img, op)
}
