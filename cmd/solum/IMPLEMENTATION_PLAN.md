Target Architecture
Solum should own one mutable editor state tree and a small set of action handlers. The state should absorb the old ECS component data from editor_components.go:1: document, catalogs, session flags, camera, pointer, tool stroke, move selection, area drag, overview state, and undo stack. The replacement for “systems” should be plain packages or files under Solum that mutate *State directly, grouped by domain: layer, tool, entity, area, autotile, overview, persistence, camera, input, render.

Do not carry over the ECS shell from app.go:1. Carry over the behavior. The ECS layer is mostly acting as an indirection around state that Solum can now hold directly.

Implementation Plan

1. Freeze the feature baseline and write the migration inventory.
Current behavior lives across editor_commands.go:1, editor_tool.go:1, editor_entity.go:1, editor_layer.go:1, editor_overview.go:1, editor_persistence.go:1, and editor_render.go:1. Before moving code, define parity buckets: file I/O, layers, paint tools, autotile, entities, inspector, transitions/gates, overview, undo, save/load, shortcuts, viewport/camera.

2. Extract the reusable non-UI editor core out of editor.
The packages io, model, and autotile are already mostly reusable and should become neutral shared packages first. I would move them into an editor-neutral location such as internal/editorcore/... before the larger migration, so both old editor and Solum can compile against the same code during the transition.

3. Expand Solum state to fully replace editor ECS components.
state.go:1 currently holds only a subset of session data. Add direct fields for the old component state: layer records, entity selection, prefab placement, raw input snapshot, pointer state, camera state, stroke state, move-selection state, area drag, overview graph, undo snapshots, dirty flags, and modal/form state. The target is that nothing in editor_components.go:1 remains necessary once Solum reaches parity.

4. Introduce an action model for Solum before porting behavior.
The old editor routes UI and keyboard intent through EditorActions plus session flags. Keep that idea, but make it explicit in Solum: Action or Command types for SelectTool, PaintAt, AddLayer, MoveLayer, ToggleAutotile, SelectPrefab, PlaceEntity, EditInspectorField, SaveLevel, LoadOverviewLevel, and so on. This prevents giu callbacks from mutating deep state ad hoc and gives you a straight replacement for editor_ui.go:1 without reproducing ECS.

5. Port the pure state mutation slices first.
Start with the lowest-risk systems: undo, layer operations, persistence, autotile, prefab conversion, overview graph building. Those correspond to editor_undo.go:1, editor_layer.go:1, editor_persistence.go:1, editor_autotile.go:1, editor_prefab.go:1, and the data side of editor_overview.go:1. These are the easiest wins because they depend more on document state than on UI framework details.

6. Port entity and area editing next.
Move entity placement, selection, drag, clipboard, prefab placement, transition editing, and gate editing out of ECS into state handlers. The main sources are editor_entity.go:1 and editor_area.go:1. This is the point where Solum stops being a shell and becomes functionally useful even before the canvas is fully polished.

7. Port paint-tool behavior after state handlers are stable.
Move brush, erase, fill, box, line, spike, and room-move logic from editor_tool.go:1 into Solum controllers that operate on *State. This code is valuable and should mostly survive, but every ECS lookup should be collapsed into direct field access on Solum state.

8. Replace ebiten input polling with viewport-scoped Solum input.
The input logic in editor_input.go:1, editor_camera.go:1, and editor_commands.go:1 needs a real redesign rather than a mechanical port. In Solum, define a viewport-local input snapshot each frame, normalize mouse coordinates relative to the canvas, then run keyboard shortcuts and tool handlers against that snapshot. This is where the app-state approach pays off.

9. Rewrite the UI as giu-native panels, not widget-for-widget clones.
The ebitenui layer in editor_ui.go:1 and components should be treated as a behavior reference, not as structure to preserve. Rebuild the layout in giu as: top toolbar, left project/layer/entity panel, center viewport, right asset/tileset/inspector panel, plus modal flows for resize and prefab conversion. Keep the same commands and forms, but embrace immediate-mode giu instead of recreating the old callback graph.

10. Decide the viewport rendering strategy early and keep it incremental.
The biggest technical risk is editor_render.go:1. The pragmatic first step is to keep the existing ebiten-style drawing logic conceptually intact, render into an offscreen surface or texture, and present that inside the giu window. A full custom giu-native renderer can come later; parity matters more than purity here.

11. Port the overview screen after base editing is stable.
The overview graph is a separate workflow with its own camera, hit testing, persistence, and load behavior in editor_overview.go:1. Treat it as a later milestone, because it is self-contained and not necessary to prove the core Solum editor architecture.

12. Migrate tests alongside each slice, then remove editor.
The existing tests under system, ui, io, and autotile should be ported or split into editor-core tests versus Solum integration tests. Once Solum reaches parity, delete the obsolete UI/ECS packages and reduce editor to nothing or remove the command entirely.

Recommended Delivery Milestones

Milestone 1: shared editor-core extracted, Solum state expanded, save/load and layer ops working.
Milestone 2: entity placement, selection, inspector edits, and prefab workflows working in giu panels.
Milestone 3: tile tools, autotile, undo, and viewport camera working in Solum.
Milestone 4: overview mode, remaining shortcuts, and UI polish.

Main Risks
The highest-risk work is not giu panel construction. It is viewport rendering/input integration and preserving the old editor interaction rules without the ECS scheduler. The other risk is trying to preserve too much of the old package layout; if you do not extract shared core early, you will end up with Solum importing cmd/editor/* for too long and the migration will stall.