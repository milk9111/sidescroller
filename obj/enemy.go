//go:build legacy
// +build legacy

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
	enemySheetRows = 3
	enemySheetCols = 12
	// enemy behavior tuning (pixels)
	enemyAggroRange        = 220.0
	enemyStopDistance      = 8.0
	enemyMoveSpeed         = 1.6
	enemyWaypointReachDist = 6.0
	enemyPathRecalcFrames  = 12
	enemyPathMaxNodes      = 2000
)

// melee attack tuning
const (
	enemyMeleeRange = 40.0
)

var enemyNextID = 1000

// enemyState is the interface each concrete enemy state implements.
type enemyState interface {
	Enter(e *Enemy)
	Exit(e *Enemy)
	HandleInput(e *Enemy, p *Player)
	OnPhysics(e *Enemy, p *Player)
	Name() string
}

type enemyIdleState struct{}

func (enemyIdleState) Name() string { return "idle" }
func (enemyIdleState) Enter(e *Enemy) {
	if e == nil {
		return
	}
	if e.body != nil {
		v := e.body.Velocity()
		e.body.SetVelocity(0, v.Y)
	} else {
		e.VelocityX = 0
	}
}
func (enemyIdleState) Exit(e *Enemy)                   {}
func (enemyIdleState) HandleInput(e *Enemy, p *Player) {}
func (enemyIdleState) OnPhysics(e *Enemy, p *Player) {
	if e == nil || p == nil {
		return
	}
	_, _, dist := e.distanceToPlayer(p)
	if dist <= enemyAggroRange {
		e.setState(stateEnemyMoving)
	}
}

type enemyMovingState struct{}

func (enemyMovingState) Name() string                    { return "moving" }
func (enemyMovingState) Enter(e *Enemy)                  {}
func (enemyMovingState) Exit(e *Enemy)                   {}
func (enemyMovingState) HandleInput(e *Enemy, p *Player) {}
func (enemyMovingState) OnPhysics(e *Enemy, p *Player) {
	if e == nil || p == nil {
		e.setState(stateEnemyIdle)
		return
	}
	_, _, dist := e.distanceToPlayer(p)
	if dist > enemyAggroRange {
		e.setState(stateEnemyIdle)
		return
	}
	// enter melee attack if close enough
	if dist <= enemyMeleeRange {
		e.AttackTarget = p
		e.setState(stateEnemyAttacking)
		return
	}
	e.ensurePathToPlayer(p)
	if !e.moveAlongPath() {
		// fallback if no path available
		e.moveTowardPlayer(p)
	}
}

// singletons for each state to avoid allocating on every transition
var (
	stateEnemyIdle      enemyState = &enemyIdleState{}
	stateEnemyMoving    enemyState = &enemyMovingState{}
	stateEnemyAttacking enemyState = &enemyAttackingState{}
)

type enemyAttackingState struct{}

func (enemyAttackingState) Name() string { return "attacking" }
func (enemyAttackingState) Enter(e *Enemy) {
	if e == nil {
		return
	}
	if e.animAttack != nil {
		e.anim = e.animAttack
		e.anim.Reset()
		// on 11th 1-based frame -> index 10 (0-based)
		frameIdx := 10
		if frameIdx < e.anim.FrameCount {
			e.anim.AddFrameCallback(frameIdx, func(a *component.Animation, frame int) {
				// schedule attack to be handled on main update loop
				e.pendingAttack = true
			})
		}
		// when animation ends, return to moving state
		endIdx := e.anim.FrameCount - 1
		if endIdx >= 0 {
			e.anim.AddFrameCallback(endIdx, func(a *component.Animation, frame int) {
				e.setState(stateEnemyMoving)
			})
		}
	}
}
func (enemyAttackingState) Exit(e *Enemy) {
	if e == nil {
		return
	}
	if e.anim != nil {
		e.anim.ClearFrameCallbacks()
	}
	e.AttackTarget = nil
}
func (enemyAttackingState) HandleInput(e *Enemy, p *Player) {}
func (enemyAttackingState) OnPhysics(e *Enemy, p *Player) {
	// stay facing toward the target while attacking
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

// Enemy is a simple animated enemy with a state-driven animation.
type Enemy struct {
	common.Rect
	ID             int
	CollisionWorld *CollisionWorld
	body           *cp.Body
	shape          *cp.Shape

	state       enemyState
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

	health        *component.Health
	hitboxes      []component.Hitbox
	hurtboxes     []component.Hurtbox
	faction       component.Faction
	combatEmitter component.CombatEventEmitter

	// Attack target while in attacking state
	AttackTarget  *Player
	pendingAttack bool
	attackTimer   int
}

// NewEnemy creates an enemy at world pixel (x,y).
func NewEnemy(x, y float32, collisionWorld *CollisionWorld) *Enemy {
	enemyNextID++
	e := &Enemy{
		Rect: common.Rect{
			X:      x,
			Y:      y,
			Width:  64,
			Height: 64,
		},
		ID:             enemyNextID,
		state:          stateEnemyIdle,
		facingRight:    true,
		CollisionWorld: collisionWorld,
		faction:        component.FactionEnemy,
	}
	e.health = component.NewHealth(3)
	e.hurtboxes = []component.Hurtbox{
		{
			ID:      "enemy_body",
			Rect:    e.Rect,
			Faction: e.faction,
			Enabled: true,
			OwnerID: e.ID,
		},
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
			e.animAttack = component.NewAnimationRow(sheet, frameW, frameH, 2, 15, 12, false)
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
		e.CollisionWorld.AttachEnemy(e)
	}
	log.Printf("NewEnemy created: enemy_ptr=%p health_ptr=%p id=%d", e, e.health, e.ID)

	return e
}

// setState switches states and resets animation.
func (e *Enemy) setState(s enemyState) {
	if e == nil || s == nil || e.state == s {
		return
	}
	if e.state != nil {
		e.state.Exit(e)
	}
	e.state = s
	switch s {
	case stateEnemyIdle:
		e.anim = e.animIdle
	case stateEnemyMoving:
		e.anim = e.animMove
	default:
		e.anim = e.animIdle
	}
	if e.anim != nil {
		e.anim.Reset()
	}
	e.state.Enter(e)
}

// Update advances the current animation and state machine.
func (e *Enemy) Update(player *Player) {
	if e == nil {
		return
	}
	if e.state == nil {
		e.state = stateEnemyIdle
		e.state.Enter(e)
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

	e.syncCombatBoxes()
	if e.health != nil {
		e.health.Tick()
	}

	// ensure attack target is cleared if dead or out-of-range
	if e.AttackTarget != nil {
		_, _, d := e.distanceToPlayer(e.AttackTarget)
		if !e.AttackTarget.Health().IsAlive() || d > enemyMeleeRange*1.5 {
			e.AttackTarget = nil
		}
	}

	// perform pending attack: create transient hitbox for a couple frames
	if e.pendingAttack {
		e.pendingAttack = false
		hbW := float32(28)
		hbH := e.Height * 0.8
		var hbX float32
		if e.facingRight {
			hbX = e.X + e.Width
		} else {
			hbX = e.X - hbW
		}
		hbY := e.Y + (e.Height-hbH)/2
		hb := component.Hitbox{
			ID:      "enemy_attack",
			Rect:    common.Rect{X: hbX, Y: hbY, Width: hbW, Height: hbH},
			Active:  true,
			OwnerID: e.ID,
			Damage: component.Damage{
				Amount:         1,
				KnockbackX:     2.0,
				KnockbackY:     -1.0,
				HitstunFrames:  6,
				CooldownFrames: 12,
				IFrameFrames:   12,
				Faction:        e.faction,
				MultiHit:       false,
			},
		}
		e.hitboxes = []component.Hitbox{hb}
		e.attackTimer = 2
	}

	if e.attackTimer > 0 {
		e.attackTimer--
		if e.attackTimer <= 0 {
			e.hitboxes = nil
		}
	}
}

func (e *Enemy) syncCombatBoxes() {
	if e == nil {
		return
	}
	if len(e.hurtboxes) == 0 {
		e.hurtboxes = []component.Hurtbox{{
			ID:      "enemy_body",
			Faction: e.faction,
			Enabled: true,
			OwnerID: e.ID,
		}}
	}
	for i := range e.hurtboxes {
		h := e.hurtboxes[i]
		h.Rect = e.Rect
		h.Faction = e.faction
		h.OwnerID = e.ID
		if !h.Enabled {
			h.Enabled = true
		}
		e.hurtboxes[i] = h
	}
	for i := range e.hitboxes {
		h := e.hitboxes[i]
		h.OwnerID = e.ID
		e.hitboxes[i] = h
	}
}

// Combat component accessors
func (e *Enemy) Hitboxes() []component.Hitbox         { return e.hitboxes }
func (e *Enemy) SetHitboxes(boxes []component.Hitbox) { e.hitboxes = boxes }
func (e *Enemy) DamageFaction() component.Faction     { return e.faction }
func (e *Enemy) EmitHit(evt component.CombatEvent)    { e.combatEmitter.Emit(evt) }

func (e *Enemy) Hurtboxes() []component.Hurtbox         { return e.hurtboxes }
func (e *Enemy) SetHurtboxes(boxes []component.Hurtbox) { e.hurtboxes = boxes }
func (e *Enemy) HurtboxFaction() component.Faction      { return e.faction }
func (e *Enemy) CanBeHit() bool {
	return e.health != nil && e.health.IsAlive()
}

func (e *Enemy) Health() *component.Health { return e.health }

// performAttack constructs a transient hitbox and resolves combat against the
// current AttackTarget. This is invoked from an animation frame callback.
func (e *Enemy) performAttack() {
	if e == nil || e.AttackTarget == nil || e.health == nil {
		return
	}
	target := e.AttackTarget
	if target == nil || !target.Health().IsAlive() {
		return
	}

	// build a forward-facing melee hitbox in front of the enemy
	hbW := float32(28)
	hbH := e.Height * 0.8
	var hbX float32
	if e.facingRight {
		hbX = e.X + e.Width
	} else {
		hbX = e.X - hbW
	}
	hbY := e.Y + (e.Height-hbH)/2

	hb := component.Hitbox{
		ID:      "enemy_attack",
		Rect:    common.Rect{X: hbX, Y: hbY, Width: hbW, Height: hbH},
		Active:  true,
		OwnerID: e.ID,
		Damage: component.Damage{
			Amount:         1,
			KnockbackX:     2.0,
			KnockbackY:     -1.0,
			HitstunFrames:  6,
			CooldownFrames: 12,
			IFrameFrames:   12,
			Faction:        e.faction,
			MultiHit:       false,
		},
	}

	// attach transient hitbox to enemy and resolve against target
	prev := e.hitboxes
	e.hitboxes = []component.Hitbox{hb}
	resolver := component.NewCombatResolver()
	resolver.Emitter = &e.combatEmitter
	// single-target resolve
	resolver.Resolve(e, target, target.Health())
	// publish any recent collisions so debug overlay can highlight them
	if len(resolver.Recent) > 0 {
		component.AddRecentHighlights(resolver.Recent)
	}
	// restore previous hitboxes
	e.hitboxes = prev
}

func (e *Enemy) distanceToPlayer(p *Player) (float64, float64, float64) {
	if e == nil || p == nil {
		return 0, 0, math.MaxFloat64
	}
	ex := float64(e.X + e.Width/2)
	ey := float64(e.Y + e.Height/2)
	px := float64(p.X + p.Width/2)
	py := float64(p.Y + p.Height/2)
	dx := px - ex
	dy := py - ey
	return dx, dy, math.Hypot(dx, dy)
}

func (e *Enemy) moveTowardPlayer(p *Player) {
	if e == nil || p == nil {
		return
	}
	dx, _, _ := e.distanceToPlayer(p)
	if math.Abs(dx) < enemyStopDistance {
		if e.body != nil {
			v := e.body.Velocity()
			e.body.SetVelocity(0, v.Y)
		} else {
			e.VelocityX = 0
		}
		return
	}
	var dir float64 = 1
	if dx < 0 {
		dir = -1
	}
	e.facingRight = dir > 0
	if e.body != nil {
		v := e.body.Velocity()
		e.body.SetVelocity(dir*enemyMoveSpeed, v.Y)
	} else {
		e.VelocityX = float32(dir * enemyMoveSpeed)
	}
}

func (e *Enemy) ensurePathToPlayer(p *Player) {
	if e == nil || p == nil || e.CollisionWorld == nil || e.CollisionWorld.level == nil {
		return
	}
	level := e.CollisionWorld.level
	ex := float64(e.X + e.Width/2)
	ey := float64(e.Y + e.Height/2)
	px := float64(p.X + p.Width/2)
	py := float64(p.Y + p.Height/2)
	startX := int(math.Floor(ex / float64(common.TileSize)))
	startY := int(math.Floor(ey / float64(common.TileSize)))
	goalX := int(math.Floor(px / float64(common.TileSize)))
	goalY := int(math.Floor(py / float64(common.TileSize)))

	if e.pathRecalcTimer > 0 {
		e.pathRecalcTimer--
	}

	startChanged := startX != e.lastPathStartX || startY != e.lastPathStartY
	goalChanged := goalX != e.lastPathGoalX || goalY != e.lastPathGoalY
	pathEmpty := len(e.path) == 0 || e.pathIndex >= len(e.path)
	if e.pathRecalcTimer > 0 && !startChanged && !goalChanged && !pathEmpty {
		return
	}

	path := component.AStar(startX, startY, goalX, goalY, level.Width, level.Height, func(x, y int) bool {
		return level.physicsTileAt(x, y)
	}, enemyPathMaxNodes)
	e.path = path
	e.pathIndex = 0
	e.lastPathStartX = startX
	e.lastPathStartY = startY
	e.lastPathGoalX = goalX
	e.lastPathGoalY = goalY
	e.pathRecalcTimer = enemyPathRecalcFrames

	// skip the starting node if it matches current tile
	if len(e.path) > 0 && e.path[0].X == startX && e.path[0].Y == startY {
		e.pathIndex = 1
	}
}

func (e *Enemy) moveAlongPath() bool {
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
	if math.Hypot(dx, dy) <= enemyWaypointReachDist {
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

	if math.Abs(dx) < enemyStopDistance {
		if e.body != nil {
			v := e.body.Velocity()
			e.body.SetVelocity(0, v.Y)
		} else {
			e.VelocityX = 0
		}
		return true
	}

	var dirX float64 = 1
	if dx < 0 {
		dirX = -1
	}
	e.facingRight = dirX > 0
	if e.body != nil {
		v := e.body.Velocity()
		e.body.SetVelocity(dirX*enemyMoveSpeed, v.Y)
	} else {
		len := math.Hypot(dx, dy)
		if len > 0 {
			e.VelocityX = float32((dx / len) * enemyMoveSpeed)
			e.VelocityY = float32((dy / len) * enemyMoveSpeed)
		}
	}
	return true
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

// DrawDebugPath renders the current path (if any) on the debug overlay.
func (e *Enemy) DrawDebugPath(screen *ebiten.Image, camX, camY, zoom float64) {
	if e == nil || screen == nil || len(e.path) == 0 {
		return
	}
	if zoom <= 0 {
		zoom = 1
	}
	lineColor := color.RGBA{R: 0x20, G: 0xff, B: 0x7a, A: 0xff}
	nodeColor := color.RGBA{R: 0xff, G: 0xb3, B: 0x3a, A: 0xff}

	// draw nodes and segments
	var lastX, lastY float64
	for i, n := range e.path {
		wx := float64(n.X*common.TileSize + common.TileSize/2)
		wy := float64(n.Y*common.TileSize + common.TileSize/2)
		sx := (wx - camX) * zoom
		sy := (wy - camY) * zoom
		// node marker
		size := 3.0 * zoom
		ebitenutil.DrawRect(screen, sx-size/2, sy-size/2, size, size, nodeColor)
		if i > 0 {
			ebitenutil.DrawLine(screen, lastX, lastY, sx, sy, lineColor)
		}
		lastX = sx
		lastY = sy
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
