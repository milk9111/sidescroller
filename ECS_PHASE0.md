# Phase 0 — Discovery & Inventory (Executed)

Date: 2026-02-06

## 1) Inventory of Current Gameplay Objects and Responsibilities

### Game Loop / Orchestration
- Game
  - Owns global update order and draw order.
  - Manages pause, transitions, camera, input, debug draw, UI.
  - Calls: collision step, player update, pickup/enemy updates, bullets, combat resolver, transitions.

### World / Level
- system.World
  - Level loading and transitions.
  - Creates CollisionWorld and spawns entities from level data.
- obj.Level
  - Tile map, layers, parallax, transitions, placed entities.
  - Draws layers and outlines; caches tileset sprites.
- obj.Layer
  - Tile drawing and optional outline rendering per layer.

### Player and Movement
- obj.Player
  - State machine (idle/run/jump/fall/wallgrab/aim/swing/dash/attack).
  - Physics body integration (Chipmunk) + manual velocity tracking.
  - Animation selection per state.
  - Combat: hitboxes/hurtboxes + health + faction.
  - Abilities: double jump, wall grab, swing, dash.
  - Uses obj.Input, obj.Anchor, CollisionWorld.

### Enemies
- obj.Enemy
  - State machine (idle/move/attack).
  - Pathfinding (A*), combat, animation.
  - Physics body via CollisionWorld.
- obj.FlyingEnemy
  - State machine (idle/move/attack).
  - Pathfinding and ranged attack (spawns bullets).
  - Physics body in Chipmunk (no gravity).

### Projectiles
- obj.Bullet
  - Pooled projectiles with position/velocity/lifetime.
  - Handles collision with physics tiles and player hurtboxes.
  - Emits combat events via hitboxes.

### Pickups
- obj.Pickup
  - Floating pickup animation and trigger on overlap.
  - Enables abilities on player.

### Camera / Input / UI / Transitions
- obj.Camera
  - Smooth follow, zoom, world bounds, shake.
- obj.Input
  - Keyboard/gamepad input and mouse world coordinates.
- obj.Transition
  - Fade transition + level load callback.
- pause_ui.go
  - Pause menu UI overlay.

### Physics / Collision
- obj.CollisionWorld
  - Chipmunk space setup, static tile shapes, player/enemy bodies.
  - Grounded/wall detection via collision handlers.
  - Debug draw for physics shapes.

### Combat (Components + System)
- component.*
  - Health, hitboxes/hurtboxes, combat resolver, animation events.
- system.ResolveCombat
  - Central combat resolution each frame.


## 2) Component & System Taxonomy (Draft)

### Core Components (Data-Only)
- Transform (position, rotation, scale)
- Velocity (vx, vy)
- Acceleration / Forces
- PhysicsBody (Chipmunk body/shape handles or indices)
- Collider (shape, collision type, sensor, grounded/wall flags)
- Sprite (image id, size, offsets, flip)
- Animation (clip id, current frame, fps, loop)
- RenderLayer (z-order / layer index)
- Health (current, max, iframes)
- Hitbox / Hurtbox (rects, enabled, faction)
- Faction
- Lifetime (age, max)
- InputState (moveX, jump, aim, dash, mouse world)
- AbilityFlags (double jump, wall grab, swing, dash)
- StateTag (player state enum, enemy state enum)
- AIState (aggro range, cooldowns, target id)
- Pathfinding (path list, index, recalc timer)
- Projectile (owner id, damage, speed, rotation)
- Pickup (type, enabled)
- CameraFollow (target entity, smoothing, bounds)
- TransitionTrigger (target, link id, direction)
- Anchor/Grapple (anchor position, rope length, joint handle)

### Systems (Boundaries)
- InputSystem (fills InputState, aim ray)
- MovementSystem (apply velocity/forces, ability effects)
- PhysicsSystem (Chipmunk step, grounded/wall flags)
- CollisionSystem (tile/shape collision + sensors)
- AnimationSystem (advance frames, emit events)
- RenderSystem (draw sprites, layers, bullets, debug)
- CombatSystem (resolve hit/hurtbox collisions, damage)
- AISystem (enemy/flying enemy behavior)
- PathfindingSystem (A* planning)
- ProjectileSystem (spawn/update/bullet lifetime)
- PickupSystem (overlap with player, grant ability)
- CameraSystem (follow, bounds, shake)
- TransitionSystem (fade + level load)
- SpawnSystem (spawn entities from level placed entities)
- UISystem (pause UI)


## 3) Migration Map by Feature Area

### Rendering & Animation
- From: obj.Player.Draw, obj.Enemy.Draw, obj.FlyingEnemy.Draw, obj.Layer.Draw, obj.Bullet.Draw
- To: RenderSystem + AnimationSystem using Sprite/Animation components.

### Movement & Physics
- From: obj.Player.Update/state machine + CollisionWorld.Step
- To: MovementSystem + PhysicsSystem + CollisionSystem.

### Combat
- From: component.CombatResolver + system.ResolveCombat
- To: CombatSystem using Hitbox/Hurtbox/Health components.

### AI
- From: obj.Enemy.Update + obj.FlyingEnemy.Update + A* usage
- To: AISystem + PathfindingSystem.

### Projectiles
- From: obj.Bullet (pool + Update/Draw)
- To: ProjectileSystem with Projectile + Transform + Velocity + Lifetime components.

### Pickups
- From: obj.Pickup Update/Draw
- To: PickupSystem + RenderSystem.

### Input / Camera
- From: obj.Input + obj.Camera
- To: InputSystem + CameraSystem with InputState/CameraFollow.

### World / Level / Transitions
- From: system.World + obj.Level + obj.Transition
- To: WorldSystem + TransitionSystem with data-driven spawn definitions.


## 4) ECS Implementation Decision

### Decision
Implement a minimal in-house ECS tailored to this game.

### Rationale (Keep Simple + Data-Oriented)
- Existing codebase is straightforward; a minimal ECS avoids heavy dependencies.
- DOD-friendly layouts are easier to control with a custom ECS (SoA storage, sparse sets).
- Chipmunk integration and ebiten rendering need tight, predictable update order.

### Proposed ECS API Surface
- Entity: integer id with generation (to avoid stale references).
- Component storage: sparse-set per component type (SoA slices for cache-friendly iteration).
- Queries: select entities with required component sets.
- Systems: ordered list with Update(delta) + optional Draw().
- Events: lightweight event queue for cross-system signals (combat, animation, transition).

### Proposed Folder Layout
- ecs/ (core ECS runtime)
  - world.go (entities, systems, update order)
  - entity.go (id + generation)
  - storage.go (sparse sets)
  - query.go
  - events.go
- ecs/components/ (ECS component data structs)
- ecs/systems/ (ECS systems)
- legacy/ (temporary wrappers / adapters during migration)


## 5) Phase 0 Acceptance Criteria: Met
- Component & system taxonomy drafted.
- Migration map by feature area drafted.
- ECS implementation approach decided.
- ECS API surface + folder layout defined.
