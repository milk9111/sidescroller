package system

import (
	"log"

	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/obj"
)

// ResolveCombat wires the combat resolver and processes hits for the frame.
func ResolveCombat(player *obj.Player, enemies []*obj.Enemy, camera *obj.Camera) {
	resolver := component.NewCombatResolver()
	// Add a resolver-level emitter to trigger camera shake on damage to player
	if camera != nil {
		em := component.CombatEventEmitter{}
		em.Handlers = append(em.Handlers, func(evt component.CombatEvent) {
			if evt.Type == component.EventDamageApplied && player != nil && evt.TargetID == player.ID {
				camera.StartShake(6.0, 12)
			}
		})
		resolver.Emitter = &em
	}
	// collect dealers (players + enemies)
	dealers := make([]component.DamageDealerComponent, 0)
	targets := make([]component.HurtboxComponent, 0)
	healthByOwner := make(map[int]component.HealthComponent)
	if player != nil {
		dealers = append(dealers, player)
		targets = append(targets, player)
		if player.Health() != nil {
			h := player.Health()
			healthByOwner[player.ID] = h
			// ensure camera shake handler is attached to the live Health instance
			if camera != nil {
				h.OnDamage = func(hh *component.Health, evt component.CombatEvent) {
					log.Printf("Player took damage (handler): amt=%.2f attacker=%d target=%d frame=%d", evt.Damage, evt.AttackerID, evt.TargetID, evt.Frame)
					camera.StartShake(6.0, 12)
				}
			}
		}
	}
	for _, enemy := range enemies {
		if enemy == nil {
			continue
		}
		dealers = append(dealers, enemy)
		targets = append(targets, enemy)
	}
	if len(dealers) > 0 && len(targets) > 0 {
		resolver.Tick()
		resolver.ResolveAll(dealers, targets, healthByOwner)
		if len(resolver.Recent) > 0 {
			component.AddRecentHighlights(resolver.Recent)
		}
	}
}
