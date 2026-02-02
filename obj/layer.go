package obj

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/common"
)

// Layer owns the tile data and drawing/update logic for a single layer.
type Layer struct {
	Index int
	Level *Level
	Tiles []int
	Usage [][]*TilesetEntry
	Meta  *LayerMeta

	// references to level-wide images (kept here for convenience)
	TileImg    *ebiten.Image
	ShapeImg   *ebiten.Image
	OutlineImg *ebiten.Image
	MissingImg *ebiten.Image
}

// NewLayer constructs a Layer from a Level and layer index.
func NewLayer(l *Level, idx int) *Layer {
	var tiles []int
	if l.Layers != nil && idx < len(l.Layers) {
		tiles = l.Layers[idx]
	}
	var usage [][]*TilesetEntry
	if l.TilesetUsage != nil && idx < len(l.TilesetUsage) {
		usage = l.TilesetUsage[idx]
	}
	var meta *LayerMeta
	if l.LayerMeta != nil && idx < len(l.LayerMeta) {
		meta = &l.LayerMeta[idx]
	}
	var tileImg *ebiten.Image
	if l.layerTileImgs != nil && idx < len(l.layerTileImgs) {
		tileImg = l.layerTileImgs[idx]
	}
	var shapeImg *ebiten.Image
	if l.layerShapeImgs != nil && idx < len(l.layerShapeImgs) {
		shapeImg = l.layerShapeImgs[idx]
	}
	var outlineImg *ebiten.Image
	if l.layerOutlineImgs != nil && idx < len(l.layerOutlineImgs) {
		outlineImg = l.layerOutlineImgs[idx]
	}

	return &Layer{
		Index:      idx,
		Level:      l,
		Tiles:      tiles,
		Usage:      usage,
		Meta:       meta,
		TileImg:    tileImg,
		ShapeImg:   shapeImg,
		OutlineImg: outlineImg,
		MissingImg: l.missingTileImg,
	}
}

// Draw draws this layer to screen with optional parallax from Meta.
func (ly *Layer) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if ly == nil || ly.Level == nil || ly.Tiles == nil {
		return
	}
	l := ly.Level
	parallax := 1.0
	if ly.Meta != nil && ly.Meta.Parallax > 0 {
		parallax = ly.Meta.Parallax
	}
	layerOffsetX := -camX * parallax
	layerOffsetY := -camY * parallax

	if ly.OutlineImg != nil {
		opOutline := &ebiten.DrawImageOptions{}
		opOutline.GeoM.Scale(zoom, zoom)
		opOutline.GeoM.Translate(layerOffsetX*zoom, layerOffsetY*zoom)
		screen.DrawImage(ly.OutlineImg, opOutline)
	}

	img := ly.TileImg
	if img == nil {
		img = l.tileImg
	}

	for y := 0; y < l.Height; y++ {
		for x := 0; x < l.Width; x++ {
			idx := y*l.Width + x
			if idx < 0 || idx >= len(ly.Tiles) {
				continue
			}
			v := ly.Tiles[idx]
			if v == 1 {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(zoom, zoom)
				op.GeoM.Translate((float64(x*common.TileSize)+layerOffsetX)*zoom, (float64(y*common.TileSize)+layerOffsetY)*zoom)
				screen.DrawImage(img, op)
			} else if v == 2 {
				if l.triangleImg != nil {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(zoom, zoom)
					op.GeoM.Translate((float64(x*common.TileSize)+layerOffsetX)*zoom, (float64(y*common.TileSize)+layerOffsetY)*zoom)
					screen.DrawImage(l.triangleImg, op)
				}
			} else if v >= 3 {
				if ly.Usage != nil && y < len(ly.Usage) && x < len(ly.Usage[y]) {
					entry := ly.Usage[y][x]
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
								if sx >= 0 && sy >= 0 && sx+entry.TileW <= tsImg.Bounds().Dx() && sy+entry.TileH <= tsImg.Bounds().Dy() {
									r := image.Rect(sx, sy, sx+entry.TileW, sy+entry.TileH)
									if sub, ok := tsImg.SubImage(r).(*ebiten.Image); ok {
										dop := &ebiten.DrawImageOptions{}
										scaleX := float64(common.TileSize) / float64(entry.TileW)
										scaleY := float64(common.TileSize) / float64(entry.TileH)
										dop.GeoM.Scale(scaleX*zoom, scaleY*zoom)
										dop.GeoM.Translate((float64(x*common.TileSize)+layerOffsetX)*zoom, (float64(y*common.TileSize)+layerOffsetY)*zoom)
										screen.DrawImage(sub, dop)
										drawn = true
									}
								}
							}
						}
					}
					if !drawn {
						var tsBounds image.Rectangle
						var tsPresent bool
						if entry != nil {
							if tsImg, ok := l.tilesetImgs[entry.Path]; ok && tsImg != nil {
								tsBounds = tsImg.Bounds()
								tsPresent = true
							}
							log.Printf("tileset draw failed: path=%q index=%d tileW=%d tileH=%d tsPresent=%v tsBounds=%v", entry.Path, entry.Index, entry.TileW, entry.TileH, tsPresent, tsBounds)
						} else {
							log.Printf("tileset draw failed: missing entry at layer=%d x=%d y=%d", ly.Index, x, y)
						}
						if ly.MissingImg != nil {
							op := &ebiten.DrawImageOptions{}
							op.GeoM.Scale(zoom, zoom)
							op.GeoM.Translate((float64(x*common.TileSize)+layerOffsetX)*zoom, (float64(y*common.TileSize)+layerOffsetY)*zoom)
							screen.DrawImage(ly.MissingImg, op)
						}
					}
				}
			}
		}
	}
}

// Update is a placeholder for per-layer update logic (e.g., animated tiles).
func (ly *Layer) Update() {
	// currently no per-layer update logic; placeholder for future features.
}
