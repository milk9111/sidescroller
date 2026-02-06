package system

import (
	"strings"

	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/obj"
)

func (w *World) spawnPickupsFromEntities(entities []obj.PlacedEntity, player *obj.Player) ([]*obj.Pickup, []obj.PlacedEntity) {
	if w == nil || len(entities) == 0 {
		return nil, entities
	}

	pickups := make([]*obj.Pickup, 0)
	remaining := make([]obj.PlacedEntity, 0, len(entities))
	for _, pe := range entities {
		if !isPickupEntity(pe) {
			remaining = append(remaining, pe)
			continue
		}

		x := float32(pe.X * common.TileSize)
		y := float32(pe.Y * common.TileSize)

		var pickup *obj.Pickup
		switch {
		case strings.Contains(pe.Name, "dash_pickup"):
			var p *obj.Pickup
			p = obj.NewPickup(x, y, pe.Sprite, func() {
				if player != nil {
					player.DashEnabled = true
				}
				if p != nil {
					p.Disabled = true
				}
			})
			pickup = p
		case strings.Contains(pe.Name, "anchor_pickup"):
			var p *obj.Pickup
			p = obj.NewPickup(x, y, pe.Sprite, func() {
				if player != nil {
					player.SwingEnabled = true
				}
				if p != nil {
					p.Disabled = true
				}
			})
			pickup = p
		case strings.Contains(pe.Name, "double_jump_pickup"):
			var p *obj.Pickup
			p = obj.NewPickup(x, y, pe.Sprite, func() {
				if player != nil {
					player.DoubleJumpEnabled = true
				}
				if p != nil {
					p.Disabled = true
				}
			})
			pickup = p
		default:
			remaining = append(remaining, pe)
			continue
		}

		pickups = append(pickups, pickup)
	}

	return pickups, remaining
}

func (w *World) spawnEnemiesFromEntities(entities []obj.PlacedEntity) ([]*obj.Enemy, []obj.PlacedEntity) {
	if w == nil || len(entities) == 0 {
		return nil, entities
	}

	enemies := make([]*obj.Enemy, 0)
	remaining := make([]obj.PlacedEntity, 0, len(entities))
	for _, pe := range entities {
		if !isEnemyEntity(pe) {
			remaining = append(remaining, pe)
			continue
		}

		x := float32(pe.X * common.TileSize)
		y := float32(pe.Y * common.TileSize)
		enemy := obj.NewEnemy(x, y, w.CollisionWorld)
		if enemy != nil {
			enemies = append(enemies, enemy)
		}
	}

	return enemies, remaining
}

func (w *World) spawnFlyingEnemiesFromEntities(entities []obj.PlacedEntity) ([]*obj.FlyingEnemy, []obj.PlacedEntity) {
	if w == nil || len(entities) == 0 {
		return nil, entities
	}

	flyingEnemies := make([]*obj.FlyingEnemy, 0)
	remaining := make([]obj.PlacedEntity, 0, len(entities))
	for _, pe := range entities {
		if !isFlyingEnemyEntity(pe) {
			remaining = append(remaining, pe)
			continue
		}

		x := float32(pe.X * common.TileSize)
		y := float32(pe.Y * common.TileSize)
		enemy := obj.NewFlyingEnemy(x, y, w.CollisionWorld)
		if enemy != nil {
			flyingEnemies = append(flyingEnemies, enemy)
		}
	}

	return flyingEnemies, remaining
}

func isPickupEntity(pe obj.PlacedEntity) bool {
	if strings.EqualFold(strings.TrimSpace(pe.Type), "pickup") {
		return true
	}
	return pe.Type == "" && strings.Contains(pe.Name, "pickup")
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
