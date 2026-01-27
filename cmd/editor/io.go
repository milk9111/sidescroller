package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// UndoSnapshot represents either a full snapshot (Full != nil) or a set of
// per-layer deltas (Deltas).
type UndoSnapshot struct {
	Full   [][]int
	Deltas []LayerDelta
}

type LayerDelta struct {
	Layer   int
	Changes map[int]int // cell index -> previous value
}

func (g *Editor) Save() error {
	if g.level != nil {
		normalizeBackgroundPaths(g.level)
	}
	if g.filename == "" {
		// ensure levels dir
		if err := os.MkdirAll("levels", 0755); err != nil {
			return err
		}
		g.filename = filepath.Join("levels", fmt.Sprintf("level_%d.json", time.Now().Unix()))
	} else {
		// ensure directory exists
		if err := os.MkdirAll(filepath.Dir(g.filename), 0755); err != nil {
			return err
		}
	}
	f, err := os.Create(g.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	// build TilesetUsage: per-layer 2D arrays of tileset info for cells that use a tileset
	if g.level != nil {
		usage := make([][][]*TilesetEntry, len(g.level.Layers))
		for li := range g.level.Layers {
			layer := g.level.Layers[li]
			rows := make([][]*TilesetEntry, g.level.Height)
			for y := 0; y < g.level.Height; y++ {
				row := make([]*TilesetEntry, g.level.Width)
				for x := 0; x < g.level.Width; x++ {
					idx := y*g.level.Width + x
					if idx >= 0 && idx < len(layer) {
						v := layer[idx]
						if v >= 3 && g.tilesetPanel.tilesetPath != "" && g.tilesetPanel.tilesetTileW > 0 && g.tilesetPanel.tilesetTileH > 0 {
							row[x] = &TilesetEntry{Path: g.tilesetPanel.tilesetPath, Index: v - 3, TileW: g.tilesetPanel.tilesetTileW, TileH: g.tilesetPanel.tilesetTileH}
						} else {
							row[x] = nil
						}
					}
				}
				rows[y] = row
			}
			usage[li] = rows
		}
		g.level.TilesetUsage = usage
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(g.level)
}

func (g *Editor) Load(filename string) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var lvl Level
	if err := json.Unmarshal(b, &lvl); err != nil {
		return err
	}

	// ensure there is at least one layer
	if lvl.Layers == nil || len(lvl.Layers) == 0 {
		lvl.Layers = make([][]int, 1)
		lvl.Layers[0] = make([]int, lvl.Width*lvl.Height)
	}

	// ensure layer meta exists for each layer
	if lvl.LayerMeta == nil || len(lvl.LayerMeta) < len(lvl.Layers) {
		// fill missing with defaults
		meta := make([]LayerMeta, len(lvl.Layers))
		for i := range meta {
			if lvl.LayerMeta != nil && i < len(lvl.LayerMeta) {
				meta[i] = lvl.LayerMeta[i]
			} else {
				meta[i] = LayerMeta{HasPhysics: false, Color: "#3c78ff"}
			}
		}
		lvl.LayerMeta = meta
	}

	normalizeBackgroundPaths(&lvl)

	g.level = &lvl
	if g.currentLayer >= len(g.level.Layers) {
		g.currentLayer = 0
	}
	// rebuild per-layer images
	g.layerTileImgs = make([]*ebiten.Image, len(g.level.LayerMeta))
	for i := range g.level.LayerMeta {
		g.layerTileImgs[i] = layerImageFromHex(g.cellSize, g.level.LayerMeta[i].Color)
	}

	g.filename = filename

	// preload and cache any background images listed in the level
	if g.backgrounds == nil {
		g.backgrounds = NewBackground()
	}
	// Clear existing images and re-add from level entries
	g.backgrounds.Images = make([]*ebiten.Image, 0, len(g.level.Backgrounds))
	g.backgrounds.Entries = make([]BackgroundEntry, 0, len(g.level.Backgrounds))
	for _, be := range g.level.Backgrounds {
		if be.Path == "" {
			continue
		}
		loaded := false
		if b, err := os.ReadFile(be.Path); err == nil {
			if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
				g.backgrounds.Add(be.Path, img, g.level, g.cellSize)
				loaded = true
			}
		}
		if !loaded {
			if b, err := os.ReadFile(filepath.Join("assets", be.Path)); err == nil {
				if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
					g.backgrounds.Add(be.Path, img, g.level, g.cellSize)
					loaded = true
				}
			}
		}
		if !loaded {
			base := filepath.Base(be.Path)
			if b, err := os.ReadFile(filepath.Join("assets", base)); err == nil {
				if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
					g.backgrounds.Add(be.Path, img, g.level, g.cellSize)
				}
			}
		}
	}

	// If the saved level contains TilesetUsage metadata, try to open the first referenced tileset
	if g.level != nil && g.level.TilesetUsage != nil {
		found := false
		for li := range g.level.TilesetUsage {
			layerUsage := g.level.TilesetUsage[li]
			if layerUsage == nil {
				continue
			}

			for y := 0; y < g.level.Height && !found; y++ {
				for x := 0; x < g.level.Width; x++ {
					if y < len(layerUsage) && x < len(layerUsage[y]) {
						entry := layerUsage[y][x]
						if entry != nil && entry.Path != "" {
							// attempt to load tileset from assets/<path>
							if b, err := os.ReadFile(filepath.Join("assets", entry.Path)); err == nil {
								if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
									g.tilesetPanel.tilesetImg = ebiten.NewImageFromImage(img)
									g.tilesetPanel.tilesetPath = entry.Path
									// g.tilesetPanel.tilesetTileW = entry.TileW
									// g.tilesetPanel.tilesetTileH = entry.TileH
									if g.tilesetPanel.tilesetTileW > 0 {
										g.tilesetPanel.tilesetCols = g.tilesetPanel.tilesetImg.Bounds().Dx() / g.tilesetPanel.tilesetTileW
									}
									g.tilesetPanel.selectedTile = entry.Index
									found = true
									break
								}
							}
						}
					}
				}
				if found {
					break
				}
			}
		}
	}

	return nil
}

// pushSnapshot stores a deep copy of the current Layers for undo.
// pushSnapshot records either a delta (when indices provided) or a full
// snapshot (when indices == nil). Deltas store previous values for the
// specified indices on the given layer.
func (g *Editor) pushSnapshot(layer int, indices []int) {
	if g.level == nil || g.level.Layers == nil {
		return
	}
	if indices == nil {
		// full snapshot (existing behavior with size guard)
		totalInts := 0
		for i := range g.level.Layers {
			totalInts += len(g.level.Layers[i])
		}
		estBytes := int64(totalInts) * 8
		const maxSnapshotBytes = int64(100 * 1024 * 1024) // 100 MB
		if estBytes > maxSnapshotBytes {
			log.Printf("pushSnapshot: skipping full snapshot; estimated size %d MB exceeds %d MB", estBytes/1024/1024, maxSnapshotBytes/1024/1024)
			return
		}
		copyLayers := make([][]int, len(g.level.Layers))
		for i := range g.level.Layers {
			layerData := g.level.Layers[i]
			lcopy := make([]int, len(layerData))
			copy(lcopy, layerData)
			copyLayers[i] = lcopy
		}
		snap := UndoSnapshot{Full: copyLayers}
		g.undoStack = append(g.undoStack, snap)
	} else {
		// delta snapshot for a specific layer
		if layer < 0 || layer >= len(g.level.Layers) {
			return
		}
		changes := make(map[int]int)
		layerData := g.level.Layers[layer]
		for _, idx := range indices {
			if idx >= 0 && idx < len(layerData) {
				if _, seen := changes[idx]; !seen {
					changes[idx] = layerData[idx]
				}
			}
		}
		if len(changes) == 0 {
			return
		}
		ld := LayerDelta{Layer: layer, Changes: changes}
		snap := UndoSnapshot{Deltas: []LayerDelta{ld}}
		g.undoStack = append(g.undoStack, snap)
	}
	if len(g.undoStack) > g.maxUndo {
		// drop oldest
		g.undoStack = g.undoStack[1:]
	}
}

// pushSnapshotDelta appends a prepared LayerDelta into the undo stack.
func (g *Editor) pushSnapshotDelta(ld LayerDelta) {
	snap := UndoSnapshot{Deltas: []LayerDelta{ld}}
	g.undoStack = append(g.undoStack, snap)
	if len(g.undoStack) > g.maxUndo {
		g.undoStack = g.undoStack[1:]
	}
}

// Undo restores the last snapshot if available.
func (g *Editor) Undo() {
	n := len(g.undoStack)
	if n == 0 {
		return
	}
	snap := g.undoStack[n-1]
	g.undoStack = g.undoStack[:n-1]
	// apply snapshot: full or deltas
	if snap.Full != nil {
		g.level.Layers = make([][]int, len(snap.Full))
		for i := range snap.Full {
			layer := snap.Full[i]
			lcopy := make([]int, len(layer))
			copy(lcopy, layer)
			g.level.Layers[i] = lcopy
		}
		if g.currentLayer >= len(g.level.Layers) {
			g.currentLayer = len(g.level.Layers) - 1
			if g.currentLayer < 0 {
				g.currentLayer = 0
			}
		}
		return
	}

	// apply deltas
	for _, ld := range snap.Deltas {
		if ld.Layer < 0 || ld.Layer >= len(g.level.Layers) {
			continue
		}
		layer := g.level.Layers[ld.Layer]
		for idx, val := range ld.Changes {
			if idx >= 0 && idx < len(layer) {
				layer[idx] = val
			}
		}
		g.level.Layers[ld.Layer] = layer
	}
}
