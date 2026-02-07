package systems

import (
	"math"

	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
	"github.com/milk9111/sidescroller/obj"
)

// AISystem updates simple enemy AI behavior.
type AISystem struct {
	Level *obj.Level
}

// NewAISystem creates an AISystem.
func NewAISystem(level *obj.Level) *AISystem {
	return &AISystem{Level: level}
}

// Update advances AI for entities with AIState.
func (s *AISystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	aiSet := w.AIStates()
	trSet := w.Transforms()
	if aiSet == nil || trSet == nil {
		return
	}
	pathSet := w.Pathfindings()
	bodySet := w.PhysicsBodies()

	for _, id := range aiSet.Entities() {
		av := aiSet.Get(id)
		ai, ok := av.(*components.AIState)
		if !ok || ai == nil {
			continue
		}
		if ai.AttackCooldownTimer > 0 {
			ai.AttackCooldownTimer--
		}
		if ai.TargetEntity <= 0 {
			continue
		}
		selfTr := trSet.Get(id)
		self, ok := selfTr.(*components.Transform)
		if !ok || self == nil {
			continue
		}
		gtv := trSet.Get(ai.TargetEntity)
		gt, ok := gtv.(*components.Transform)
		if !ok || gt == nil {
			continue
		}

		cx := float64(self.X)
		cy := float64(self.Y)
		tx := float64(gt.X)
		ty := float64(gt.Y)
		if ai.Kind == components.AIKindFlyingEnemy {
			ty = float64(gt.Y - ai.AttackYOffset)
		}
		dx := tx - cx
		dy := ty - cy
		dist := math.Hypot(dx, dy)
		if ai.AggroRange > 0 && dist > ai.AggroRange {
			setBodyVelocity(bodySet, id, 0, getBodyYVelocity(bodySet, id))
			continue
		}

		if ai.Kind == components.AIKindGroundEnemy {
			s.handleGroundEnemy(w, ai, id, self, gt, pathSet, bodySet)
		} else {
			s.handleFlyingEnemy(w, ai, id, self, gt, dx, dy, dist, bodySet)
		}
	}
}

func (s *AISystem) handleGroundEnemy(w *ecs.World, ai *components.AIState, id int, self *components.Transform, target *components.Transform, pathSet *ecs.SparseSet, bodySet *ecs.SparseSet) {
	if ai == nil || self == nil || target == nil {
		return
	}
	dx := float64(target.X - self.X)
	dist := math.Abs(dx)
	if ai.AttackRange > 0 && dist <= ai.AttackRange {
		setBodyVelocity(bodySet, id, 0, getBodyYVelocity(bodySet, id))
		return
	}

	var desiredVX float32
	if pathSet != nil && s.Level != nil {
		pv := pathSet.Get(id)
		pf, _ := pv.(*components.Pathfinding)
		if pf != nil {
			desiredVX = s.followPath(pf, ai.MoveSpeed, self, target)
		} else {
			desiredVX = chaseVX(ai.MoveSpeed, dx)
		}
	} else {
		desiredVX = chaseVX(ai.MoveSpeed, dx)
	}
	setBodyVelocity(bodySet, id, desiredVX, getBodyYVelocity(bodySet, id))
	ai.FacingRight = desiredVX >= 0
}

func (s *AISystem) followPath(pf *components.Pathfinding, speed float32, self *components.Transform, target *components.Transform) float32 {
	if pf == nil || self == nil || target == nil {
		return 0
	}
	if pf.RecalcFrames <= 0 {
		pf.RecalcFrames = 12
	}
	if pf.MaxNodes <= 0 {
		pf.MaxNodes = 2000
	}
	if pf.WaypointReachDist <= 0 {
		pf.WaypointReachDist = 6
	}
	if pf.RecalcTimer > 0 {
		pf.RecalcTimer--
	}

	startX, startY := toTile(self.X), toTile(self.Y)
	goalX, goalY := toTile(target.X), toTile(target.Y)
	if pf.RecalcTimer <= 0 || pf.LastGoalX != goalX || pf.LastGoalY != goalY || len(pf.Path) == 0 {
		pf.Path = component.AStar(startX, startY, goalX, goalY, s.Level.Width, s.Level.Height, s.isBlockedTile, pf.MaxNodes)
		pf.PathIndex = 0
		pf.RecalcTimer = pf.RecalcFrames
		pf.LastGoalX = goalX
		pf.LastGoalY = goalY
	}
	if len(pf.Path) == 0 {
		return 0
	}
	if pf.PathIndex >= len(pf.Path) {
		pf.PathIndex = len(pf.Path) - 1
	}
	wp := pf.Path[pf.PathIndex]
	wpX := float32(wp.X*common.TileSize + common.TileSize/2)
	if math.Abs(float64(self.X-wpX)) <= float64(pf.WaypointReachDist) && pf.PathIndex < len(pf.Path)-1 {
		pf.PathIndex++
		wp = pf.Path[pf.PathIndex]
		wpX = float32(wp.X*common.TileSize + common.TileSize/2)
	}
	if wpX < self.X {
		return -speed
	}
	return speed
}

func (s *AISystem) handleFlyingEnemy(w *ecs.World, ai *components.AIState, id int, self *components.Transform, target *components.Transform, dx, dy, dist float64, bodySet *ecs.SparseSet) {
	if ai == nil {
		return
	}
	if ai.AttackCooldownTimer <= 0 && ai.AttackRange > 0 && dist <= ai.AttackRange {
		if ai.AttackAlignDist <= 0 || math.Abs(dx) <= float64(ai.AttackAlignDist) {
			if target != nil && self != nil {
				bx := self.X
				by := self.Y
				vx := float32(0)
				vy := float32(0)
				if dist > 0 {
					nx := dx / dist
					ny := dy / dist
					vx = float32(nx) * 5.0
					vy = float32(ny) * 5.0
				}
				SpawnBullet(w, bx, by, vx, vy, id)
				ai.AttackCooldownTimer = ai.AttackCooldown
			}
		}
	}
	if dist == 0 {
		setBodyVelocity(bodySet, id, 0, 0)
		return
	}
	nx := dx / dist
	ny := dy / dist
	vx := float32(nx) * ai.MoveSpeed
	vy := float32(ny) * ai.MoveSpeed
	setBodyVelocity(bodySet, id, vx, vy)
	ai.FacingRight = vx >= 0
}

func (s *AISystem) isBlockedTile(x, y int) bool {
	if s == nil || s.Level == nil {
		return false
	}
	if x < 0 || y < 0 || x >= s.Level.Width || y >= s.Level.Height {
		return true
	}
	idx := y*s.Level.Width + x
	if s.Level.PhysicsLayers != nil && len(s.Level.PhysicsLayers) > 0 {
		for _, ly := range s.Level.PhysicsLayers {
			if ly == nil || ly.Tiles == nil || len(ly.Tiles) != s.Level.Width*s.Level.Height {
				continue
			}
			if v := ly.Tiles[idx]; v != 0 {
				return true
			}
		}
		return false
	}
	if s.Level.Layers == nil || len(s.Level.Layers) == 0 {
		return false
	}
	for layerIdx, layer := range s.Level.Layers {
		if layer == nil || len(layer) != s.Level.Width*s.Level.Height {
			continue
		}
		if s.Level.LayerMeta == nil || layerIdx >= len(s.Level.LayerMeta) || !s.Level.LayerMeta[layerIdx].HasPhysics {
			continue
		}
		if v := layer[idx]; v != 0 {
			return true
		}
	}
	return false
}

func toTile(v float32) int {
	return int(math.Floor(float64(v) / float64(common.TileSize)))
}

func chaseVX(speed float32, dx float64) float32 {
	if dx < 0 {
		return -speed
	}
	if dx > 0 {
		return speed
	}
	return 0
}

func setBodyVelocity(bodySet *ecs.SparseSet, id int, vx, vy float32) {
	if bodySet == nil {
		return
	}
	if bv := bodySet.Get(id); bv != nil {
		if body, ok := bv.(*components.PhysicsBody); ok && body != nil && body.Body != nil {
			body.Body.SetVelocity(float64(vx), float64(vy))
		}
	}
}

func getBodyYVelocity(bodySet *ecs.SparseSet, id int) float32 {
	if bodySet == nil {
		return 0
	}
	if bv := bodySet.Get(id); bv != nil {
		if body, ok := bv.(*components.PhysicsBody); ok && body != nil && body.Body != nil {
			v := body.Body.Velocity()
			return float32(v.Y)
		}
	}
	return 0
}
