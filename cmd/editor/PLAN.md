# Editor Implementation Plan

## Phase 0 — Foundations
- Establish project structure for the editor entrypoint, assets loading, and config.
- Define core data models for levels, layers, tiles, entities, and metadata.
- Set up Ebiten + EbitenUI integration scaffolding.

## Phase 1 — Core Canvas & Grid
- Implement the central canvas with a fixed 32px grid.
- Render tiled layers with per-layer tinting and visibility control.
- Add basic input handling for mouse position → grid cell mapping.

## Phase 2 — Tileset System
- Build the right panel asset list from the assets/ folder.
- Load selected asset as a tileset and display selectable tile grid.
- Support tileset zoom and right-drag panning.
- Persist per-cell tileset references for reload fidelity.

## Phase 3 — Tools & Editing
- Implement brush tool (drag paint) with Ctrl+B hotkey.
- Implement erase tool (drag erase) with Ctrl+E hotkey.
- Implement fill tool (flood fill) with Ctrl+F hotkey.
- Implement line tool (straight line) with Ctrl+L hotkey.
- Ensure tools operate on the currently selected layer.

## Phase 4 — Floating Toolbar
- Add a floating toolbar over the canvas with buttons for each tool.
- Highlight active tool button and sync with hotkeys.
- Support detach/attach and drag repositioning.

## Phase 5 — Canvas Navigation
- Implement mouse wheel zoom centered on cursor.
- Implement middle-mouse drag panning.
- Apply consistent zoom/pan transforms for all canvas interactions.

## Phase 6 — Layer Management
- Create left panel layer list with selection.
- Add New Layer button and N hotkey.
- Add Q/E hotkeys to cycle layers.
- Implement layer reorder (up/down buttons).
- Add layer rename on double-click with dialog input.

## Phase 7 — Physics Metadata
- Toggle physics flag for current layer (H hotkey + button).
- Highlight physics tiles (Y hotkey + button).
- Persist physics metadata into `layer_meta` in level JSON.

## Phase 8 — Undo & Save
- Implement undo stack for all edit actions (Ctrl+Z).
- Implement save flow (Ctrl+S), prompt on first save.
- Store levels as JSON in levels/.

## Phase 9 — Prefabs
- Implement prefab placement and editing on the canvas.
- Read from `prefabs/` folder and list prefabs in bottom of left panel.
- Serialize all prefab data into level JSON as entities.

## Phase 10 — Polish & QA
- Validate all tool hotkeys and UI consistency.
- Verify tileset persistence across reloads.
- Stress-test large levels, multiple layers, and undo limits.
- Document build notes and tag usage in editor README.
