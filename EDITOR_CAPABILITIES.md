# Defective Editor Capabilities

This document describes the current capabilities of the in-game level editor implemented in `cmd/editor`.

## 1) Launch and startup behavior

### Command-line flags
- `-dir` (default: `assets`)
  - Directory scanned recursively for tileset images.
  - The editor currently loads `.png` files from this directory.
- `-level`
  - Optional level file to load from `levels/`.
  - Accepts either `name` or `name.json`.
- `-autotile-map`
  - Optional JSON file containing a 47-entry autotile mapping (mask-order index mapping).
  - If valid, this remaps which tiles are selected for each autotile mask.

### Initial setup behavior
- Runs fullscreen.
- If `-level` is provided and loads successfully:
  - Uses that level’s width/height.
  - Loads layer tile data, per-tile tileset metadata, layer physics metadata, and entities.
  - Attempts to restore referenced tileset from `assets/`.
- If no level is loaded:
  - Prompts in terminal for width/height in tiles (defaults based on monitor size and 32px grid).
  - Creates two default layers:
    - `Background` (physics off)
    - `Physics` (physics on)

## 2) Data model and persistence support

### Level content managed by editor
- Grid dimensions (`width`, `height`)
- Tile layers (flat storage in save file, 2D in editor runtime)
- Per-layer metadata:
  - `physics` flag
- Per-cell tileset usage metadata:
  - tileset path
  - tile index
  - tile size
  - autotile info (`auto`, `base_index`, `mask`)
- Entities with:
  - `id`
  - `type`
  - `x`, `y`
  - arbitrary `props`

### Save behavior
- Save shortcut: `Ctrl+S`
- Save target comes from left-panel **File** input.
- Output path is normalized to `levels/<basename>.json`.
  - Any directory typed in input is ignored (basename only).
  - `.json` is appended if missing.
- Automatically ensures every entity has a unique ID before saving.
- Writes pretty-printed JSON.

### Load behavior
- Can load level from startup `-level`.
- Can load levels interactively from the overview map (see Section 11).

## 3) Editor UI layout

### Left panel
- File name field (save target)
- Layers section
- Prefabs section
- Active Entities section
- Transitions section
- Gates section
- Layer rename modal dialog support

### Center
- Main editable tile canvas (Ebiten rendered)

### Top-center toolbar
- Tool radio group:
  - Brush
  - Erase
  - Fill
  - Line
  - Spike

### Right panel
- Asset list (`.png`)
- Tileset tile-grid selector

## 4) View/camera controls

- Middle mouse drag: pan canvas
- Mouse wheel: zoom canvas in/out (cursor-centered)
- Zoom range on main canvas: `0.25x` to `4.0x`
- Grid remains visible and scales with zoom

## 5) Tile editing tools

### Brush
- Paints selected tile while holding left mouse.
- Paint action starts with undo snapshot once at stroke start.

### Erase
- Clears tiles while holding left mouse.
- Erase action starts with undo snapshot once at stroke start.

### Fill
- Flood-fills contiguous region from clicked tile.
- Uses two modes:
  - Autotile-aware fill (when autotile enabled)
  - Raw tile-value fill (when autotile disabled)

### Line
- Click-drag-release to define start/end cells.
- Draws with Bresenham line rasterization.
- Supports preview while dragging.
- Integrates with autotile recomputation in affected region.

### Spike
- Line-like placement for spike entities.
- Supports preview while dragging.
- Places or updates `spike` entities at grid cells.
- Auto-computes and stores spike `rotation` based on nearby solid/boundary surfaces.

## 6) Autotiling capabilities

- Supports 47-mask autotile logic.
- Uses cardinal + corner mask rules with corner constraints.
- Autotile metadata is persisted per cell in `TilesetUsage`.
- Neighboring autotiles are recomputed automatically after paint/erase.
- Full-layer recompute after autotile flood fill.
- Optional external remap via `-autotile-map` JSON.

### Important behavior
- When autotile is enabled:
  - selected tile is forced to index `0`
  - manual tile selection in tileset grid is disabled

## 7) Layer management

### Operations
- Select current layer from list
- Add layer (`New` button or hotkey)
- Move layer up/down (`Up` / `Down`)
- Rename layer via modal dialog (`Rename`)
- Toggle current layer physics (`Physics On/Off`)

### Effects and metadata
- Layer reordering also swaps entity `props.layer` indices accordingly.
- Current layer selection updates physics button state.

### Visual aid
- Physics highlight overlay can be toggled to visualize solid tiles on all physics-enabled layers.

## 8) Prefab workflow and entity placement

### Prefab catalog
- Reads top-level `.yaml` / `.yml` files in `prefabs/`.
- Selecting a prefab enters prefab placement mode.

### Placement behavior
- Left click on canvas places entity at snapped grid cell.
- Entity `props` include:
  - `layer` (current layer index)
  - `prefab` path when available

### Preview behavior
- Attempts to render prefab preview using (in order):
  1. animation preview spec first animation definition first frame
  2. sprite preview image
  3. fallback decode via full entity build spec (`animation`/`sprite` components)
  4. colored square fallback if no visual data

### Existing entity selection and movement
- Left click entity to select it.
- Non-area entities can be dragged to reposition.
- Drag move creates one undo snapshot at drag start.
- Selected entity is mirrored in Active Entities list.

### Deletion and cancel
- `Delete` / `Backspace`: delete selected entity.
- `Escape`: clear prefab/entity selection and drag state.

## 9) Transition area editing mode

### Mode behavior
- Toggle with **Transitions: On/Off** button.
- Mutually exclusive with Gate mode.
- When active:
  - normal prefab/tile interactions are suppressed for area creation.
  - left-drag defines rectangular transition area.

### Create/resize transition entities
- Creates or resizes `transition` entities with rectangle props:
  - `w`, `h` (in pixels)
  - `layer`
  - `id`
  - `to_level`
  - `linked_id`
  - `enter_dir`
- Auto-generates transition IDs (`t1`, `t2`, ...).

### Transition property editor form
- Visible when a transition is selected.
- Editable fields:
  - ID
  - To level
  - Linked transition ID
  - Enter direction (radio group): up/down/left/right
- Property edits are applied live.
- First property edit after selection records undo snapshot.

## 10) Gate area editing mode

### Mode behavior
- Toggle with **Gates: On/Off** button.
- Mutually exclusive with Transition mode.
- Left-drag creates/resizes rectangular `gate` entities.

### Gate defaults
- New gates include:
  - `w`, `h`
  - `layer`
  - `group` default: `boss_gate`

## 11) Level overview map mode

### Open/close
- Press `Z` (without Ctrl) to toggle overview.

### Purpose
- Visualize all levels in `levels/` as draggable boxes.
- Show transition connectivity lines between levels.
- Load a level by clicking its box.

### Features
- Reads all `.json` levels from `levels/`.
- Infers relative placement from transition directions when possible.
- Stores manual layout positions in `.level_overview_layout.json`.
- Supports:
  - wheel zoom
  - middle-mouse pan
  - left-drag to move level boxes
  - click box to load level

### Validation diagnostics shown in overview
- Transition target level missing
- Multiple outgoing transitions using the same direction to different levels
- Incoming transition slot conflicts from multiple source levels
- Problematic levels are highlighted and show issue tooltip on hover

## 12) Hotkeys and mouse controls

### Keyboard
- `Ctrl+B`: Brush
- `Ctrl+E`: Erase
- `Ctrl+F`: Fill
- `Ctrl+L`: Line
- `Ctrl+K`: Spike
- `Ctrl+Z`: Undo
- `Ctrl+S`: Save
- `Q`: Previous layer
- `E`: Next layer
- `N`: New layer
- `H`: Toggle current layer physics
- `Y`: Toggle physics highlight
- `T`: Toggle autotile
- `Z`: Toggle level overview
- `Delete` / `Backspace`: Delete selected entity
- `Escape`: Clear selection/modes state
- `F12`: Exit editor process

### Mouse
- Left click / drag: tool action, entity action, or area drag (mode-dependent)
- Middle drag: pan
- Wheel: zoom

## 13) Undo system

- Undo stack stores snapshots of:
  - all layers and tile metadata
  - current layer index
  - all entities
- Max undo depth: 100 snapshots.
- No redo stack currently.

## 14) Rendering and visual behavior

- Draws visible layers in runtime-like order with entity interleaving by render layer.
- Entity render layer can be inferred from prefab `render_layer` component.
- Shows:
  - line placement preview
  - selected tile preview under cursor
  - spike preview (with computed rotation)
  - prefab preview under cursor
  - transition/gate area overlays
  - physics highlight overlay

## 15) Focus/input handling details

- If a text input is focused, editor hotkeys are suppressed to avoid accidental actions while typing.
- UI hover suppresses canvas paint actions to prevent panel clicks from painting beneath UI.

## 16) Current practical limits / notes

- Asset scanning is PNG-only for tilesets.
- Prefab listing is non-recursive (top-level `prefabs/` files only).
- Layer visibility exists in runtime data but no explicit UI toggle is currently exposed.
- Save path is constrained to `levels/<basename>.json`.
- No redo command currently.
