package obj

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
)

// Level represents a simple tile map stored as JSON.
type Level struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	// Layers is an optional slice of layers. Each layer is a flat array
	// of length Width*Height (row-major). Layer 0 is drawn first (bottom),
	// then layer 1, etc.
	Layers [][]int `json:"layers,omitempty"`
	// TilesetUsage stores per-layer, per-cell tileset metadata when a tileset tile is used.
	TilesetUsage [][][]*TilesetEntry `json:"tileset_usage,omitempty"`

	// LayerMeta holds per-layer metadata such as whether tiles on the layer
	// have physics and the display color for that layer's tiles.
	LayerMeta []LayerMeta `json:"layer_meta,omitempty"`

	// per-layer rendered images built from LayerMeta.Color
	layerTileImgs []*ebiten.Image

	// player spawn in tile coordinates
	SpawnX int `json:"spawn_x,omitempty"`
	SpawnY int `json:"spawn_y,omitempty"`

	// Backgrounds stores background layers for parallax rendering.
	Backgrounds []BackgroundEntry `json:"backgrounds,omitempty"`
	// legacy single-background path (backwards compatible)
	BackgroundPath string `json:"background_path,omitempty"`

	tileImg     *ebiten.Image
	triangleImg *ebiten.Image
	// cache of loaded tileset images keyed by path
	tilesetImgs map[string]*ebiten.Image
	// missingTileImg is drawn when a referenced tileset tile cannot be found.
	missingTileImg *ebiten.Image
	backgroundImgs []*ebiten.Image
}

// TilesetEntry records which tileset file and tile index plus tile size used for a cell.
type TilesetEntry struct {
	Path  string `json:"path"`
	Index int    `json:"index"`
	TileW int    `json:"tile_w"`
	TileH int    `json:"tile_h"`
}

type LayerMeta struct {
	HasPhysics bool   `json:"has_physics"`
	Color      string `json:"color"`
}

// BackgroundEntry stores a background image reference and optional parallax factor.
type BackgroundEntry struct {
	Path     string  `json:"path"`
	Parallax float64 `json:"parallax,omitempty"`
}

// Query returns all adjacent non-zero tiles near the provided rect.
// It searches tiles that are within one tile of the rect's bounding tile area.

// LoadLevel loads a level from a JSON file at path.
func LoadLevel(path string) (*Level, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return loadLevelFromBytes(b)
}

// LoadLevelFromFS loads a level JSON from an fs.FS (e.g. embedded levels).
func LoadLevelFromFS(fsys fs.FS, path string) (*Level, error) {
	clean := strings.TrimPrefix(filepath.ToSlash(path), "levels/")
	b, err := fs.ReadFile(fsys, clean)
	if err != nil {
		return nil, err
	}
	return loadLevelFromBytes(b)
}

func loadLevelFromBytes(b []byte) (*Level, error) {
	var lvl Level
	if err := json.Unmarshal(b, &lvl); err != nil {
		return nil, err
	}

	if lvl.Width <= 0 || lvl.Height <= 0 {
		return nil, fmt.Errorf("invalid level dimensions: %dx%d", lvl.Width, lvl.Height)
	}

	lvl.tileImg = ebiten.NewImage(common.TileSize, common.TileSize)
	lvl.tileImg.Fill(color.RGBA{R: 0x00, G: 0x00, B: 0xff, A: 0xff})

	// triangle image (always red)
	lvl.triangleImg = triangleImage(common.TileSize, color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff})

	// missing / placeholder image (magenta)
	lvl.missingTileImg = ebiten.NewImage(common.TileSize, common.TileSize)
	lvl.missingTileImg.Fill(color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff})

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
			lvl.layerTileImgs[i] = layerImageFromHex(common.TileSize, lvl.LayerMeta[i].Color)
		}

		// If TilesetUsage metadata exists, preload referenced tileset images
		if lvl.TilesetUsage != nil {
			lvl.tilesetImgs = make(map[string]*ebiten.Image)
			for li := range lvl.TilesetUsage {
				if lvl.TilesetUsage[li] == nil {
					continue
				}
				for y := 0; y < lvl.Height; y++ {
					if y >= len(lvl.TilesetUsage[li]) {
						continue
					}
					for x := 0; x < lvl.Width; x++ {
						if x >= len(lvl.TilesetUsage[li][y]) {
							continue
						}
						entry := lvl.TilesetUsage[li][y][x]
						if entry == nil || entry.Path == "" {
							continue
						}
						if _, ok := lvl.tilesetImgs[entry.Path]; !ok {
							// try load from assets/<path>
							if b, err := os.ReadFile(filepath.Join("assets", entry.Path)); err == nil {
								if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
									lvl.tilesetImgs[entry.Path] = ebiten.NewImageFromImage(img)
								}
							}
						}
					}
				}
			}
		}
	}

	if lvl.BackgroundPath != "" {
		// legacy single background path
		if len(lvl.Backgrounds) == 0 {
			lvl.Backgrounds = []BackgroundEntry{{Path: lvl.BackgroundPath, Parallax: 1.0}}
		}
	}

	// load background images (parallax layers)
	if len(lvl.Backgrounds) > 0 {
		lvl.backgroundImgs = make([]*ebiten.Image, 0, len(lvl.Backgrounds))
		for _, be := range lvl.Backgrounds {
			if be.Path == "" {
				lvl.backgroundImgs = append(lvl.backgroundImgs, nil)
				continue
			}
			if img := loadImageFromPath(be.Path); img != nil {
				lvl.backgroundImgs = append(lvl.backgroundImgs, img)
			} else {
				lvl.backgroundImgs = append(lvl.backgroundImgs, nil)
			}
		}
	}

	return &lvl, nil
}

// Draw renders the level to screen. camX/camY are the camera view's top-left in world coords.
// Tile value 1 draws a red common.TileSize x common.TileSize square.
func (l *Level) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if l == nil || l.tileImg == nil {
		return
	}
	if zoom <= 0 {
		zoom = 1
	}
	offsetX := -camX
	offsetY := -camY

	// draw parallax backgrounds if present
	if l.backgroundImgs != nil && len(l.backgroundImgs) > 0 && len(l.Backgrounds) > 0 {
		for i, bimg := range l.backgroundImgs {
			if bimg == nil {
				continue
			}
			parallax := 1.0
			if i < len(l.Backgrounds) {
				parallax = l.Backgrounds[i].Parallax
			}
			op := &ebiten.DrawImageOptions{}
			bw := float64(bimg.Bounds().Dx())
			bh := float64(bimg.Bounds().Dy())
			if bw > 0 && bh > 0 {
				worldW := float64(l.Width * common.TileSize)
				worldH := float64(l.Height * common.TileSize)
				// scale to world width, keep aspect ratio, then anchor to bottom
				scaleX := worldW / bw
				scaleY := scaleX
				scaledH := bh * scaleY
				baseY := worldH - scaledH
				op.GeoM.Scale(scaleX*zoom, scaleY*zoom)
				offX := camX * (1.0 - parallax)
				offY := camY*(1.0-parallax) + baseY
				op.GeoM.Translate((offX+offsetX)*zoom, (offY+offsetY)*zoom)
				screen.DrawImage(bimg, op)
			}
		}
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
					v := layerTiles[idx]
					if v == 1 {
						op := &ebiten.DrawImageOptions{}
						op.GeoM.Scale(zoom, zoom)
						op.GeoM.Translate((float64(x*common.TileSize)+offsetX)*zoom, (float64(y*common.TileSize)+offsetY)*zoom)
						screen.DrawImage(img, op)
					} else if v == 2 {
						// draw red triangle for value 2
						if l.triangleImg != nil {
							op := &ebiten.DrawImageOptions{}
							op.GeoM.Scale(zoom, zoom)
							op.GeoM.Translate((float64(x*common.TileSize)+offsetX)*zoom, (float64(y*common.TileSize)+offsetY)*zoom)
							screen.DrawImage(l.triangleImg, op)
						}
					} else if v >= 3 {
						// tileset tile: use TilesetUsage metadata if available
						if l.TilesetUsage != nil && layer < len(l.TilesetUsage) {
							usageLayer := l.TilesetUsage[layer]
							if usageLayer != nil && y < len(usageLayer) && x < len(usageLayer[y]) {
								entry := usageLayer[y][x]
								drawn := false
								if entry != nil && entry.Path != "" && l.tilesetImgs != nil {
									if tsImg, ok := l.tilesetImgs[entry.Path]; ok && entry.TileW > 0 && entry.TileH > 0 {
										cols := tsImg.Bounds().Dx() / entry.TileW
										rows := tsImg.Bounds().Dy() / entry.TileH
										if cols > 0 && rows > 0 && entry.Index >= 0 {
											col := entry.Index % cols
											row := entry.Index / cols
											sx := col * entry.TileW
											sy := row * entry.TileH
											// ensure sub-rect fits inside the image
											if sx >= 0 && sy >= 0 && sx+entry.TileW <= tsImg.Bounds().Dx() && sy+entry.TileH <= tsImg.Bounds().Dy() {
												r := image.Rect(sx, sy, sx+entry.TileW, sy+entry.TileH)
												if sub, ok := tsImg.SubImage(r).(*ebiten.Image); ok {
													dop := &ebiten.DrawImageOptions{}
													scaleX := float64(common.TileSize) / float64(entry.TileW)
													scaleY := float64(common.TileSize) / float64(entry.TileH)
													dop.GeoM.Scale(scaleX*zoom, scaleY*zoom)
													dop.GeoM.Translate((float64(x*common.TileSize)+offsetX)*zoom, (float64(y*common.TileSize)+offsetY)*zoom)
													screen.DrawImage(sub, dop)
													drawn = true
												}
											}
										}
									}
								}
								if !drawn {
									// draw placeholder
									if l.missingTileImg != nil {
										op := &ebiten.DrawImageOptions{}
										op.GeoM.Scale(zoom, zoom)
										op.GeoM.Translate((float64(x*common.TileSize)+offsetX)*zoom, (float64(y*common.TileSize)+offsetY)*zoom)
										screen.DrawImage(l.missingTileImg, op)
									}
								}
							}
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

// loadImageFromPath attempts to read an image from a direct path, assets/<path>, or assets/<basename>.
func loadImageFromPath(path string) *ebiten.Image {
	if path == "" {
		return nil
	}
	if img, err := assets.LoadImage(path); err == nil {
		return img
	}
	return nil
}

// Query returns all adjacent non-zero tiles near the provided rect.
// It searches tiles that are within one tile of the rect's bounding tile area.
func (l *Level) Query(r common.Rect) []common.Rect {
	if l == nil {
		return nil
	}
	// compute tile range covering the rect
	minX := int(math.Floor(float64(r.X)/common.TileSize)) - 1
	minY := int(math.Floor(float64(r.Y)/common.TileSize)) - 1
	maxX := int(math.Floor(float64(r.X+r.Width)/common.TileSize)) + 1
	maxY := int(math.Floor(float64(r.Y+r.Height)/common.TileSize)) + 1

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

	var out []common.Rect
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if l.physicsTileAt(x, y) {
				trect := common.Rect{
					X:      float32(x * common.TileSize),
					Y:      float32(y * common.TileSize),
					Width:  common.TileSize,
					Height: common.TileSize,
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
func (l *Level) QueryHorizontal(r common.Rect) []common.Rect {
	if l == nil {
		return nil
	}
	tileLeft := int(math.Floor(float64(r.X) / common.TileSize))
	tileRight := int(math.Floor(float64(r.X+r.Width-1) / common.TileSize))

	leftCol := tileLeft - 1
	rightCol := tileRight + 1

	tileTop := int(math.Floor(float64(r.Y) / common.TileSize))
	tileBottom := int(math.Floor(float64(r.Y+r.Height-1) / common.TileSize))

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

	var out []common.Rect
	cols := []int{leftCol, rightCol}
	for _, x := range cols {
		for y := tileTop; y <= tileBottom; y++ {
			if l.physicsTileAt(x, y) {
				out = append(out, common.Rect{
					X:      float32(x * common.TileSize),
					Y:      float32(y * common.TileSize),
					Width:  common.TileSize,
					Height: common.TileSize,
				})
			}
		}
	}
	return out
}

// QueryVertical returns tiles immediately above and below r.
// It returns non-zero tiles in the row above the rect and the row below the rect,
// over the columns that the rect overlaps.
func (l *Level) QueryVertical(r common.Rect) []common.Rect {
	if l == nil {
		return nil
	}
	tileTop := int(math.Floor(float64(r.Y) / common.TileSize))
	tileBottom := int(math.Floor(float64(r.Y+r.Height-1) / common.TileSize))

	topRow := tileTop - 1
	bottomRow := tileBottom + 1

	tileLeft := int(math.Floor(float64(r.X) / common.TileSize))
	tileRight := int(math.Floor(float64(r.X+r.Width-1) / common.TileSize))

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

	var out []common.Rect
	rows := []int{topRow, bottomRow}
	for _, y := range rows {
		for x := tileLeft; x <= tileRight; x++ {
			if l.physicsTileAt(x, y) {
				out = append(out, common.Rect{
					X:      float32(x * common.TileSize),
					Y:      float32(y * common.TileSize),
					Width:  common.TileSize,
					Height: common.TileSize,
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
func (l *Level) IsGrounded(r common.Rect) bool {
	if l == nil {
		return false
	}
	eps := float32(0.001)
	bottom := r.Y + r.Height
	row := int(math.Floor(float64((bottom + eps) / common.TileSize)))
	if row < 0 || row >= l.Height {
		return false
	}
	left := int(math.Floor(float64(r.X) / common.TileSize))
	right := int(math.Floor(float64((r.X + r.Width - 1) / common.TileSize)))
	if left < 0 {
		left = 0
	}
	if right >= l.Width {
		right = l.Width - 1
	}
	for x := left; x <= right; x++ {
		if l.physicsTileAt(x, row) {
			tileTop := float32(row * common.TileSize)
			if bottom >= tileTop-eps && bottom <= tileTop+eps {
				return true
			}
		}
	}
	return false
}

type wallSide int

const (
	WALL_NONE wallSide = iota
	WALL_LEFT
	WALL_RIGHT
)

func (l *Level) IsTouchingWall(r common.Rect) wallSide {
	if l == nil {
		return WALL_NONE
	}

	eps := float32(0.001)
	left := r.X
	right := r.X + r.Width
	colLeft := int(math.Floor(float64((left - eps) / common.TileSize)))
	colRight := int(math.Floor(float64((right + eps) / common.TileSize)))
	tileTop := int(math.Floor(float64(r.Y) / common.TileSize))
	tileBottom := int(math.Floor(float64((r.Y + r.Height - 1) / common.TileSize)))

	if tileTop < 0 {
		tileTop = 0
	}

	if tileBottom >= l.Height {
		tileBottom = l.Height - 1
	}

	// check left side
	if colLeft >= 0 && colLeft < l.Width {
		for y := tileTop; y <= tileBottom; y++ {
			if l.physicsTileAt(colLeft, y) {
				tileRight := float32((colLeft + 1) * common.TileSize)
				if left >= tileRight-eps && left <= tileRight+eps {
					return WALL_LEFT
				}
			}
		}
	}

	// check right side
	if colRight >= 0 && colRight < l.Width {
		for y := tileTop; y <= tileBottom; y++ {
			if l.physicsTileAt(colRight, y) {
				tileLeft := float32(colRight * common.TileSize)
				if right >= tileLeft-eps && right <= tileLeft+eps {
					return WALL_RIGHT
				}
			}
		}
	}

	return WALL_NONE
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
	return float32(x * common.TileSize), float32(y * common.TileSize)
}
