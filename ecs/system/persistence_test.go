package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/internal/savegame"
	"github.com/milk9111/sidescroller/levels"
)

func TestRespawnCurrentLevelEnemiesRestoresOnlyDefeatedEnemies(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	_ = ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	stateMap := &component.LevelEntityStateMap{States: map[string]component.PersistedLevelEntityState{
		"test.json#enemy_1":  component.PersistedLevelEntityStateDefeated,
		"test.json#shrine_1": component.PersistedLevelEntityStateDefeated,
	}}
	_ = ecs.Add(w, player, component.LevelEntityStateMapComponent.Kind(), stateMap)

	runtimeEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, runtimeEntity, component.LevelRuntimeComponent.Kind(), &component.LevelRuntime{
		Name: "test.json",
		Level: &levels.Level{Entities: []levels.Entity{
			{ID: "enemy_1", Type: "flying_enemy", X: 96, Y: 64, Props: map[string]interface{}{"layer": float64(0)}},
			{ID: "shrine_1", Type: "shrine", X: 160, Y: 64, Props: map[string]interface{}{"layer": float64(0)}},
		}},
	})

	p := &PersistenceSystem{}
	if err := p.respawnCurrentLevelEnemies(w); err != nil {
		t.Fatalf("respawnCurrentLevelEnemies() error = %v", err)
	}

	enemyCount := 0
	shrineCount := 0
	ecs.ForEach(w, component.GameEntityIDComponent.Kind(), func(e ecs.Entity, id *component.GameEntityID) {
		if id == nil {
			return
		}
		switch id.Value {
		case "enemy_1":
			enemyCount++
			if !ecs.Has(w, e, component.AIComponent.Kind()) {
				t.Fatal("expected respawned enemy to include AI component")
			}
		case "shrine_1":
			shrineCount++
		}
	})

	if enemyCount != 1 {
		t.Fatalf("expected one respawned enemy, got %d", enemyCount)
	}
	if shrineCount != 0 {
		t.Fatalf("expected no non-enemy respawn, got %d", shrineCount)
	}
	if _, ok := stateMap.States["test.json#enemy_1"]; ok {
		t.Fatal("expected defeated enemy state to be cleared after respawn")
	}
	if got := stateMap.States["test.json#shrine_1"]; got != component.PersistedLevelEntityStateDefeated {
		t.Fatalf("expected non-enemy defeated state to remain, got %q", got)
	}
	if _, ok := ecs.First(w, component.LevelLoadedComponent.Kind()); ok {
		t.Fatal("expected local enemy respawn to avoid full level reload")
	}
}

func TestHasParticleEmitterNamed(t *testing.T) {
	w := ecs.NewWorld()

	other := ecs.CreateEntity(w)
	if err := ecs.Add(w, other, component.ParticleEmitterComponent.Kind(), &component.ParticleEmitter{Name: "other_emitter"}); err != nil {
		t.Fatalf("add other emitter: %v", err)
	}

	if hasParticleEmitterNamed(w, playerAttackHitEmitterName) {
		t.Fatal("expected helper to ignore unrelated emitters")
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.ParticleEmitterComponent.Kind(), &component.ParticleEmitter{Name: playerAttackHitEmitterName}); err != nil {
		t.Fatalf("add target emitter: %v", err)
	}

	if !hasParticleEmitterNamed(w, playerAttackHitEmitterName) {
		t.Fatal("expected helper to find player attack hit emitter by name")
	}
}

func TestSpawnPlayerAtLinkedTransitionFallsBackToReverseLink(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 40, AlignTopLeft: true}); err != nil {
		t.Fatalf("add player body: %v", err)
	}

	transition := ecs.CreateEntity(w)
	if err := ecs.Add(w, transition, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 800, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transition transform: %v", err)
	}
	if err := ecs.Add(w, transition, component.TransitionComponent.Kind(), &component.Transition{
		ID:       "e1",
		LinkedID: "right",
		EnterDir: component.TransitionDirRight,
		Bounds: component.AABB{
			W: 32,
			H: 128,
		},
	}); err != nil {
		t.Fatalf("add transition: %v", err)
	}

	p := &PersistenceSystem{}
	p.spawnPlayerAtLinkedTransition(w, component.LevelChangeRequest{
		SpawnTransitionID: "left",
		FromTransitionID:  "right",
	})

	transform, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok || transform == nil {
		t.Fatal("expected player transform")
	}
	if transform.X != 6 || transform.Y != 888 {
		t.Fatalf("expected player to spawn at fallback transition, got (%v,%v)", transform.X, transform.Y)
	}

	cooldown, ok := ecs.Get(w, player, component.TransitionCooldownComponent.Kind())
	if !ok || cooldown == nil {
		t.Fatal("expected transition cooldown")
	}
	if !cooldown.Active || cooldown.TransitionID != "e1" {
		t.Fatalf("expected cooldown on resolved transition e1, got active=%v id=%q", cooldown.Active, cooldown.TransitionID)
	}
	if len(cooldown.TransitionIDs) != 1 || cooldown.TransitionIDs[0] != "e1" {
		t.Fatalf("expected cooldown ids [e1], got %+v", cooldown.TransitionIDs)
	}
}

func TestSpawnPlayerAtLinkedTransitionPrefersExactID(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	_ = ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 40, AlignTopLeft: true})

	fallback := ecs.CreateEntity(w)
	_ = ecs.Add(w, fallback, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 800, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, fallback, component.TransitionComponent.Kind(), &component.Transition{ID: "e1", LinkedID: "right", EnterDir: component.TransitionDirRight, Bounds: component.AABB{W: 32, H: 128}})

	exact := ecs.CreateEntity(w)
	_ = ecs.Add(w, exact, component.TransformComponent.Kind(), &component.Transform{X: 1248, Y: 32, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, exact, component.TransitionComponent.Kind(), &component.Transition{ID: "left", LinkedID: "something_else", EnterDir: component.TransitionDirLeft, Bounds: component.AABB{W: 32, H: 128}})

	p := &PersistenceSystem{}
	p.spawnPlayerAtLinkedTransition(w, component.LevelChangeRequest{
		SpawnTransitionID: "left",
		FromTransitionID:  "right",
	})

	transform, _ := ecs.Get(w, player, component.TransformComponent.Kind())
	if transform.X != 1254 || transform.Y != 120 {
		t.Fatalf("expected exact transition spawn, got (%v,%v)", transform.X, transform.Y)
	}

	cooldown, _ := ecs.Get(w, player, component.TransitionCooldownComponent.Kind())
	if cooldown == nil || cooldown.TransitionID != "left" {
		t.Fatalf("expected cooldown id left, got %+v", cooldown)
	}
	if len(cooldown.TransitionIDs) != 1 || cooldown.TransitionIDs[0] != "left" {
		t.Fatalf("expected cooldown ids [left], got %+v", cooldown)
	}
}

func TestResolvePersistentSingletonsCopiesLevelScopedComponents(t *testing.T) {
	w := ecs.NewWorld()
	kept := ecs.CreateEntity(w)
	if err := ecs.Add(w, kept, component.PersistentComponent.Kind(), &component.Persistent{ID: "player", KeepOnLevelChange: true}); err != nil {
		t.Fatalf("add kept persistent: %v", err)
	}
	if err := ecs.Add(w, kept, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: 1}); err != nil {
		t.Fatalf("add kept layer: %v", err)
	}

	loaded := ecs.CreateEntity(w)
	if err := ecs.Add(w, loaded, component.PersistentComponent.Kind(), &component.Persistent{ID: "player", KeepOnLevelChange: true}); err != nil {
		t.Fatalf("add loaded persistent: %v", err)
	}
	if err := ecs.Add(w, loaded, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: 4}); err != nil {
		t.Fatalf("add loaded layer: %v", err)
	}
	if err := ecs.Add(w, loaded, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "e3"}); err != nil {
		t.Fatalf("add loaded game entity id: %v", err)
	}

	p := &PersistenceSystem{}
	p.resolvePersistentSingletons(w, map[string]ecs.Entity{"player": kept})

	if ecs.IsAlive(w, loaded) {
		t.Fatal("expected duplicate loaded entity to be destroyed")
	}

	layer, ok := ecs.Get(w, kept, component.EntityLayerComponent.Kind())
	if !ok || layer == nil || layer.Index != 4 {
		t.Fatalf("expected kept entity layer to update to 4, got %+v", layer)
	}

	id, ok := ecs.Get(w, kept, component.GameEntityIDComponent.Kind())
	if !ok || id == nil || id.Value != "e3" {
		t.Fatalf("expected kept entity game id e3, got %+v", id)
	}
}

func TestPruneForReloadKeepsAbilitiesSingleton(t *testing.T) {
	w := ecs.NewWorld()
	abilities := ecs.CreateEntity(w)
	if err := ecs.Add(w, abilities, component.PersistentComponent.Kind(), &component.Persistent{
		ID:                "player_abilities",
		KeepOnLevelChange: true,
		KeepOnReload:      true,
	}); err != nil {
		t.Fatalf("add abilities persistent: %v", err)
	}
	if err := ecs.Add(w, abilities, component.AbilitiesComponent.Kind(), &component.Abilities{DoubleJump: true, Anchor: true}); err != nil {
		t.Fatalf("add abilities component: %v", err)
	}

	transient := ecs.CreateEntity(w)
	if err := ecs.Add(w, transient, component.TransformComponent.Kind(), &component.Transform{X: 32, Y: 48}); err != nil {
		t.Fatalf("add transient transform: %v", err)
	}

	p := &PersistenceSystem{}
	p.pruneForReload(w, PersistenceOnReload)

	if !ecs.IsAlive(w, abilities) {
		t.Fatal("expected abilities singleton to survive reload pruning")
	}
	if ecs.IsAlive(w, transient) {
		t.Fatal("expected transient entity to be destroyed during reload pruning")
	}

	abilitiesComp, ok := ecs.Get(w, abilities, component.AbilitiesComponent.Kind())
	if !ok || abilitiesComp == nil || !abilitiesComp.DoubleJump || !abilitiesComp.Anchor {
		t.Fatalf("expected abilities to remain intact after reload pruning, got %+v", abilitiesComp)
	}
}

func TestNewPersistenceSystemPrefersLoadedSaveLevel(t *testing.T) {
	p := NewPersistenceSystem("long_fall.json", false, nil, false, nil, nil, &savegame.File{Level: "boss_room.json"})

	if p.levelName != "boss_room.json" {
		t.Fatalf("expected level name from loaded save, got %q", p.levelName)
	}
	if p.initialLevelName != "boss_room.json" {
		t.Fatalf("expected initial level name from loaded save, got %q", p.initialLevelName)
	}
}

func TestArmTransitionCooldownForCurrentOverlap(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	_ = ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 40, AlignTopLeft: true, OffsetX: 10, OffsetY: 20})

	transition := ecs.CreateEntity(w)
	_ = ecs.Add(w, transition, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, transition, component.TransitionComponent.Kind(), &component.Transition{
		ID:          "start_transition",
		TargetLevel: "other_level.json",
		LinkedID:    "linked_transition",
		Bounds:      component.AABB{W: 64, H: 64},
	})

	p := &PersistenceSystem{}
	p.armTransitionCooldownForCurrentOverlap(w)

	cooldown, ok := ecs.Get(w, player, component.TransitionCooldownComponent.Kind())
	if !ok || cooldown == nil || !cooldown.Active || cooldown.TransitionID != "start_transition" {
		t.Fatalf("expected overlap cooldown for start_transition, got %+v", cooldown)
	}
	if len(cooldown.TransitionIDs) != 1 || cooldown.TransitionIDs[0] != "start_transition" {
		t.Fatalf("expected overlap cooldown ids [start_transition], got %+v", cooldown.TransitionIDs)
	}

	NewTransitionSystem().Update(w)
	if _, ok := ecs.First(w, component.TransitionRuntimeComponent.Kind()); ok {
		t.Fatal("expected transition system to stay idle while overlap cooldown is active")
	}
}

func TestArmTransitionCooldownForCurrentOverlapTracksAllOverlaps(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	_ = ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 24, Y: 8, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 40, AlignTopLeft: true})

	left := ecs.CreateEntity(w)
	_ = ecs.Add(w, left, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, left, component.TransitionComponent.Kind(), &component.Transition{
		ID:          "left_transition",
		TargetLevel: "left.json",
		LinkedID:    "linked_left",
		Bounds:      component.AABB{W: 32, H: 96},
	})

	top := ecs.CreateEntity(w)
	_ = ecs.Add(w, top, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, top, component.TransitionComponent.Kind(), &component.Transition{
		ID:          "top_transition",
		TargetLevel: "top.json",
		LinkedID:    "linked_top",
		Bounds:      component.AABB{W: 96, H: 32},
	})

	p := &PersistenceSystem{}
	p.armTransitionCooldownForCurrentOverlap(w)

	cooldown, ok := ecs.Get(w, player, component.TransitionCooldownComponent.Kind())
	if !ok || cooldown == nil {
		t.Fatal("expected transition cooldown")
	}
	if len(cooldown.TransitionIDs) != 2 {
		t.Fatalf("expected both overlapping transitions to be tracked, got %+v", cooldown.TransitionIDs)
	}

	_ = ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 4, Y: 40, ScaleX: 1, ScaleY: 1})
	NewTransitionSystem().Update(w)
	if _, ok := ecs.First(w, component.TransitionRuntimeComponent.Kind()); ok {
		t.Fatal("expected transition system to stay idle while still inside one of the startup overlaps")
	}
}

func TestArmTransitionCooldownForCurrentOverlapIgnoresInsideTransitions(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	_ = ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 40, AlignTopLeft: true})

	inside := ecs.CreateEntity(w)
	_ = ecs.Add(w, inside, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, inside, component.TransitionComponent.Kind(), &component.Transition{
		ID:          "inside_transition",
		TargetLevel: "inside.json",
		LinkedID:    "door_a",
		Type:        component.TransitionTypeInside,
		Bounds:      component.AABB{W: 64, H: 64},
	})

	p := &PersistenceSystem{}
	p.armTransitionCooldownForCurrentOverlap(w)

	if cooldown, ok := ecs.Get(w, player, component.TransitionCooldownComponent.Kind()); ok && cooldown != nil && cooldown.Active {
		t.Fatalf("expected inside transitions to be ignored by startup cooldown, got %+v", cooldown)
	}
}

func TestApplyTransitionPopUsesUpSpawnTransitions(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	_ = ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(w, player, component.PlayerComponent.Kind(), &component.Player{MoveSpeed: 90, JumpSpeed: 110})
	_ = ecs.Add(w, player, component.SpriteComponent.Kind(), &component.Sprite{})

	transition := ecs.CreateEntity(w)
	_ = ecs.Add(w, transition, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, transition, component.TransitionComponent.Kind(), &component.Transition{ID: "bottom", EnterDir: component.TransitionDirUp})

	p := &PersistenceSystem{}
	p.applyTransitionPop(w, component.LevelChangeRequest{
		SpawnTransitionID: "bottom",
		EntryFromBelow:    true,
		FromFacingLeft:    true,
	})

	pop, ok := ecs.Get(w, player, component.TransitionPopComponent.Kind())
	if !ok || pop == nil {
		t.Fatal("expected transition pop for up enter direction")
	}
	if pop.VY >= 0 {
		t.Fatalf("expected upward pop, got VY=%v", pop.VY)
	}
	if !pop.FacingLeft {
		t.Fatal("expected pop to preserve source facing")
	}
}
