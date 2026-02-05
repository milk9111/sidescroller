package obj

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
)

const (
	enemySheetRows = 3
	enemySheetCols = 12
)

// EnemyState represents the current animation/state.
type EnemyState int

const (
	EnemyStateIdle EnemyState = iota
	EnemyStateMove
	EnemyStateAttacking
)

// Enemy is a simple animated enemy with a state-driven animation.
type Enemy struct {
	common.Rect
	CollisionWorld *CollisionWorld
	body           *cp.Body
	shape          *cp.Shape

	state       EnemyState
	facingRight bool
	VelocityX   float32
	VelocityY   float32

	sheet *ebiten.Image
	anim  *component.Animation

	animIdle   *component.Animation
	animMove   *component.Animation
	animAttack *component.Animation

	RenderWidth  float32
	RenderHeight float32

	ColliderOffsetX float32
	ColliderOffsetY float32

	SpriteOffsetX float32
	SpriteOffsetY float32

	placeholder *ebiten.Image
}

// NewEnemy creates an enemy at world pixel (x,y).
func NewEnemy(x, y float32, collisionWorld *CollisionWorld) *Enemy {
	e := &Enemy{
		Rect: common.Rect{
			X:      x,
			Y:      y,
			Width:  64,
			Height: 64,
		},
		state:          EnemyStateIdle,
		facingRight:    true,
		CollisionWorld: collisionWorld,
	}

	sheet, err := assets.LoadImage("enemy-Sheet.png")
	if err == nil {
		e.sheet = sheet
		frameW, frameH := enemyFrameSize(sheet)
		if frameW > 0 && frameH > 0 {
			e.Width = float32(frameW)
			e.Height = float32(frameH)
			e.RenderWidth = float32(frameW)
			e.RenderHeight = float32(frameH)

			e.animIdle = component.NewAnimationRow(sheet, frameW, frameH, 0, 1, 12, true)
			e.animMove = component.NewAnimationRow(sheet, frameW, frameH, 1, 5, 12, true)
			e.animAttack = component.NewAnimationRow(sheet, frameW, frameH, 2, 12, 12, false)
		}
	}

	if e.RenderWidth == 0 || e.RenderHeight == 0 {
		e.RenderWidth = e.Width
		e.RenderHeight = e.Height
	}

	if e.animIdle != nil {
		e.anim = e.animIdle
	} else {
		img := ebiten.NewImage(int(e.Width), int(e.Height))
		img.Fill(color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff})
		e.placeholder = img
	}

	if e.CollisionWorld != nil {
		e.CollisionWorld.AttachEnemy(e)
	}

	return e
}

// SetState switches the enemy state and resets the animation.
func (e *Enemy) SetState(s EnemyState) {
	if e == nil || e.state == s {
		return
	}
	e.state = s
	switch s {
	case EnemyStateIdle:
		e.anim = e.animIdle
	case EnemyStateMove:
		e.anim = e.animMove
	case EnemyStateAttacking:
		e.anim = e.animAttack
	}
	if e.anim != nil {
		e.anim.Reset()
	}
}

// State returns the current state.
func (e *Enemy) State() EnemyState {
	if e == nil {
		return EnemyStateIdle
	}
	return e.state
}

// Update advances the current animation.
func (e *Enemy) Update() {
	if e == nil {
		return
	}
	if e.body != nil {
		v := e.body.Velocity()
		e.body.SetAngle(0)
		e.body.SetAngularVelocity(0)
		e.VelocityX = float32(v.X)
		e.VelocityY = float32(v.Y)
		pos := e.body.Position()
		e.Rect.X = float32(pos.X - float64(e.Width)/2.0 - float64(e.ColliderOffsetX))
		e.Rect.Y = float32(pos.Y - float64(e.Height)/2.0 - float64(e.ColliderOffsetY))
	}
	if e.anim != nil {
		e.anim.Update()
	}
}

func (e *Enemy) syncBodyFromRect() {
	if e == nil || e.body == nil {
		return
	}
	e.body.SetPosition(cp.Vector{X: float64(e.X + e.Width/2 + e.ColliderOffsetX), Y: float64(e.Y + e.Height/2 + e.ColliderOffsetY)})
	e.body.SetVelocity(float64(e.VelocityX), float64(e.VelocityY))
}

// Draw renders the enemy at camera-local coordinates.
func (e *Enemy) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if e == nil {
		return
	}
	if zoom <= 0 {
		zoom = 1
	}

	drawX := float64(e.X) - camX + float64(e.SpriteOffsetX)
	drawY := float64(e.Y) - camY + float64(e.SpriteOffsetY)

	if e.anim != nil {
		op := &ebiten.DrawImageOptions{}
		fw, fh := e.anim.Size()
		if fw <= 0 || fh <= 0 {
			return
		}
		sx := float64(e.RenderWidth) / float64(fw)
		sy := float64(e.RenderHeight) / float64(fh)
		if e.facingRight {
			op.GeoM.Scale(sx*zoom, sy*zoom)
			op.GeoM.Translate(math.Round(drawX*sx*zoom), math.Round(drawY*sy*zoom))
		} else {
			op.GeoM.Scale(-sx*zoom, sy*zoom)
			op.GeoM.Translate(math.Round((drawX+float64(fw))*sx*zoom), math.Round(drawY*sy*zoom))
		}
		op.Filter = ebiten.FilterNearest
		e.anim.Draw(screen, op)
		return
	}

	if e.placeholder != nil {
		op := &ebiten.DrawImageOptions{}
		if e.facingRight {
			op.GeoM.Scale(zoom, zoom)
			op.GeoM.Translate(math.Round(drawX*zoom), math.Round(drawY*zoom))
		} else {
			op.GeoM.Scale(-zoom, zoom)
			op.GeoM.Translate(math.Round((drawX+float64(e.Width))*zoom), math.Round(drawY*zoom))
		}
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(e.placeholder, op)
	}
}

func enemyFrameSize(sheet *ebiten.Image) (int, int) {
	if sheet == nil {
		return 0, 0
	}
	w := sheet.Bounds().Dx()
	h := sheet.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	fh := h / enemySheetRows
	if fh <= 0 {
		fh = h
	}
	// Prefer square frames when possible.
	fw := w / enemySheetCols
	if fh > 0 && w%fh == 0 {
		fw = fh
	}
	if fw <= 0 {
		fw = w
	}
	return fw, fh
}
