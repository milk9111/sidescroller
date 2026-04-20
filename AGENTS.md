## Defective (Go + Ebitengine)

Defective is a scene-driven 2D action-platformer built with Go, Ebitengine, Chipmunk2D (`cp`) for physics, and EbitenUI for UI. Gameplay and the editor both run on the in-repo ECS.

## Working Rules

- NEVER inject one system into another. Communicate through components.
- DO NOT read or modify TODO.md.
- Prefer reusing entities instead of creating new ones.
- Prefer adding behavior in systems instead of growing scene/app orchestration.
- ECS components are stored as pointers; mutating a fetched component usually does not require re-adding it.

## Runtime Shape

- App entry: [main.go](main.go) builds config, save/profiler state, and a [scenes.Manager](scenes/manager.go).
- Scenes: [scenes/scene.go](scenes/scene.go) defines `start_menu`, `intro`, `game`, and `test`.
- Game orchestration: [scenes/game_scene.go](scenes/game_scene.go) owns one ECS world and two schedulers.
- Gameplay scheduler: audio/music, player control, pathfinding/navigation, aim/animation, color/flash/shake/fade, invulnerability/combat/knockback, lever/arena/health/hazard/anchor/repulsion/platforms, physics, popups/triggers/pickups, particles, scripts, gates, ttl/respawn/transitions, persistence, child spawning, camera, parallax.
- Dialogue scheduler: music, animation, dialogue/item/inventory, debug/tutorial messages, UI.
- Draw path: render, particles, and UI are separate; physics/AI/path/trigger debug overlays are optional.

## ECS + Content

- ECS core: [ecs/world.go](ecs/world.go) and [ecs/scheduler.go](ecs/scheduler.go).
- Major libraries: Ebitengine for app/render/input, Chipmunk2D `cp` for physics state/simulation, and EbitenUI for in-game and menu UI.
- Scripts: [ecs/system/script.go](ecs/system/script.go) wraps the Tengo runtime in [ecs/script/runtime.go](ecs/script/runtime.go).
- Script data supports `Path` or `Paths`, built-in modules, per-entity runtime state, entity signal queues, and a global hit-signal queue.
- Content is data-driven from [levels/](levels/) and [prefabs/](prefabs/). Optional prefab hot reload uses [prefabs/watch.go](prefabs/watch.go) and enqueues a `ReloadRequest` component.

## Editor

- The level/prefab editor is a separate ECS app in [cmd/editor/app.go](cmd/editor/app.go).
- Its scheduler runs input, UI, commands, undo, camera, layers, areas, overview, entities, prefabs, tools, autotile, and persistence systems.

## Primary Files

- Runtime entry: [main.go](main.go)
- Scene manager: [scenes/manager.go](scenes/manager.go)
- Game scene wiring: [scenes/game_scene.go](scenes/game_scene.go)
- ECS core: [ecs/world.go](ecs/world.go), [ecs/scheduler.go](ecs/scheduler.go)
- Systems: [ecs/system/](ecs/system/)
- Scripts: [ecs/script/runtime.go](ecs/script/runtime.go), [ecs/component/script.go](ecs/component/script.go)
- Prefabs + reload: [prefabs/](prefabs/), [prefabs/watch.go](prefabs/watch.go)
- Editor: [cmd/editor/app.go](cmd/editor/app.go)
