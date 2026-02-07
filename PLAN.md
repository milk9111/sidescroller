# ECS Migration Plan

## Overview
This document outlines a phased migration plan to move the current sidescroller game to an Entity Component System (ECS) architecture while keeping the game playable throughout the transition. Each phase includes goals, deliverables, and acceptance criteria.

## REQUIREMENTS
- Keep things simple
- Use Data-Oriented Design principles

## Phase 0 — Discovery and Inventory
**Goal:** Understand the current architecture and identify migration targets.

**Tasks:**
- Inventory current gameplay objects and responsibilities (player, enemies, bullets, pickups, camera, input, world, combat).
- Map current behavior to potential components (e.g., Transform, Velocity, Health, Sprite, Collider, AI, Input, Lifetime).
- Identify system boundaries (render, physics, AI, combat, spawning, UI, transitions).
- Decide ECS library or in-house minimal ECS (entities, components, systems, queries, events).

**Deliverables:**
- Component and system taxonomy draft.
- Migration map by feature area.
- Decision on ECS implementation approach.

**Acceptance Criteria:**
- Clear list of components and systems with ownership boundaries.
- Defined ECS API surface and folder layout.

---

## Phase 1 — ECS Foundation
**Goal:** Introduce the ECS runtime without changing gameplay behavior.

**Tasks:**
- Implement or integrate ECS core (Entity ID, Component storage, Query, System scheduler).
- Add ECS lifecycle integration into the main game loop (update order, fixed update if needed).
- Add event or message bus for cross-system interactions.
- Create basic components: Transform, Velocity, Sprite, Animation, Collider, Health, Damage, Lifetime.
- Add conversion helpers to create ECS entities from existing objects.

**Deliverables:**
- ECS core module with tests or small demo.
- Systems wiring in main loop.
- Component definitions with serialization where needed.

**Acceptance Criteria:**
- ECS loop runs with no behavioral changes.
- Components can be created, queried, and updated each frame.

---

## Phase 2 — Rendering and Animation Migration
**Goal:** Move visual representation into ECS.

**Tasks:**
- Create RenderSystem and AnimationSystem.
- Migrate `obj/entity_sprite.go` responsibilities into components and systems.
- Map existing sprite assets and animation events to ECS components.
- Ensure depth/layer handling via components.

**Deliverables:**
- RenderSystem and AnimationSystem.
- Entities render via ECS, not direct object methods.

**Acceptance Criteria:**
- Player and at least one enemy render via ECS pipeline.
- Animation events are triggered correctly in ECS.

---

## Phase 3 — Movement, Physics, and Collision
**Goal:** Move movement and collision logic into ECS systems.

**Tasks:**
- Introduce MovementSystem and CollisionSystem.
- Migrate `obj/collision_world.go` and `obj/layer.go` responsibilities.
- Convert player and enemy movement to components (Velocity, Acceleration, Gravity, Grounded).
- Define collision components and contact events.

**Deliverables:**
- CollisionSystem integrated with ECS.
- MovementSystem for all dynamic entities.

**Acceptance Criteria:**
- Player movement and collisions behave the same as before.
- Enemies interact with world collisions correctly.

---

## Phase 4 — Combat and Damage
**Goal:** Move combat logic into ECS.

**Tasks:**
- Create CombatSystem, DamageSystem, HealthSystem.
- Migrate `system/combat.go`, `component/combat.go`, and resolver logic.
- Move invincibility frames, knockback, and hit detection to components.

**Deliverables:**
- Combat systems operating on ECS components.

**Acceptance Criteria:**
- Combat behavior matches existing gameplay for player and enemies.
- Damage application, death, and pickups behave correctly.

---

## Phase 5 — AI and Spawning
**Goal:** Move AI behaviors and spawners to ECS systems.

**Tasks:**
- Implement AISystem and Behavior components.
- Migrate `obj/enemy.go`, `obj/flying_enemy.go` logic.
- Implement SpawnSystem to replace `system/spawn.go`.

**Deliverables:**
- ECS-based enemy AI and spawning.

**Acceptance Criteria:**
- Enemies spawn and behave as before.
- AI updates are frame-stable and deterministic.

---

## Phase 6 — Input, Camera, and UI
**Goal:** ECS-ify input handling, camera, and UI elements.

**Tasks:**
- Create InputSystem and map input components to entities.
- Migrate `obj/input.go` usage into ECS.
- Convert camera logic (`obj/camera.go`) into a Camera component and system.
- Keep UI as-is or introduce UI entities (pause, HUD).

**Deliverables:**
- InputSystem and CameraSystem.
- Camera behavior driven by ECS components.

**Acceptance Criteria:**
- Player input and camera tracking identical to current behavior.
- Pause UI still works.

---

## Phase 7 — Project Cleanup and Deletion of Legacy Objects
**Goal:** Remove legacy object-based gameplay code.

**Tasks:**
- Decommission obsolete `obj/*.go` structs once migrated.
- Remove unused component/resolver code from legacy paths.
- Update level loading to generate ECS entities directly.
- Ensure editor output and level JSON map to ECS components.

**Deliverables:**
- Simplified project structure centered on ECS.
- Updated level loading and editor integration.

**Acceptance Criteria:**
- Game builds and runs without legacy object types.
- All levels load and play correctly.

---

## Phase 8 — Optimization and Testing
**Goal:** Validate performance and correctness.

**Tasks:**
- Add benchmarks for core systems (movement, collision, render).
- Add regression tests for combat and input.
- Profile frame time and optimize component storage.

**Deliverables:**
- Performance report and regression test suite.

**Acceptance Criteria:**
- No regression in frame time compared to pre-ECS baseline.
- Passing tests for critical gameplay systems.

---

## Phase 9 — Documentation and Developer Enablement
**Goal:** Ensure team can work effectively with ECS.

**Tasks:**
- Document ECS conventions, system order, and component definitions.
- Provide migration guidelines for new features.
- Add diagrams for entity composition and system flow.

**Deliverables:**
- ECS developer guide.
- Updated README for building and running with ECS.

**Acceptance Criteria:**
- New developers can add a system or component with minimal guidance.

---

## Risks and Mitigations
- **Risk:** Large refactor causes regression.
  - **Mitigation:** Migrate feature-by-feature and keep old behavior running in parallel until verified.
- **Risk:** Performance overhead from ECS queries.
  - **Mitigation:** Benchmark early and optimize storage layout.
- **Risk:** Tight coupling between systems.
  - **Mitigation:** Use events and data-driven components to reduce interdependencies.

---

## Suggested Migration Order (Entities)
1. Bullets / projectiles
2. Pickups
3. Enemies (ground)
4. Enemies (flying)
5. Player
6. Camera and UI
