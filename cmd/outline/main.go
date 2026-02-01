package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 800
	screenHeight = 600
	squareSize   = 64
)

type Game struct {
	sq           *ebiten.Image
	outline      *ebiten.Image
	outlineReady bool
	shader       *ebiten.Shader
}

func NewGame() *Game {
	// build a non-uniform shape by stacking small squares into a CPU RGBA
	tile := 24
	cols := 8
	rows := 6
	w := cols * tile
	h := rows * tile
	srcRGBA := image.NewRGBA(image.Rect(0, 0, w, h))

	// draw a few tiles to make a non-uniform shape
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	positions := []image.Point{
		{1, 2}, {2, 2}, {3, 2}, {4, 2},
		{2, 1}, {2, 3}, {4, 1}, {4, 3},
		{6, 2}, {6, 3}, {5, 3},
	}
	for _, p := range positions {
		r := image.Rect(p.X*tile, p.Y*tile, p.X*tile+tile, p.Y*tile+tile)
		draw.Draw(srcRGBA, r, &image.Uniform{white}, image.Point{}, draw.Src)
	}

	// create ebiten image for the shape; outline will be generated once the game starts
	shapeImg := ebiten.NewImageFromImage(srcRGBA)

	// try to load shader source
	var sh *ebiten.Shader
	if b, err := os.ReadFile("shaders/outline.kage"); err == nil {
		if s, err := ebiten.NewShader(b); err == nil {
			sh = s
		} else {
			log.Printf("outline shader compile error: %v", err)
		}
	} else {
		log.Printf("outline shader not found: %v", err)
	}

	return &Game{sq: shapeImg, outline: nil, outlineReady: false, shader: sh}
}

// GenerateOutlineFromRGBA returns an RGBA image containing outline pixels (outlineCol)
// around the opaque areas of src. Thickness is in pixels.
func GenerateOutlineFromRGBA(src *image.RGBA, thickness int, outlineCol color.RGBA) *image.RGBA {
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))

	pix := src.Pix

	isOpaque := func(x, y int) bool {
		if x < 0 || y < 0 || x >= w || y >= h {
			return false
		}
		idx := (y*w + x) * 4 // RGBA
		return pix[idx+3] != 0
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
				out.Set(x+b.Min.X, y+b.Min.Y, outlineCol)
			}
		}
	}
	return out
}

// GenerateOutlineFromEbiten generates an outline image from an ebiten.Image source.
// This reads pixels via At (safe to call after the game has started) and returns
// an *ebiten.Image containing the outline.
func GenerateOutlineFromEbiten(src *ebiten.Image, thickness int, outlineCol color.RGBA) *ebiten.Image {
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

func (g *Game) Update() error { return nil }
func (g *Game) Draw(screen *ebiten.Image) {
	// generate outline via shader on first Draw (safe after game start)
	if !g.outlineReady && g.shader != nil {
		sw, sh := g.sq.Size()
		off := ebiten.NewImage(sw, sh)
		imgs := [4]*ebiten.Image{g.sq, nil, nil, nil}
		drop := &ebiten.DrawRectShaderOptions{
			Images: imgs,
			Uniforms: map[string]interface{}{
				"TexelSize":    []float32{1.0 / float32(sw), 1.0 / float32(sh)},
				"Threshold":    float32(0.01),
				"OutlineColor": []float32{1.0, 0.0, 0.0, 1.0},
			},
		}
		off.DrawRectShader(sw, sh, g.shader, drop)
		g.outline = off
		g.outlineReady = true
	}

	w, h := screen.Size()
	sw, sh := g.sq.Size()
	ox := (w - sw) / 2
	oy := (h - sh) / 2

	// draw outline first (if available)
	if g.outline != nil {
		opOutline := &ebiten.DrawImageOptions{}
		opOutline.GeoM.Translate(float64(ox), float64(oy))
		screen.DrawImage(g.outline, opOutline)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(ox), float64(oy))
	screen.DrawImage(g.sq, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Outline Test")
	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
