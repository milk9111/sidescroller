# ECS Editor Implementation Plan

## Goal
Build a production-ready in-game level editor under `cmd/editor` that follows the existing ECS architecture used by the game, while fully implementing all capabilities listed in `EDITOR_CAPABILITIES.md`.

## Non-Negotiable Architectural Constraints
Derived from project conventions in AGENTS.md:

- Do not inject one system into another.
- Systems communicate through components/events in the ECS world.
- Prefer reusing entities over creating/removing aggressively each frame.
- Keep orchestration thin and push logic into systems.
- Store mutable component payloads by pointer and mutate in place.
- Naming convention: while the editor can use any existing game system, any system implemented for the editor must be named with the `editor_` prefix
    (for example `editor_input`, `editor_render`). This rule applies to all systems under
    `cmd/editor` and is intended to clearly separate editor systems from runtime/game systems.

## High-Level Runtime Design

### 1) Editor app shell (thin orchestration)
Create a minimal editor runtime entrypoint in `cmd/editor` that does only:

- process startup flags (`-dir`, `-level`, `-autotile-map`)
- initialize services (assets, prefab cache, filesystem adapters)
- create ECS world + scheduler
- register systems in strict update order
- forward Ebiten update/draw/layout to ECS-backed systems

The shell must not contain editing rules.

### 2) ECS data domains
Model editor state via ECS entities/components in these domains:

- Session/global state (active tool, mode flags, focused input, current layer, save target)
- Canvas state (camera pan/zoom, cursor world position, hover cell, preview overlays)
- Level model (dimensions, layers, tiles, tileset usage metadata, entities)
- Input events (edge-triggered key/mouse actions represented as transient components)
- Tool execution (stroke start/end, drag anchor, preview geometry, pending commits)
- Undo snapshots (bounded history ring)
- UI state (panel selections, modal visibility, form text values)
- Overview map state (nodes, links, diagnostics, box layout)

### 3) Avoid per-tile entity explosion
Use a hybrid model:

- One entity per logical layer with dense 2D/flat buffers in components.
- Optional chunk entities only if needed for culling/perf.
- Keep per-cell metadata in layer-owned arrays/maps (tile value + tileset usage + autotile info).

This keeps ECS as orchestration/control while preserving performance.

## Proposed Components

Define editor-specific components in `cmd/editor/component` (or equivalent package namespace) following existing component-kind patterns.

### Core session/components

- `EditorSession`: singleton state (`activeTool`, `autotileEnabled`, `transitionMode`, `gateMode`, `physicsHighlight`, `overviewOpen`, `saveTarget`, `assetDir`)
- `EditorFocus`: focused text field/modal info; hotkey suppression signal
- `EditorClock`: frame counter, debounce timings

### Level/components

- `LevelMeta`: width/height, loaded level name, dirty flag
- `LayerData`: layer name, order index, physics flag, tile grid buffer
- `LayerTilesetUsage`: per-cell tileset path/index/size and autotile metadata
- `LevelEntityRef`: runtime editor representation of placed entities (`id`, `type`, `x`, `y`, `props`)
- `LayerVisibility`: runtime visibility flags (even if no UI toggle yet)

### Input/components

- `RawInputState`: current frame keyboard/mouse state
- `InputActionBuffer`: normalized actions (`ToolBrush`, `Undo`, `ToggleOverview`, etc.)
- `PointerState`: screen/world coords, hovered layer/cell, drag states

### Tool/components

- `ToolStroke`: active stroke data (start cell, last cell, touched cells)
- `LinePreview`: Bresenham preview points
- `AreaDragPreview`: rectangle preview for transitions/gates
- `PrefabPlacementState`: selected prefab + preview sprite handle
- `SelectionState`: selected tile, selected entity, selected transition/gate

### Autotile/components

- `AutotileConfig`: optional 47-map override loaded from `-autotile-map`
- `AutotileDirtyRegion`: regions/cells pending recompute

### Undo/components

- `UndoStack`: ring buffer (max 100) of level snapshots
- `PendingSnapshotRequest`: edge component for “capture once at stroke start” semantics

### Overview/components

- `OverviewGraph`: level nodes, inferred directional edges, diagnostics list
- `OverviewLayout`: persisted manual positions + zoom/pan state
- `OverviewSelection`: hovered/dragged/clicked node state

### Rendering/components

- `CanvasCamera`: pan + zoom (0.25 to 4.0)
- `RenderOverlayState`: grid visibility, physics overlay, selection outlines, previews

## Proposed Systems and Update Order

Order matters; this mirrors game ECS scheduling discipline.

1. `InputCaptureSystem`
   - Reads Ebiten input, writes `RawInputState` and `PointerState`.

2. `InputActionSystem`
   - Translates raw input into high-level actions and edge-triggered events.

3. `FocusGateSystem`
   - Suppresses hotkeys/tool actions when text inputs are focused.

4. `EditorModeSystem`
   - Handles mutually-exclusive mode toggles (normal, transition, gate, overview).

5. `CanvasCameraSystem`
   - Middle-drag pan + cursor-centered wheel zoom with clamped range.

6. `LayerManagementSystem`
   - Layer select/create/move/rename/physics toggle; updates entity `props.layer` indices on reorder.

7. `TilesetAssetSystem`
   - Recursively scans `-dir` for PNGs, populates asset list and tile atlas metadata.

8. `TilesetSelectionSystem`
   - Handles right-panel tile picking and autotile selection constraints.

9. `ToolStrokeSystem`
   - Begins/continues/ends brush/erase/fill/line/spike interactions; raises snapshot requests at stroke begin.

10. `TileMutationSystem`
    - Applies tile writes/erases/fills and queues autotile dirty regions.

11. `AutotileSystem`
    - 47-mask computation, neighbor recompute, full-layer recompute for autotile flood fill.

12. `PrefabCatalogSystem`
    - Loads top-level prefab YAML catalog and preview metadata.

13. `EntityPlacementSystem`
    - Places prefabs at snapped cells; supports selecting, dragging, deleting entities.

14. `SpikePlacementSystem`
    - Specialized spike line placement + rotation inference from nearby solids/bounds.

15. `TransitionEditSystem`
    - Transition area create/resize/select + form-driven property updates with one-time undo capture.

16. `GateEditSystem`
    - Gate area create/resize/select with defaults (`group=boss_gate`).

17. `OverviewSystem`
    - Loads all levels, builds connectivity diagnostics, supports pan/zoom/drag/load, persists layout JSON.

18. `UndoSystem`
    - Materializes snapshot requests and applies Ctrl+Z restores.

19. `PersistenceSystem`
    - Save/load normalization (`levels/<basename>.json`), unique entity ID enforcement, pretty JSON output.

20. `RenderSystem`
    - Draws canvas, layers, entities, overlays, previews, and UI composition.

21. `EbitenUISystem`
    - Hosts the `ebitenui` root UI tree, manages layout and rendering of `ebitenui` widgets,
      and forwards focus/input state into the ECS via `EditorFocus` and transient UI event components.
    - Loads and composes reusable components from `cmd/editor/ui/components/` and ensures
      components communicate via ECS events rather than direct coupling.

## File/Package Structure Plan

Recommended initial structure:

- `cmd/editor/main.go` (flags + app bootstrap)
- `cmd/editor/app.go` (thin loop orchestration)
- `cmd/editor/component/` (editor component definitions)
- `cmd/editor/system/` (editor systems, one concern per file)
- `cmd/editor/model/` (level DTOs + conversion helpers)
- `cmd/editor/io/` (save/load, path normalization, asset scanning)
- `cmd/editor/autotile/` (mask logic + map remap support)
- `cmd/editor/ui/` (ebitenui components and UI composition)
- `cmd/editor/ui/components/` (self-contained ebitenui widgets)
- `cmd/editor/overview/` (graph + diagnostics + layout persistence)

Important UI requirement:

- The editor UI MUST be implemented using the `ebitenui` library.
- UI must be decomposed into self-contained, reusable components (non-negotiable).
- Each UI component should live under `cmd/editor/ui/components/` and expose a clear contract
  (props/events) so components are reusable across editor panels and modes.
- UI components must not hold global editor logic; they should emit ECS-compatible events or
  transient components that editor systems consume. This preserves the ECS-first architecture and
  prevents tight coupling between UI and systems.

## Capability-by-Capability Delivery Plan

### Phase 0 — Scaffold and ECS backbone

Deliverables:

- Editor boot path under `cmd/editor` with fullscreen startup.
- World, scheduler, and session singleton entity.
- Base render loop and input capture.

Acceptance:

- App launches and cleanly exits on F12.
- Empty level canvas renders grid and camera pan/zoom works.

### Phase 1 — Level model + startup load behavior

Deliverables:

- `-level` loading from `levels/` with `name` or `name.json` handling.
- Prompt/default level dimensions when no level provided.
- Two default layers (`Background`, `Physics`) for new levels.
- Level metadata + layer physics persistence support.

Acceptance:

- Existing level round-trips with layer/tile/entity fidelity.

### Phase 2 — Core tile editing and camera controls

Deliverables:

- Brush, Erase, Fill, Line tools with previews and undo-at-stroke-start behavior.
- Top-center tool radio state + hotkeys.
- UI hover suppression for canvas actions.

Acceptance:

- Continuous paint/erase while mouse held.
- Flood fill contiguous behavior in non-autotile mode.

### Phase 3 — Autotile system

Deliverables:

- 47-mask autotile engine with corner constraints.
- Neighbor recompute after paint/erase.
- Full-layer recompute after autotile flood fill.
- Optional `-autotile-map` remap support.

Acceptance:

- Autotile-enabled painting forces selected index 0.
- Tileset grid selection disabled while autotile is enabled.

### Phase 4 — Layer management and metadata

Deliverables:

- Layer list/select/add/move/rename/physics toggle UI.
- Reorder operation updates entity `props.layer` indices.
- Physics-highlight overlay toggle.

Acceptance:

- Moving layers preserves visual ordering and entity associations.

### Phase 5 — Prefab workflow and entity editing

Deliverables:

- Prefab catalog scan (top-level `.yaml`/`.yml`).
- Prefab preview fallback chain.
- Place/select/drag/delete entities; active entities panel sync.

Acceptance:

- Drag move uses one undo snapshot at drag start.
- Escape clears prefab/entity selections.

### Phase 6 — Transition and gate area modes

Deliverables:

- Mutually exclusive transition/gate editing modes.
- Rectangle drag create/resize behavior.
- Transition property form with live edits and one-time snapshot on first change.
- Gate defaults and area overlay rendering.

Acceptance:

- Transition IDs auto-generate (`t1`, `t2`, ...).
- Enter-direction radio supports up/down/left/right.

### Phase 7 — Overview map mode

Deliverables:

- Toggle overview via `Z`.
- Load all level JSON files and infer relative placements from transitions.
- Draw links, issue diagnostics, hover tooltips.
- Persist manual box layout to `.level_overview_layout.json`.

Acceptance:

- Clicking a level box loads that level into editor state.

### Phase 8 — Save/load polish and constraints

Deliverables:

- Ctrl+S save with filename input.
- Output always normalized to `levels/<basename>.json`.
- Unique entity ID enforcement before write.
- Pretty JSON formatting.

Acceptance:

- Typing nested paths still saves by basename only.

### Phase 9 — Final UX parity and hardening

Deliverables:

- Complete hotkey parity from capabilities doc.
- Focus-aware shortcut suppression.
- Robust undo depth cap (100) and memory guardrails.
- Visual polish for previews/selection/highlights.

Acceptance:

- Full checklist from `EDITOR_CAPABILITIES.md` passes.

## Data and Persistence Contracts

Use explicit DTO structs for save/load compatibility:

- Preserve existing level JSON schema for runtime compatibility.
- Store tile layers flattened in file, expanded to 2D/runtime buffers in memory.
- Persist per-cell tileset usage metadata and autotile fields.
- Preserve entity shape (`id`, `type`, `x`, `y`, `props`).

## Undo Strategy

Snapshot content:

- all layers + tile metadata
- current layer index
- all entities

Rules:

- capture snapshot once at action start (stroke start, drag start, first property edit)
- keep max 100 snapshots (ring)
- no redo in initial implementation

## Testing Plan

### Unit tests

- Autotile mask generation and remap indexing.
- Flood fill (autotile and non-autotile paths).
- Layer reorder with entity `props.layer` remap.
- Transition diagnostics inference in overview graph.
- Save path normalization and unique ID generation.

### ECS/system tests

- Tool action pipelines from input to tile mutation.
- Undo capture semantics for stroke/drag/property-edit.
- Mode exclusivity invariants (transition vs gate vs overview).

### Golden/integration tests

- Load/save round-trip JSON golden fixtures.
- Overview diagnostics against crafted bad graph fixtures.

## Performance and Reliability Notes

- Reuse preview/selection entities; avoid per-frame churn.
- Keep heavy scans (assets, prefabs, level catalog) incremental or cached.
- Restrict expensive recompute to dirty regions whenever possible.
- Add debug overlays for tool state and autotile dirty regions to aid validation.

## Capability Traceability Matrix

All sections in `EDITOR_CAPABILITIES.md` map as follows:

- Sections 1–2: Phases 0, 1, 8
- Sections 3–4: Phases 0, 2
- Sections 5–6: Phases 2, 3
- Section 7: Phase 4
- Section 8: Phase 5
- Sections 9–10: Phase 6
- Section 11: Phase 7
- Sections 12–13: Phases 2, 9
- Sections 14–16: Phases 2, 5, 9

## Definition of Done

The editor is done when:

- every capability in `EDITOR_CAPABILITIES.md` is implemented and manually verified,
- all planned unit/system/integration tests pass,
- architecture follows ECS constraints (systems decoupled via components/events),
- and resulting level files are fully compatible with runtime loading.

```
