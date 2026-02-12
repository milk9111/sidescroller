## Sidescroller (Go + Ebitengine)

Sidescroller is a 2D platformer built with Go and Ebitengine. It features a controllable player with responsive movement, aiming, and physics-driven interactions across tiled levels authored in a custom editor. The game is designed around data-driven content and a small ECS core to keep iteration fast and behaviors easy to extend.

## REQUIREMENTS 
- NEVER inject one system into another. The systems should be independent from each other and speak through components.
- DO NOT read or modify the TODO.md file.

## Architecture

- **ECS core:** The game is structured around a lightweight ECS in [ecs/](ecs/) with a component store and query helpers; see the `World` implementation in [ecs/world.go](ecs/world.go#L1-L192).
- **Systems pipeline:** The scheduler runs systems in a strict order (input → player controller/state machine → aiming → animation → physics → camera), configured in [game.go](game.go#L29-L48) and executed in [ecs/scheduler.go](ecs/scheduler.go#L11-L27).
- **Input:** Keyboard and gamepad input are normalized into a single input component, including aim state and right-stick vectors; see [ecs/system/input.go](ecs/system/input.go#L18-L77).
- **Player state machine:** Player behavior is modeled with explicit states (idle/run/jump/double jump/fall/wall grab/aim) in [ecs/system/player_states.go](ecs/system/player_states.go#L5-L132), coordinated by the controller in [ecs/system/player_controller.go](ecs/system/player_controller.go#L24-L257).
- **Physics:** Physics uses Chipmunk (cp) for collisions, gravity, and contact handling, with grounded and wall contact tracking in [ecs/system/physics.go](ecs/system/physics.go#L68-L129).
- **Rendering:** A camera-aware renderer draws sprites and optional line traces, with layer-based sorting and camera transforms in [ecs/system/render.go](ecs/system/render.go#L20-L116).
- **Data-driven content:**
	- **Levels:** JSON files in [levels/](levels/) define tile layers, physics metadata, and entity spawns, embedded via [levels/embed.go](levels/embed.go#L10-L49).
	- **Prefabs:** YAML specs in [prefabs/](prefabs/) define player, camera, and aim target configuration in [prefabs/spec.go](prefabs/spec.go#L12-L79). Prefab edits trigger hot reload via [prefabs/watch.go](prefabs/watch.go#L20-L84).
- **Assets:** Sprite sheets and tiles live in [assets/](assets/), loaded on demand.
- **Editor:** The Ebitengine-based level editor in [cmd/editor/](cmd/editor/) provides tile painting, layers, physics metadata, and asset tools.

## Major Features

- **Responsive platforming:** Movement, jump, double-jump, coyote time, jump buffering, and wall grab/slide behaviors.
- **Player state machine:** Clear state transitions (idle, run, jump, double jump, fall, wall grab, aim) drive animation and behavior.
- **Aiming system:** Mouse or gamepad right-stick aiming with a target reticle and line trace to the first collision; see [ecs/system/aim.go](ecs/system/aim.go#L25-L219).
- **Physics debugging:** Toggleable debug visualization for collision shapes and player state.
- **Camera system:** Follow camera with zoom support and resize-aware layout handling.
- **Level editing workflow:** Tileset-based painting, layers, undo/save, physics layer flags, and background images.
- **Prefab hot reload:** YAML prefab edits trigger world reloads for rapid iteration.

## Code Pointers

- **Main update loop:** Scheduler update + prefab reload checks in [game.go](game.go#L66-L159).
- **ECS scheduling:** System ordering and updates in [ecs/scheduler.go](ecs/scheduler.go#L11-L27).
- **Entity queries:** Component-based queries and filtering in [ecs/world.go](ecs/world.go#L140-L192).
- **Input normalization:** Keyboard + gamepad state into a single component in [ecs/system/input.go](ecs/system/input.go#L18-L77).
- **Player controller context:** Shared closures for state logic in [ecs/system/player_controller.go](ecs/system/player_controller.go#L69-L203).
- **Collision contacts:** Wall/ground contact resolution in [ecs/system/physics.go](ecs/system/physics.go#L91-L129).
- **Rendering order + camera:** Layered draw with camera transforms in [ecs/system/render.go](ecs/system/render.go#L20-L116).
