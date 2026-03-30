package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func collectPickupEntity(w *ecs.World, e ecs.Entity, pickup *component.Pickup) {
	if w == nil || pickup == nil || !e.Valid() || !ecs.IsAlive(w, e) {
		return
	}

	if abilitiesEntity, found := ecs.First(w, component.AbilitiesComponent.Kind()); found {
		if abilities, ok := ecs.Get(w, abilitiesEntity, component.AbilitiesComponent.Kind()); ok && abilities != nil {
			if pickup.GrantDoubleJump {
				abilities.DoubleJump = true
			}
			if pickup.GrantWallGrab {
				abilities.WallGrab = true
			}
			if pickup.GrantAnchor {
				abilities.Anchor = true
				showAnchorTutorialHint(w)
			}
			_ = ecs.Add(w, abilitiesEntity, component.AbilitiesComponent.Kind(), abilities)
		}
	} else {
		ent := ecs.CreateEntity(w)
		_ = ecs.Add(w, ent, component.AbilitiesComponent.Kind(), &component.Abilities{
			DoubleJump: pickup.GrantDoubleJump,
			WallGrab:   pickup.GrantWallGrab,
			Anchor:     pickup.GrantAnchor,
		})
		if pickup.GrantAnchor {
			showAnchorTutorialHint(w)
		}
	}

	if pickup.Kind == "gear" {
		if gears := ensurePlayerGearCount(w); gears != nil {
			gears.Count++
		}
	}

	recordLevelEntityState(w, e, component.PersistedLevelEntityStateCollected)
	EmitEntitySignal(w, e, e, "on_pickup_collected")

	_ = ecs.Remove(w, e, component.ItemComponent.Kind())
	_ = ecs.Remove(w, e, component.PickupComponent.Kind())
	_ = ecs.Remove(w, e, component.SpriteComponent.Kind())
	_ = ecs.Add(w, e, component.TTLComponent.Kind(), &component.TTL{Frames: 2})
}
