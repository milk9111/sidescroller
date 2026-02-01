package component

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// OutlineComponent is responsible for generating and caching per-layer
// outline images (offscreen) and drawing them to the screen.
type OutlineComponent struct {
	LayerShapeImgs   []*ebiten.Image
	LayerOutlineImgs []*ebiten.Image
	LayerReady       []bool
	OutlineShader    *ebiten.Shader
}

// NewOutlineComponent creates a component from the provided full-size
// layer shape images and optional compiled shader.
func NewOutlineComponent(shapeImgs []*ebiten.Image, shader *ebiten.Shader) *OutlineComponent {
	oc := &OutlineComponent{
		LayerShapeImgs:   shapeImgs,
		LayerOutlineImgs: make([]*ebiten.Image, len(shapeImgs)),
		LayerReady:       make([]bool, len(shapeImgs)),
		OutlineShader:    shader,
	}
	return oc
}

// DrawLayer ensures the outline for the given layer is generated once
// and draws it to the provided screen at the specified offset/zoom.
func (o *OutlineComponent) DrawLayer(screen *ebiten.Image, layer int, offsetX, offsetY, zoom float64) {
	if o == nil || layer < 0 || layer >= len(o.LayerShapeImgs) {
		return
	}

	// If already generated, draw cached outline
	if o.LayerOutlineImgs != nil && o.LayerOutlineImgs[layer] != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoom, zoom)
		op.GeoM.Translate(offsetX*zoom, offsetY*zoom)
		screen.DrawImage(o.LayerOutlineImgs[layer], op)
		return
	}

	src := o.LayerShapeImgs[layer]
	if src == nil {
		return
	}

	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	if sw <= 0 || sh <= 0 {
		return
	}

	// Try shader path first
	if o.OutlineShader != nil {
		off := ebiten.NewImage(sw, sh)
		imgs := [4]*ebiten.Image{src, nil, nil, nil}
		drop := &ebiten.DrawRectShaderOptions{
			Images: imgs,
			Uniforms: map[string]interface{}{
				"TexelSize":    []float32{1.0 / float32(sw), 1.0 / float32(sh)},
				"Threshold":    float32(0.01),
				"OutlineColor": []float32{0.0, 0.0, 0.0, 1.0},
			},
		}
		off.DrawRectShader(sw, sh, o.OutlineShader, drop)
		o.LayerOutlineImgs[layer] = off
		o.LayerReady[layer] = true
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoom, zoom)
		op.GeoM.Translate(offsetX*zoom, offsetY*zoom)
		screen.DrawImage(off, op)
		return
	}

	// Fallback: CPU-based outline generation by sampling the ebiten.Image.
	outline := generateOutlineFromEbiten(src, 1, color.RGBA{0, 0, 0, 0xff})
	o.LayerOutlineImgs[layer] = outline
	o.LayerReady[layer] = true
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(offsetX*zoom, offsetY*zoom)
	screen.DrawImage(outline, op)
}

// generateOutlineFromEbiten is a local copy of the pixel-scan outline
// generator. It is safe to call from the game loop because it reads pixels
// via At() on the supplied *ebiten.Image.
func generateOutlineFromEbiten(src *ebiten.Image, thickness int, outlineCol color.RGBA) *ebiten.Image {
	w, h := src.Size()
	outRGBA := image.NewRGBA(image.Rect(0, 0, w, h))

	isOpaque := func(x, y int) bool {
		if x < 0 || y < 0 || x >= w || y >= h {
			return false
		}
		_, _, _, a := src.At(x, y).RGBA()
		return a != 0
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if isOpaque(x, y) {
				continue
			}
			found := false
			ymin := y - thickness
			if ymin < 0 {
				ymin = 0
			}
			ymax := y + thickness
			if ymax >= h {
				ymax = h - 1
			}
			xmin := x - thickness
			if xmin < 0 {
				xmin = 0
			}
			xmax := x + thickness
			if xmax >= w {
				xmax = w - 1
			}
			for yy := ymin; yy <= ymax && !found; yy++ {
				for xx := xmin; xx <= xmax; xx++ {
					if isOpaque(xx, yy) {
						found = true
						break
					}
				}
			}
			if found {
				outRGBA.Set(x, y, outlineCol)
			}
		}
	}

	return ebiten.NewImageFromImage(outRGBA)
}
