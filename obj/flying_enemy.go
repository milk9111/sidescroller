package obj

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
)

const (
	flyingEnemySheetRows = 3
	flyingEnemySheetCols = 16
	// flying enemy behavior tuning (pixels)
	flyingEnemyAggroRange        = 660.0
	flyingEnemyAttackRange       = 180.0
	flyingEnemyAttackYOffset     = 128.0
	flyingEnemyAttackAlignDist   = 30.0
	flyingEnemyAttackCooldown    = 300
	flyingEnemyMoveSpeed         = 2.0
	flyingEnemyWaypointReachDist = 8.0
	flyingEnemyPathRecalcFrames  = 12
	flyingEnemyPathMaxNodes      = 2000
)

var flyingEnemyNextID = 20000

type flyingEnemyState interface {
	Enter(e *FlyingEnemy)
	Exit(e *FlyingEnemy)
	HandleInput(e *FlyingEnemy, p *Player)
	OnPhysics(e *FlyingEnemy, p *Player)
	Name() string
}

type flyingEnemyIdleState struct{}

type flyingEnemyMovingState struct{}

type flyingEnemyAttackingState struct{}

func (flyingEnemyIdleState) Name() string { return "idle" }
func (flyingEnemyIdleState) Enter(e *FlyingEnemy) {
	log.Print("flying enemy: entering idle state")
	if e == nil {
		return
	}
	e.stopMovement()
}
func (flyingEnemyIdleState) Exit(e *FlyingEnemy)                   {}
func (flyingEnemyIdleState) HandleInput(e *FlyingEnemy, p *Player) {}
func (flyingEnemyIdleState) OnPhysics(e *FlyingEnemy, p *Player) {
	if e == nil || p == nil {
		return
	}
	_, _, dist := e.distanceToPlayer(p)
	if math.Abs(flyingEnemyAttackYOffset-dist) > 100 && dist <= flyingEnemyAggroRange {
		e.setState(stateFlyingEnemyMoving)
	}

	if e.isAlignedAbovePlayer(p) && e.attackCooldownTimer <= 0 {
		e.AttackTarget = p
		e.attackCooldownTimer = flyingEnemyAttackCooldown
		e.stopMovement()
		e.setState(stateFlyingEnemyAttacking)
		return
	}
}

func (flyingEnemyMovingState) Name() string { return "moving" }
func (flyingEnemyMovingState) Enter(e *FlyingEnemy) {
	log.Print("flying enemy: entering moving state")
}
func (flyingEnemyMovingState) Exit(e *FlyingEnemy) {}
func (flyingEnemyMovingState) HandleInput(e *FlyingEnemy, p *Player) {
}
func (flyingEnemyMovingState) OnPhysics(e *FlyingEnemy, p *Player) {
	if e == nil || p == nil {
		e.setState(stateFlyingEnemyIdle)
		return
	}
	_, _, dist := e.distanceToPlayer(p)
	if dist > flyingEnemyAggroRange {
		e.setState(stateFlyingEnemyIdle)
		return
	}
	if e.isAlignedAbovePlayer(p) {
		if e.attackCooldownTimer <= 0 {
			e.AttackTarget = p
			e.attackCooldownTimer = flyingEnemyAttackCooldown
			e.stopMovement()
			e.setState(stateFlyingEnemyAttacking)
			return
		}
		e.setState(stateFlyingEnemyIdle)
		e.stopMovement()
		return
	}

	targetX, targetY := e.targetAbovePlayer(p)
	e.ensurePathToTarget(targetX, targetY)
	if !e.moveAlongPath() {
		e.moveTowardTarget(targetX, targetY)
	}
}

func (flyingEnemyAttackingState) Name() string { return "attacking" }
func (flyingEnemyAttackingState) Enter(e *FlyingEnemy) {
	log.Print("flying enemy: entering attacking state")

	if e == nil {
		return
	}
	if e.animAttack != nil {
		e.anim = e.animAttack
		e.anim.Reset()
		endIdx := e.anim.FrameCount - 1
		if endIdx >= 0 {
			e.anim.AddFrameCallback(endIdx, func(a *component.Animation, frame int) {
				e.setState(stateFlyingEnemyIdle)
			})
		}
	} else {
		e.setState(stateFlyingEnemyIdle)
	}
}
func (flyingEnemyAttackingState) Exit(e *FlyingEnemy) {
	if e == nil {
		return
	}
	if e.anim != nil {
		e.anim.ClearFrameCallbacks()
	}
	e.AttackTarget = nil
}
func (flyingEnemyAttackingState) HandleInput(e *FlyingEnemy, p *Player) {}
func (flyingEnemyAttackingState) OnPhysics(e *FlyingEnemy, p *Player) {
	if e == nil || p == nil {
		return
	}
	if e.AttackTarget != nil {
		if e.X < e.AttackTarget.X {
			e.facingRight = true
		} else {
			e.facingRight = false
		}
	}
}

var (
	stateFlyingEnemyIdle      flyingEnemyState = &flyingEnemyIdleState{}
	stateFlyingEnemyMoving    flyingEnemyState = &flyingEnemyMovingState{}
	stateFlyingEnemyAttacking flyingEnemyState = &flyingEnemyAttackingState{}
)

type FlyingEnemy struct {
	common.Rect
	ID             int
	CollisionWorld *CollisionWorld
	body           *cp.Body
	shape          *cp.Shape

	state       flyingEnemyState
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

	path            []component.PathNode
	pathIndex       int
	pathRecalcTimer int
	lastPathStartX  int
	lastPathStartY  int
	lastPathGoalX   int
	lastPathGoalY   int

	AttackTarget *Player

	attackCooldownTimer int
}

func NewFlyingEnemy(x, y float32, collisionWorld *CollisionWorld) *FlyingEnemy {
	flyingEnemyNextID++
	e := &FlyingEnemy{
		Rect: common.Rect{
			X:      x,
			Y:      y,
			Width:  64,
			Height: 64,
		},
		ID:             flyingEnemyNextID,
		state:          stateFlyingEnemyIdle,
		facingRight:    true,
		CollisionWorld: collisionWorld,
	}

	sheet, err := assets.LoadImage("flying_enemy-Sheet.png")
	if err == nil {
		e.sheet = sheet
		frameW, frameH := flyingEnemyFrameSize(sheet)
		if frameW > 0 && frameH > 0 {
			e.Width = float32(frameW)
			e.Height = float32(frameH)
			e.RenderWidth = float32(frameW)
			e.RenderHeight = float32(frameH)

			e.animIdle = component.NewAnimationRow(sheet, frameW, frameH, 0, 16, 12, true)
			e.animMove = component.NewAnimationRow(sheet, frameW, frameH, 1, 2, 12, true)
			e.animAttack = component.NewAnimationRow(sheet, frameW, frameH, 2, 7, 12, false)
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

	if e.state != nil {
		e.state.Enter(e)
	}

	if e.CollisionWorld != nil {
		e.attachToWorld(e.CollisionWorld)
	}

	return e
}

func (e *FlyingEnemy) setState(s flyingEnemyState) {
	if e == nil || s == nil || e.state == s {
		return
	}
	if e.state != nil {
		e.state.Exit(e)
	}
	e.state = s
	switch s {
	case stateFlyingEnemyIdle:
		e.anim = e.animIdle
	case stateFlyingEnemyMoving:
		e.anim = e.animMove
	case stateFlyingEnemyAttacking:
		e.anim = e.animAttack
	default:
		e.anim = e.animIdle
	}
	if e.anim != nil {
		e.anim.Reset()
	}
	e.state.Enter(e)
}

func (e *FlyingEnemy) attachToWorld(cw *CollisionWorld) {
	if e == nil || cw == nil || cw.space == nil {
		return
	}
	if e.body != nil {
		return
	}

	mass := 1.0
	moment := cp.MomentForBox(mass, float64(e.Width), float64(e.Height))
	body := cp.NewBody(mass, moment)
	body.SetAngle(0)
	body.SetAngularVelocity(0)
	body.SetPosition(cp.Vector{X: float64(e.X + e.Width/2 + e.ColliderOffsetX), Y: float64(e.Y + e.Height/2 + e.ColliderOffsetY)})
	body.SetVelocityUpdateFunc(func(body *cp.Body, gravity cp.Vector, damping float64, dt float64) {
		cp.BodyUpdateVelocity(body, cp.Vector{}, damping, dt)
	})
	shape := cp.NewBox(body, float64(e.Width), float64(e.Height), 0)
	shape.SetFriction(0.8)
	shape.SetCollisionType(collisionTypeEnemy)

	cw.space.AddBody(body)
	cw.space.AddShape(shape)

	e.body = body
	e.shape = shape
	e.CollisionWorld = cw
}

func (e *FlyingEnemy) Update(player *Player) {
	if e == nil {
		return
	}
	if e.state == nil {
		e.state = stateFlyingEnemyIdle
		e.state.Enter(e)
	}

	if e.attackCooldownTimer > 0 {
		e.attackCooldownTimer--
	}

	e.state.HandleInput(e, player)
	e.state.OnPhysics(e, player)

	if e.body != nil {
		v := e.body.Velocity()
		e.body.SetAngle(0)
		e.body.SetAngularVelocity(0)
		e.VelocityX = float32(v.X)
		e.VelocityY = float32(v.Y)
		pos := e.body.Position()
		e.Rect.X = float32(pos.X - float64(e.Width)/2.0 - float64(e.ColliderOffsetX))
		e.Rect.Y = float32(pos.Y - float64(e.Height)/2.0 - float64(e.ColliderOffsetY))
	} else {
		e.X += e.VelocityX
		e.Y += e.VelocityY
	}

	if e.anim != nil {
		e.anim.Update()
	}

	if e.AttackTarget != nil {
		_, _, d := e.distanceToPlayer(e.AttackTarget)
		if d > flyingEnemyAttackRange*1.5 {
			e.AttackTarget = nil
		}
	}
}

func (e *FlyingEnemy) distanceToPlayer(p *Player) (float64, float64, float64) {
	if e == nil || p == nil {
		return 0, 0, 0
	}
	ex := float64(e.X + e.Width/2)
	ey := float64(e.Y + e.Height/2)
	px := float64(p.X + p.Width/2)
	py := float64(p.Y + p.Height/2)
	dx := px - ex
	dy := py - ey
	return dx, dy, math.Hypot(dx, dy)
}

func (e *FlyingEnemy) targetAbovePlayer(p *Player) (float32, float32) {
	if e == nil || p == nil {
		return 0, 0
	}
	targetX := p.X + p.Width/2.0 - e.Width/2.0
	targetY := p.Y - float32(flyingEnemyAttackYOffset)
	return targetX, targetY
}

func (e *FlyingEnemy) isAlignedAbovePlayer(p *Player) bool {
	if e == nil || p == nil {
		return false
	}
	fx := e.X + e.Width/2.0
	fy := e.Y + e.Height/2.0
	px := p.X + p.Width/2.0
	py := p.Y - float32(flyingEnemyAttackYOffset)
	dx := math.Abs(float64(fx - px))
	dy := math.Abs(float64(fy - py))
	return dx <= flyingEnemyAttackAlignDist && dy <= flyingEnemyAttackAlignDist
}

func (e *FlyingEnemy) moveTowardTarget(tx, ty float32) {
	if e == nil {
		return
	}
	ex := float64(e.X + e.Width/2)
	ey := float64(e.Y + e.Height/2)
	dx := float64(tx) - ex
	dy := float64(ty) - ey
	if dx == 0 && dy == 0 {
		return
	}
	len := math.Hypot(dx, dy)
	if len == 0 {
		return
	}
	e.facingRight = dx > 0
	vx := (dx / len) * flyingEnemyMoveSpeed
	vy := (dy / len) * flyingEnemyMoveSpeed
	if e.body != nil {
		e.body.SetVelocity(vx, vy)
	} else {
		e.VelocityX = float32(vx)
		e.VelocityY = float32(vy)
	}
}

func (e *FlyingEnemy) ensurePathToTarget(tx, ty float32) {
	if e == nil || e.CollisionWorld == nil || e.CollisionWorld.level == nil {
		return
	}
	level := e.CollisionWorld.level
	ex := float64(e.X + e.Width/2)
	ey := float64(e.Y + e.Height/2)
	px := float64(tx + e.Width/2)
	py := float64(ty + e.Height/2)
	startX := int(math.Floor(ex / float64(common.TileSize)))
	startY := int(math.Floor(ey / float64(common.TileSize)))
	goalX := e.stableGoalIndex(px, e.lastPathGoalX, level.Width)
	goalY := e.stableGoalIndex(py, e.lastPathGoalY, level.Height)

	if e.pathRecalcTimer > 0 {
		e.pathRecalcTimer--
	}

	startChanged := startX != e.lastPathStartX || startY != e.lastPathStartY
	goalChanged := goalX != e.lastPathGoalX || goalY != e.lastPathGoalY
	pathEmpty := len(e.path) == 0 || e.pathIndex >= len(e.path)
	if e.pathRecalcTimer > 0 && !startChanged && !goalChanged && !pathEmpty {
		return
	}

	path := component.AStar(startX, startY, goalX, goalY, level.Width, level.Height, level.physicsTileAt, flyingEnemyPathMaxNodes)
	e.path = path
	e.pathIndex = 0
	e.lastPathStartX = startX
	e.lastPathStartY = startY
	e.lastPathGoalX = goalX
	e.lastPathGoalY = goalY
	e.pathRecalcTimer = flyingEnemyPathRecalcFrames

	if len(e.path) > 0 && e.path[0].X == startX && e.path[0].Y == startY {
		e.pathIndex = 1
	}
}

func (e *FlyingEnemy) stableGoalIndex(pos float64, last int, max int) int {
	if max <= 0 {
		return 0
	}
	tileSize := float64(common.TileSize)
	if last >= 0 && last < max {
		center := (float64(last) + 0.5) * tileSize
		if math.Abs(pos-center) < tileSize*0.6 {
			return last
		}
	}
	idx := int(math.Floor(pos / tileSize))
	return e.clampIndex(idx, max)
}

func (e *FlyingEnemy) clampIndex(idx int, max int) int {
	if idx < 0 {
		return 0
	}
	if idx >= max {
		return max - 1
	}
	return idx
}

func (e *FlyingEnemy) stopMovement() {
	if e == nil {
		return
	}
	if e.body != nil {
		e.body.SetVelocity(0, 0)
	} else {
		e.VelocityX = 0
		e.VelocityY = 0
	}
}

func (e *FlyingEnemy) moveAlongPath() bool {
	if e == nil || e.pathIndex < 0 || e.pathIndex >= len(e.path) {
		return false
	}
	n := e.path[e.pathIndex]
	tx := float64(n.X*common.TileSize + common.TileSize/2)
	ty := float64(n.Y*common.TileSize + common.TileSize/2)
	ex := float64(e.X + e.Width/2)
	ey := float64(e.Y + e.Height/2)
	dx := tx - ex
	dy := ty - ey
	if math.Hypot(dx, dy) <= flyingEnemyWaypointReachDist {
		e.pathIndex++
		if e.pathIndex >= len(e.path) {
			return false
		}
		n = e.path[e.pathIndex]
		tx = float64(n.X*common.TileSize + common.TileSize/2)
		ty = float64(n.Y*common.TileSize + common.TileSize/2)
		dx = tx - ex
		dy = ty - ey
	}

	len := math.Hypot(dx, dy)
	if len == 0 {
		return true
	}
	e.facingRight = dx > 0
	vx := (dx / len) * flyingEnemyMoveSpeed
	vy := (dy / len) * flyingEnemyMoveSpeed
	if e.body != nil {
		e.body.SetVelocity(vx, vy)
	} else {
		e.VelocityX = float32(vx)
		e.VelocityY = float32(vy)
	}
	return true
}

func (e *FlyingEnemy) syncBodyFromRect() {
	if e == nil || e.body == nil {
		return
	}
	e.body.SetPosition(cp.Vector{X: float64(e.X + e.Width/2 + e.ColliderOffsetX), Y: float64(e.Y + e.Height/2 + e.ColliderOffsetY)})
	e.body.SetVelocity(float64(e.VelocityX), float64(e.VelocityY))
}

func (e *FlyingEnemy) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
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

func (e *FlyingEnemy) DrawDebugPath(screen *ebiten.Image, camX, camY, zoom float64) {
	if e == nil || screen == nil || len(e.path) == 0 {
		return
	}
	if zoom <= 0 {
		zoom = 1
	}

	lineColor := color.RGBA{R: 0x24, G: 0xe7, B: 0xff, A: 0xff}
	nodeColor := color.RGBA{R: 0xff, G: 0xaa, B: 0x00, A: 0xff}
	lastX := 0.0
	lastY := 0.0
	for i, n := range e.path {
		wx := float64(n.X*common.TileSize + common.TileSize/2)
		wy := float64(n.Y*common.TileSize + common.TileSize/2)
		sx := (wx - camX) * zoom
		sy := (wy - camY) * zoom
		size := 3.0 * zoom
		ebitenutil.DrawRect(screen, sx-size/2, sy-size/2, size, size, nodeColor)
		if i > 0 {
			ebitenutil.DrawLine(screen, lastX, lastY, sx, sy, lineColor)
		}
		lastX = sx
		lastY = sy
	}
}

func flyingEnemyFrameSize(sheet *ebiten.Image) (int, int) {
	if sheet == nil {
		return 0, 0
	}
	w := sheet.Bounds().Dx()
	h := sheet.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	fh := h / flyingEnemySheetRows
	if fh <= 0 {
		fh = h
	}
	fw := w / flyingEnemySheetCols
	if fh > 0 && w%fh == 0 {
		fw = fh
	}
	if fw <= 0 {
		fw = w
	}
	return fw, fh
}
