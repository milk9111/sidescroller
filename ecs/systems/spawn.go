package systems

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
	"github.com/milk9111/sidescroller/obj"
)

const (
	enemySheetRows       = 3
	enemySheetCols       = 12
	flyingEnemySheetRows = 3
	flyingEnemySheetCols = 16
)

// SpawnSystem creates ECS entities from level data.
type SpawnSystem struct {
	Level        *obj.Level
	TargetEntity ecs.Entity
	Spawned      bool
}

// NewSpawnSystem creates a SpawnSystem.
func NewSpawnSystem(level *obj.Level, target ecs.Entity) *SpawnSystem {
	return &SpawnSystem{Level: level, TargetEntity: target}
}

// Update spawns entities once.
func (s *SpawnSystem) Update(w *ecs.World) {
	if w == nil || s == nil || s.Level == nil || s.Spawned {
		return
	}
	SpawnFromLevel(w, s.Level, s.TargetEntity)
	s.Spawned = true
}

// SpawnFromLevel spawns ECS enemies and removes them from level entities.
func SpawnFromLevel(w *ecs.World, level *obj.Level, target ecs.Entity) {
	if w == nil || level == nil || len(level.Entities) == 0 {
		return
	}
	remaining := make([]obj.PlacedEntity, 0, len(level.Entities))
	for _, pe := range level.Entities {
		if isPickupEntity(pe) {
			spawnPickup(w, pe)
			continue
		}
		if isEnemyEntity(pe) {
			spawnGroundEnemy(w, level, pe, target)
			continue
		}
		if isFlyingEnemyEntity(pe) {
			spawnFlyingEnemy(w, level, pe, target)
			continue
		}
		remaining = append(remaining, pe)
	}
	level.Entities = remaining
}

func spawnPickup(w *ecs.World, pe obj.PlacedEntity) {
	if w == nil {
		return
	}
	x := float32(pe.X * common.TileSize)
	y := float32(pe.Y * common.TileSize)

	kind := ""
	switch {
	case strings.Contains(pe.Name, "dash_pickup"):
		kind = "dash"
	case strings.Contains(pe.Name, "anchor_pickup"):
		kind = "anchor"
	case strings.Contains(pe.Name, "double_jump_pickup"):
		kind = "double_jump"
	default:
		kind = ""
	}
	if kind == "" {
		return
	}

	e := w.CreateEntity()
	w.SetTransform(e, &components.Transform{X: x, Y: y})
	w.SetSprite(e, &components.Sprite{ImageKey: pe.Sprite, Width: float32(common.TileSize), Height: float32(common.TileSize), Layer: 2})
	w.SetPickup(e, &components.Pickup{
		Kind:      kind,
		Enabled:   true,
		BaseX:     x,
		BaseY:     y,
		Phase:     float64(int(x)%7) * 0.3,
		Amplitude: 4.0,
		Frequency: 2.0,
		Width:     float32(common.TileSize),
		Height:    float32(common.TileSize),
	})
}

func spawnGroundEnemy(w *ecs.World, level *obj.Level, pe obj.PlacedEntity, target ecs.Entity) {
	if w == nil {
		return
	}
	x := float32(pe.X * common.TileSize)
	y := float32(pe.Y * common.TileSize)

	sheet := loadImage("enemy-Sheet.png")
	frameW, frameH := enemyFrameSize(sheet)
	if frameW <= 0 || frameH <= 0 {
		frameW, frameH = 64, 64
	}
	animIdle := component.NewAnimationRow(sheet, frameW, frameH, 0, 1, 12, true)
	animMove := component.NewAnimationRow(sheet, frameW, frameH, 1, 5, 12, true)

	e := w.CreateEntity()
	w.SetTransform(e, &components.Transform{X: x, Y: y})
	w.SetCollider(e, &components.Collider{Width: float32(frameW), Height: float32(frameH), IsEnemy: true})
	w.SetGroundSensor(e, &components.GroundSensor{})
	w.SetSprite(e, &components.Sprite{Width: float32(frameW), Height: float32(frameH), Layer: 1})
	w.SetAnimator(e, &components.Animator{Anim: animMove, Playing: true})

	w.SetAIState(e, &components.AIState{
		Kind:            components.AIKindGroundEnemy,
		TargetEntity:    target.ID,
		AggroRange:      220,
		AttackRange:     40,
		MoveSpeed:       1.6,
		AttackYOffset:   0,
		AttackAlignDist: 0,
	})
	w.SetPathfinding(e, &components.Pathfinding{RecalcFrames: 12, MaxNodes: 2000, WaypointReachDist: 6})

	if animIdle != nil {
		_ = animIdle
	}

	h := &components.Health{Current: 3, Max: 3}
	w.SetHealth(e, h)

	hurt := &components.HurtboxSet{
		Enabled: true,
		Faction: component.FactionEnemy,
		Boxes: []component.Hurtbox{{
			ID:      "enemy_body",
			OwnerID: e.ID,
			Rect:    common.Rect{X: x, Y: y, Width: float32(frameW), Height: float32(frameH)},
			Faction: component.FactionEnemy,
			Enabled: true,
		}},
	}
	w.SetHurtbox(e, hurt)
}

func spawnFlyingEnemy(w *ecs.World, level *obj.Level, pe obj.PlacedEntity, target ecs.Entity) {
	if w == nil {
		return
	}
	x := float32(pe.X * common.TileSize)
	y := float32(pe.Y * common.TileSize)

	sheet := loadImage("flying_enemy-Sheet.png")
	frameW, frameH := flyingEnemyFrameSize(sheet)
	if frameW <= 0 || frameH <= 0 {
		frameW, frameH = 64, 64
	}
	animMove := component.NewAnimationRow(sheet, frameW, frameH, 1, 2, 12, true)

	e := w.CreateEntity()
	w.SetTransform(e, &components.Transform{X: x, Y: y})
	w.SetCollider(e, &components.Collider{Width: float32(frameW), Height: float32(frameH), IsEnemy: true})
	w.SetSprite(e, &components.Sprite{Width: float32(frameW), Height: float32(frameH), Layer: 1})
	w.SetAnimator(e, &components.Animator{Anim: animMove, Playing: true})

	w.SetAIState(e, &components.AIState{
		Kind:            components.AIKindFlyingEnemy,
		TargetEntity:    target.ID,
		AggroRange:      660,
		AttackRange:     180,
		MoveSpeed:       2.0,
		AttackYOffset:   128,
		AttackAlignDist: 30,
		AttackCooldown:  300,
	})

	h := &components.Health{Current: 3, Max: 3}
	w.SetHealth(e, h)

	hurt := &components.HurtboxSet{
		Enabled: true,
		Faction: component.FactionEnemy,
		Boxes: []component.Hurtbox{{
			ID:      "flying_enemy_body",
			OwnerID: e.ID,
			Rect:    common.Rect{X: x, Y: y, Width: float32(frameW), Height: float32(frameH)},
			Faction: component.FactionEnemy,
			Enabled: true,
		}},
	}
	w.SetHurtbox(e, hurt)
}

func loadImage(path string) *ebiten.Image {
	if img, err := assets.LoadImage(path); err == nil {
		return img
	}
	return nil
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
	fw := w / enemySheetCols
	if fh > 0 && w%fh == 0 {
		fw = fh
	}
	if fw <= 0 {
		fw = w
	}
	return fw, fh
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

func isEnemyEntity(pe obj.PlacedEntity) bool {
	if strings.EqualFold(strings.TrimSpace(pe.Type), "enemy") {
		return true
	}
	return pe.Type == "" && strings.Contains(pe.Name, "enemy") && !strings.Contains(pe.Name, "flying_enemy")
}

func isFlyingEnemyEntity(pe obj.PlacedEntity) bool {
	if strings.EqualFold(strings.TrimSpace(pe.Type), "flying_enemy") {
		return true
	}
	return pe.Type == "" && strings.Contains(pe.Name, "flying_enemy")
}

func isPickupEntity(pe obj.PlacedEntity) bool {
	if strings.EqualFold(strings.TrimSpace(pe.Type), "pickup") {
		return true
	}
	return pe.Type == "" && strings.Contains(pe.Name, "pickup")
}
