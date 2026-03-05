## Defective (Go + Ebitengine)

Defective is a 2D metroidvania/action-platformer built with Go and Ebitengine. The runtime is ECS-driven and data-first: levels and prefab specs define most content, while systems handle player control, AI, combat, traversal, transitions, and rendering.

## Requirements

- NEVER inject one system into another. Systems communicate through components.
- DO NOT read or modify TODO.md.
- Prefer re-using entities instead of creating new ones.
- Prefer decomposing logic into systems (with callbacks into game orchestration when needed) instead of growing game.go.
- Component values are stored as pointers so there after modifying one there is no need to re-add it to the entity.

## Current Architecture

- **ECS core:** Lightweight world/query model in [ecs/](ecs/), scheduled by [ecs/scheduler.go](ecs/scheduler.go).
- **Orchestration:** [game.go](game.go) wires update order, debug toggles, hit-freeze, and prefab reload signaling.
- **System pipeline (high level):** input/audio/music → player + AI + pathfinding/navigation + phases/cooldowns → aim/animation/combat/knockback/invulnerability/hit-freeze → hazards/anchor/physics → pickups/scripts/ttl/respawn/transitions/persistence/spawning/hierarchy → camera.
- **Scripting runtime:** Tengo-based entity scripting with lifecycle hooks (`on_start`, `on_update`), per-entity runtime state, and builtin modules.
- **Signal bus:** Entity-scoped script signals are queued as components and dispatched each frame (used by systems like combat and pickups).
- **Data-driven content:** levels from [levels/](levels/) and prefab entities/specs/scripts from [prefabs/](prefabs/), with optional hot reload.
- **Physics/rendering:** Chipmunk-based physics and camera-aware layered rendering.

## Current Gameplay Scope

- Player movement/state machine with advanced platforming behaviors and unlockable abilities.
- Directional aiming and combat using hitbox/hurtbox interactions.
- Enemy and boss behaviors via FSM/scripted AI, plus phase and navigation systems.
- Script-driven interactions/events via signals (for example hit/pickup-triggered reactions).
- Damage feedback stack: knockback, invulnerability windows, white flash, and hit-freeze.
- World interaction systems: hazards, gates, arena nodes, boss arenas, pickups, respawn, and transitions.
- Progress/UI systems: persistent state across level loads, player health bar, trophy tracking/counter.
- Audio stack for SFX and music, plus optional runtime debug overlays.

## Key Code Pointers

- Runtime orchestration: [game.go](game.go)
- ECS foundation: [ecs/world.go](ecs/world.go), [ecs/scheduler.go](ecs/scheduler.go)
- Player logic: [ecs/system/player_controller.go](ecs/system/player_controller.go), [ecs/system/player_states.go](ecs/system/player_states.go)
- AI stack: [ecs/system/ai_controller.go](ecs/system/ai_controller.go), [ecs/system/ai_nav.go](ecs/system/ai_nav.go), [ecs/system/ai_phase.go](ecs/system/ai_phase.go)
- Scripting + signals: [ecs/system/script.go](ecs/system/script.go), [ecs/script/runtime.go](ecs/script/runtime.go), [ecs/component/script.go](ecs/component/script.go)
- Combat/damage: [ecs/system/combat.go](ecs/system/combat.go), [ecs/system/damage_knockback.go](ecs/system/damage_knockback.go), [ecs/system/invulnerability.go](ecs/system/invulnerability.go)
- World flow: [ecs/system/transition.go](ecs/system/transition.go), [ecs/system/transition_pop.go](ecs/system/transition_pop.go), [ecs/system/persistence.go](ecs/system/persistence.go)
- Encounters + interactions: [ecs/system/boss_arena.go](ecs/system/boss_arena.go), [ecs/system/arena_node.go](ecs/system/arena_node.go), [ecs/system/pickup_collect.go](ecs/system/pickup_collect.go)
- Physics/render/camera: [ecs/system/physics.go](ecs/system/physics.go), [ecs/system/render.go](ecs/system/render.go), [ecs/system/camera.go](ecs/system/camera.go)
