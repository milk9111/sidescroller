## Defective Editor (Legacy Ebitengine Editor)

This subtree contains the original level editor built on Ebitengine, the repo ECS, and EbitenUI. It edits level JSON, prefab-backed entities, transitions, gates, and autotiled tile layers. Treat it as the behavior baseline while Solum is being brought to parity.

## Requirements

- Preserve the ECS boundary: systems communicate through components and session state, not by directly calling into each other.
- Keep UI as intent capture only. `cmd/editor/ui/` and `cmd/editor/ui/components/` should surface callbacks and view state; domain mutations belong in editor systems.
- Prefer changing existing session/component data over adding ad hoc globals or hidden package state.
- Keep save/load/path normalization inside `cmd/editor/io`; do not duplicate filename or directory resolution logic in systems or widgets.
- Reusable editor logic that is not Ebitengine- or EbitenUI-specific should be extracted toward `internal/editorcore/` instead of being duplicated for Solum.
- Preserve authored level/entity data conventions, especially layer indexing, transition/gate props, and entity ID stability on save.

## Current Architecture

- **Entry + bootstrap:** `main.go` parses flags, scans assets/prefabs, loads the level document, and starts shared profiling support.
- **App orchestration:** `app.go` creates the ECS world, bootstraps editor session entities/components, and registers the editor scheduler.
- **State model:** `component/` holds editor session, focus, input, layer, entity, overview, autotile, undo, and camera state. Most behavior flows through the single session entity plus per-layer entities.
- **System pipeline:** input/UI intent -> commands/undo/camera -> layer/area/overview/entity/prefab -> tool/autotile -> persistence.
- **UI layer:** `ui/` and `ui/components/` build EbitenUI panels and modals. `system/editor_ui.go` is the bridge that syncs ECS state into widgets and converts callbacks into pending intents.
- **Reusable core candidates:** `io/`, `model/`, and `autotile/` already contain logic that can be shared. Keep pure document operations portable so Solum can consume the same behavior.

## Editing Guidance

- Add new editor behavior by extending components plus a focused system, rather than growing `app.go` or burying logic in widget callbacks.
- If a shortcut, toolbar action, or modal changes editor state, route it through the existing intent flow: UI/input -> action/session flags -> domain system.
- When changing entity editing, preserve `Props` compatibility with runtime level loading in `ecs/entity/level.go`; editor-authored JSON must still load in-game.
- When changing save behavior, maintain `NormalizeLevelTarget`, `ResolveLevelPath`, and JSON formatting semantics so editor and shared core stay aligned.
- Prefer tests around systems, UI sync state, autotile, and I/O rather than relying only on manual editor runs.

## Validation

- Primary test command: `go test ./cmd/editor/...`
- Useful manual smoke test: `go run ./cmd/editor -level player_test.json`
- Profiling entrypoints live in `main.go` via the shared profiler flags (`-pprof`, `-cpuprofile`, `-trace`, `-memprofile`, `-memprofile-sample`).

## Key Code Pointers

- Startup and flags: `cmd/editor/main.go`
- App bootstrap and scheduler wiring: `cmd/editor/app.go`
- Editor components/state: `cmd/editor/component/editor_components.go`
- Input and shortcuts: `cmd/editor/system/editor_input.go`, `cmd/editor/system/editor_commands.go`
- Tile/entity editing: `cmd/editor/system/editor_tool.go`, `cmd/editor/system/editor_entity.go`, `cmd/editor/system/editor_area.go`
- Layer/autotile/undo: `cmd/editor/system/editor_layer.go`, `cmd/editor/system/editor_autotile.go`, `cmd/editor/system/editor_undo.go`
- Save/load/overview/prefabs: `cmd/editor/system/editor_persistence.go`, `cmd/editor/system/editor_overview.go`, `cmd/editor/system/editor_prefab.go`
- UI bridge and widgets: `cmd/editor/system/editor_ui.go`, `cmd/editor/ui/editor_ui.go`, `cmd/editor/ui/components/`
- Shared extraction targets: `internal/editorcore/io/`, `internal/editorcore/levelops/`, `internal/editorcore/model/`, `internal/editorcore/autotile/`