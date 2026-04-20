## Defective Editor

Current Ebitengine + EbitenUI level editor for the repo ECS. It edits level JSON, tile layers, prefab-backed entities, transitions, gates, triggers, breakable walls, overview layout, level size, background color, and autotile output.

## Rules

- Preserve the ECS boundary. UI and input set session/actions/components; systems apply domain changes.
- Keep file/path normalization and save/load behavior in `cmd/editor/io`.
- Prefer extending existing editor session/component state over adding package globals.
- Preserve authored data shape used by runtime loading, especially layer order, entity IDs, entity `Props`, and special transition/gate/trigger/breakable-wall fields.
- Treat layer visibility as editor-only state: do not serialize it and do not let undo/content dirtiness regress it.
- Inspector edits work on effective prefab + override component YAML, but saved data must remain valid runtime entity overrides.

## Shape

- Major libraries: uses the EbitenUI library
- `main.go`: flags, profiling, asset/prefab scan, level load/new document, autotile remap.
- `app.go`: one session entity plus one entity per layer; scheduler order is input -> ui -> commands -> undo -> camera -> layer -> area -> overview -> entity -> prefab -> tool -> autotile -> persistence.
- `component/editor_components.go`: session, actions, input, camera, tool stroke, move/area drag, undo, overview, entity, prefab, and autotile state.
- `system/editor_ui.go`: syncs ECS state into EbitenUI and turns callbacks into pending intents.
- `system/editor_inspector.go`: builds and applies inspector YAML for prefab-backed entities.
- `io/`: level, prefab, and overview file handling.

## Validation

- `go test ./cmd/editor/...`
- `go run ./cmd/editor -level player_test.json`