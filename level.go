package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

const TileSize = 32

// Level represents a simple tile map stored as JSON.
type Level struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	// Layers is an optional slice of layers. Each layer is a flat array
	// of length Width*Height (row-major). Layer 0 is drawn first (bottom),
	// then layer 1, etc.
	Layers [][]int `json:"layers,omitempty"`

	// LayerMeta holds per-layer metadata such as whether tiles on the layer
	// have physics and the display color for that layer's tiles.
	LayerMeta []LayerMeta `json:"layer_meta,omitempty"`

	// per-layer rendered images built from LayerMeta.Color
	layerTileImgs []*ebiten.Image

	// player spawn in tile coordinates
	SpawnX int `json:"spawn_x,omitempty"`
	SpawnY int `json:"spawn_y,omitempty"`

	tileImg     *ebiten.Image
	triangleImg *ebiten.Image
}

type LayerMeta struct {
	HasPhysics bool   `json:"has_physics"`
	Color      string `json:"color"`
}

// Query returns all adjacent non-zero tiles near the provided rect.
// It searches tiles that are within one tile of the rect's bounding tile area.

// LoadLevel loads a level from a JSON file at path.
func LoadLevel(path string) (*Level, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lvl Level
	if err := json.Unmarshal(b, &lvl); err != nil {
		return nil, err
	}

	if lvl.Width <= 0 || lvl.Height <= 0 {
		return nil, fmt.Errorf("invalid level dimensions: %dx%d", lvl.Width, lvl.Height)
	}

	lvl.tileImg = ebiten.NewImage(TileSize, TileSize)
	lvl.tileImg.Fill(color.RGBA{R: 0x00, G: 0x00, B: 0xff, A: 0xff})

	// triangle image (always red)
	lvl.triangleImg = triangleImage(TileSize, color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff})

	// Ensure layer meta exists for each layer and build per-layer images.
	if lvl.Layers != nil && len(lvl.Layers) > 0 {
		if lvl.LayerMeta == nil || len(lvl.LayerMeta) < len(lvl.Layers) {
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

		lvl.layerTileImgs = make([]*ebiten.Image, len(lvl.LayerMeta))
		for i := range lvl.LayerMeta {
			lvl.layerTileImgs[i] = layerImageFromHex(TileSize, lvl.LayerMeta[i].Color)
		}
	}
	return &lvl, nil
}

// Draw renders the level to screen. Tile value 1 draws a red TileSize x TileSize square.
func (l *Level) Draw(screen *ebiten.Image) {
	if l == nil || l.tileImg == nil {
		return
	}
	// If Layers is provided, draw each layer in order (0..N-1). Otherwise
	// fall back to the legacy Tiles field as a single bottom layer.
	if l.Layers != nil && len(l.Layers) > 0 {
		for layer := 0; layer < len(l.Layers); layer++ {
			layerTiles := l.Layers[layer]
			if len(layerTiles) != l.Width*l.Height {
				// malformed layer, skip
				continue
			}
			// choose per-layer image if available
			img := l.tileImg
			if l.layerTileImgs != nil && layer < len(l.layerTileImgs) && l.layerTileImgs[layer] != nil {
				img = l.layerTileImgs[layer]
			}
			for y := 0; y < l.Height; y++ {
				for x := 0; x < l.Width; x++ {
					idx := y*l.Width + x
					if layerTiles[idx] == 1 {
						op := &ebiten.DrawImageOptions{}
						op.GeoM.Translate(float64(x*TileSize), float64(y*TileSize))
						screen.DrawImage(img, op)
					} else if layerTiles[idx] == 2 {
						// draw red triangle for value 2
						if l.triangleImg != nil {
							op := &ebiten.DrawImageOptions{}
							op.GeoM.Translate(float64(x*TileSize), float64(y*TileSize))
							screen.DrawImage(l.triangleImg, op)
						}
					}
				}
			}
		}
		return
	}
}

// layerImageFromHex creates an image filled with the provided hex color ("#rrggbb").
func layerImageFromHex(size int, hex string) *ebiten.Image {
	c := parseHexColor(hex)
	img := ebiten.NewImage(size, size)
	img.Fill(c)
	return img
}

// triangleImage builds an RGBA image with a filled upward-pointing triangle of the given color.
func triangleImage(size int, col color.RGBA) *ebiten.Image {
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	cx := float64(size) / 2
	// draw an upward triangle with base at bottom
	for y := 0; y < size; y++ {
		// row progress from top (0) to bottom (size-1)
		progress := float64(y) / float64(size-1)
		// width grows with progress
		rowWidth := progress * float64(size)
		left := cx - rowWidth/2
		right := cx + rowWidth/2
		for x := 0; x < size; x++ {
			fx := float64(x) + 0.5
			if fx >= left && fx <= right {
				rgba.Set(x, y, col)
			} else {
				rgba.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
	return ebiten.NewImageFromImage(rgba)
}

// parseHexColor parses a color in the form #rrggbb. Returns opaque color if parse fails.
func parseHexColor(s string) color.RGBA {
	var r, g, b uint8 = 0x00, 0x00, 0xff
	if len(s) == 7 && s[0] == '#' {
		var ri, gi, bi uint32
		if _, err := fmt.Sscanf(s[1:], "%02x%02x%02x", &ri, &gi, &bi); err == nil {
			r = uint8(ri)
			g = uint8(gi)
			b = uint8(bi)
		}
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xff}
}

// Query returns all adjacent non-zero tiles near the provided rect.
// It searches tiles that are within one tile of the rect's bounding tile area.
func (l *Level) Query(r Rect) []Rect {
	if l == nil {
		return nil
	}
	// compute tile range covering the rect
	minX := int(math.Floor(float64(r.X)/TileSize)) - 1
	minY := int(math.Floor(float64(r.Y)/TileSize)) - 1
	maxX := int(math.Floor(float64(r.X+r.Width)/TileSize)) + 1
	maxY := int(math.Floor(float64(r.Y+r.Height)/TileSize)) + 1

	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= l.Width {
		maxX = l.Width - 1
	}
	if maxY >= l.Height {
		maxY = l.Height - 1
	}

	var out []Rect
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if l.physicsTileAt(x, y) {
				trect := Rect{
					X:      float32(x * TileSize),
					Y:      float32(y * TileSize),
					Width:  TileSize,
					Height: TileSize,
				}
				out = append(out, trect)
			}
		}
	}
	return out
}

// QueryHorizontal returns tiles immediately to the left and right of r.
// It returns non-zero tiles in the column left of the rect and the column
// right of the rect, over the rows that the rect overlaps.
func (l *Level) QueryHorizontal(r Rect) []Rect {
	if l == nil {
		return nil
	}
	tileLeft := int(math.Floor(float64(r.X) / TileSize))
	tileRight := int(math.Floor(float64(r.X+r.Width-1) / TileSize))

	leftCol := tileLeft - 1
	rightCol := tileRight + 1

	tileTop := int(math.Floor(float64(r.Y) / TileSize))
	tileBottom := int(math.Floor(float64(r.Y+r.Height-1) / TileSize))

	if leftCol < 0 {
		leftCol = 0
	}
	if rightCol >= l.Width {
		rightCol = l.Width - 1
	}
	if tileTop < 0 {
		tileTop = 0
	}
	if tileBottom >= l.Height {
		tileBottom = l.Height - 1
	}

	var out []Rect
	cols := []int{leftCol, rightCol}
	for _, x := range cols {
		for y := tileTop; y <= tileBottom; y++ {
			if l.physicsTileAt(x, y) {
				out = append(out, Rect{
					X:      float32(x * TileSize),
					Y:      float32(y * TileSize),
					Width:  TileSize,
					Height: TileSize,
				})
			}
		}
	}
	return out
}

// QueryVertical returns tiles immediately above and below r.
// It returns non-zero tiles in the row above the rect and the row below the rect,
// over the columns that the rect overlaps.
func (l *Level) QueryVertical(r Rect) []Rect {
	if l == nil {
		return nil
	}
	tileTop := int(math.Floor(float64(r.Y) / TileSize))
	tileBottom := int(math.Floor(float64(r.Y+r.Height-1) / TileSize))

	topRow := tileTop - 1
	bottomRow := tileBottom + 1

	tileLeft := int(math.Floor(float64(r.X) / TileSize))
	tileRight := int(math.Floor(float64(r.X+r.Width-1) / TileSize))

	if topRow < 0 {
		topRow = 0
	}
	if bottomRow >= l.Height {
		bottomRow = l.Height - 1
	}
	if tileLeft < 0 {
		tileLeft = 0
	}
	if tileRight >= l.Width {
		tileRight = l.Width - 1
	}

	var out []Rect
	rows := []int{topRow, bottomRow}
	for _, y := range rows {
		for x := tileLeft; x <= tileRight; x++ {
			if l.physicsTileAt(x, y) {
				out = append(out, Rect{
					X:      float32(x * TileSize),
					Y:      float32(y * TileSize),
					Width:  TileSize,
					Height: TileSize,
				})
			}
		}
	}
	return out
}

// tileAt returns true if any layer (or legacy Tiles) has a non-zero value at x,y.
func (l *Level) tileAt(x, y int) bool {
	if l == nil {
		return false
	}

	if x < 0 || y < 0 || x >= l.Width || y >= l.Height {
		return false
	}

	idx := y*l.Width + x
	for _, layer := range l.Layers {
		if len(layer) != l.Width*l.Height {
			continue
		}
		if layer[idx] != 0 {
			return true
		}
	}
	return false
}

// TileValueAt returns the first non-zero tile value found at x,y across layers (0 if none).
func (l *Level) TileValueAt(x, y int) int {
	if l == nil {
		return 0
	}
	if x < 0 || y < 0 || x >= l.Width || y >= l.Height {
		return 0
	}
	idx := y*l.Width + x
	for _, layer := range l.Layers {
		if len(layer) != l.Width*l.Height {
			continue
		}
		if layer[idx] != 0 {
			return layer[idx]
		}
	}
	return 0
}

// physicsTileAt returns true if any layer (or legacy Tiles) has a non-zero value at x,y and has physics enabled.
func (l *Level) physicsTileAt(x, y int) bool {
	if l == nil {
		return false
	}

	if x < 0 || y < 0 || x >= l.Width || y >= l.Height {
		return false
	}

	idx := y*l.Width + x
	for i, layer := range l.Layers {
		if len(layer) != l.Width*l.Height {
			continue
		}
		if layer[idx] != 0 && len(l.LayerMeta) > 0 && l.LayerMeta[i].HasPhysics {
			return true
		}
	}
	return false
}

// IsGrounded returns true if the provided rect is exactly on top of any non-zero
// tile in any layer.
func (l *Level) IsGrounded(r Rect) bool {
	if l == nil {
		return false
	}
	eps := float32(0.001)
	bottom := r.Y + r.Height
	row := int(math.Floor(float64((bottom + eps) / TileSize)))
	if row < 0 || row >= l.Height {
		return false
	}
	left := int(math.Floor(float64(r.X) / TileSize))
	right := int(math.Floor(float64((r.X + r.Width - 1) / TileSize)))
	if left < 0 {
		left = 0
	}
	if right >= l.Width {
		right = l.Width - 1
	}
	for x := left; x <= right; x++ {
		if l.physicsTileAt(x, row) {
			tileTop := float32(row * TileSize)
			if bottom >= tileTop-eps && bottom <= tileTop+eps {
				return true
			}
		}
	}
	return false
}

// GetSpawnPosition returns the player's spawn position in world pixels (top-left of the spawn cell).
// If the stored spawn is out-of-bounds it clamps to (0,0).
func (l *Level) GetSpawnPosition() (float32, float32) {
	if l == nil {
		return 0, 0
	}
	x := l.SpawnX
	y := l.SpawnY
	if x < 0 || x >= l.Width {
		x = 0
	}
	if y < 0 || y >= l.Height {
		y = 0
	}
	return float32(x * TileSize), float32(y * TileSize)
}
