# Editor Features

This folder contains the Ebiten-based level editor used for building and editing sidescroller levels. The editor is organized into a left entity/layers panel, a central canvas, and a right tileset/assets panel.

## Dependencies 
The editor uses [github.com/ebitenui/ebitenui](https://github.com/ebitenui/ebitenui) for all UI components. It is deeply integrated into every aspect of the editor.

## Core Capabilities

- **New level creation**
  - On launch, if no file is provided with the `-level` flag, the editor prompts for width and height (in tile cells). This opens a dialog window for input.

- **Tile painting on a grid**
  - Left-click uses selected tool on the current layer.
  - Uses a fixed cell size (32px) with per-layer tinting.

- **Tileset-driven painting**
  - Right panel lists image assets from the assets/ folder.
  - Click an asset to load it as a tileset.
  - Select tiles directly from the tileset panel.
  - Tileset usage is saved per-cell so levels can restore tilesets when reloaded.

- **Tools**
  - **Brush tool** toggled with Ctrl+B for regular brush painting (drag to use continuously).
  - **Erase tool** toggled with Ctrl+E for erasure painting (drag to use continuously).
  - **Fill tool** toggled with Ctrl+F for flood-fill painting.
  - **Line tool** toggled with Ctrl+L for straight-line painting.

- **Tools UI** 
  - There is a floating (can attach/detach and be moved) toolbar on the top of the canvas.
  - Has a button for each tool.
  - The selected tool button is always highlighted.
  - Using a tool hotkey changes the highlighted tool button.

- **Canvas navigation**
  - Mouse wheel zooms the canvas (centered on cursor).
  - Middle-mouse drag pans the canvas.
  - Tileset panel also supports mouse-wheel zoom and right-drag panning.

- **Layer management**
  - Left panel lists all of the layers for the given level.
  - Create layers with N or "New Layer" button on left panel.
  - Cycle layers with Q/E or click in the layer list.
  - Reorder layers using up/down buttons in the left panel.
  - Rename layers by double-clicking the layer name. This opens a dialog window for input.

- **Physics layer metadata**
  - Toggle physics on the current layer with H or the "Toggle Physics" button.
  - Highlight physics tiles with Y or the "Highlight Physics" button.
  - Physics metadata is saved in layer_meta in level JSON.

- **Background images**
  - Press B or the "Background" button to load a background image (native dialog when built with the dialog tag).
  - Backgrounds are scaled to the level size and saved in the level JSON.

- **Undo & save**
  - Undo with Ctrl+Z.
  - Save with Ctrl+S (prompts for a filename on first save).
  - Levels are stored as JSON in levels/.

## Panels and UI Layout

- **Left panel:** Layer list and canvas buttons.
- **Canvas:** Main editing surface for tiles, entities, transitions, and spawn. Also has a floating toolbar with tool buttons.
- **Right panel:** Tileset asset list and selectable tile grid with zoom/pan.

## Build Notes

- The background file picker uses a build tag:
  - Build with dialog support: `-tags dialog` (requires github.com/sqweek/dialog).
  - Without dialog support, background selection will panic if invoked.
