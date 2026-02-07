package systems

import (
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
)

// CombatSystem resolves hitboxes against hurtboxes and applies damage.
type CombatSystem struct {
	Resolver *component.CombatResolver
	Events   *[]component.CombatEvent
	Emitter  *component.CombatEventEmitter
}

// NewCombatSystem creates a CombatSystem.
func NewCombatSystem(events *[]component.CombatEvent) *CombatSystem {
	return &CombatSystem{
		Resolver: component.NewCombatResolver(),
		Events:   events,
	}
}

// Update resolves combat for all dealers and targets.
func (s *CombatSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	if s.Resolver == nil {
		s.Resolver = component.NewCombatResolver()
	}

	if s.Events != nil {
		*s.Events = (*s.Events)[:0]
	}

	if s.Resolver.Emitter == nil {
		s.Resolver.Emitter = &component.CombatEventEmitter{}
	}
	// Ensure emitter routes events to the shared slice and optional external emitter.
	baseEmitter := s.Resolver.Emitter
	baseEmitter.Handlers = []component.CombatEventHandler{func(evt component.CombatEvent) {
		if s.Events != nil {
			*s.Events = append(*s.Events, evt)
		}
		if s.Emitter != nil {
			s.Emitter.Emit(evt)
		}
	}}

	dealersSet := w.DamageDealers()
	hurtSet := w.Hurtboxes()
	healthSet := w.Healths()
	if dealersSet == nil || hurtSet == nil || healthSet == nil {
		return
	}

	dealers := make([]component.DamageDealerComponent, 0, len(dealersSet.Entities()))
	for _, id := range dealersSet.Entities() {
		if dv := dealersSet.Get(id); dv != nil {
			if d, ok := dv.(*components.DamageDealer); ok && d != nil {
				// ensure owner IDs are set
				for i := range d.Boxes {
					if d.Boxes[i].OwnerID == 0 {
						d.Boxes[i].OwnerID = id
					}
				}
				dealers = append(dealers, d)
			}
		}
	}

	targets := make([]component.HurtboxComponent, 0, len(hurtSet.Entities()))
	for _, id := range hurtSet.Entities() {
		if hv := hurtSet.Get(id); hv != nil {
			if h, ok := hv.(*components.HurtboxSet); ok && h != nil {
				if !h.Enabled {
					h.Enabled = true
				}
				for i := range h.Boxes {
					if h.Boxes[i].OwnerID == 0 {
						h.Boxes[i].OwnerID = id
					}
				}
				targets = append(targets, h)
			}
		}
	}

	if len(dealers) == 0 || len(targets) == 0 {
		return
	}

	healthByOwner := make(map[int]component.HealthComponent, len(healthSet.Entities()))
	for _, id := range healthSet.Entities() {
		if hv := healthSet.Get(id); hv != nil {
			if h, ok := hv.(*components.Health); ok && h != nil {
				healthByOwner[id] = h
			}
		}
	}

	s.Resolver.Tick()
	s.Resolver.ResolveAll(dealers, targets, healthByOwner)
	if len(s.Resolver.Recent) > 0 {
		component.AddRecentHighlights(s.Resolver.Recent)
	}
}
