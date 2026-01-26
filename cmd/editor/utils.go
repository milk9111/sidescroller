package main

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// layerImageFromHex creates an image filled with the provided hex color ("#rrggbb").
func layerImageFromHex(size int, hex string) *ebiten.Image {
	c := parseHexColor(hex)
	img := ebiten.NewImage(size, size)
	img.Fill(c)
	return img
}

// parseHexColor parses a color in the form #rrggbb. Returns opaque color if parse fails.
func parseHexColor(s string) color.RGBA {
	var r, g, b uint8 = 0x3c, 0x78, 0xff
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

// circleImage builds an RGBA image with a filled circle of the given color.
func circleImage(size int, col color.RGBA) *ebiten.Image {
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	cx := float64(size) / 2
	cy := float64(size) / 2
	r := float64(size)/2 - 2
	rr := r * r
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= rr {
				rgba.Set(x, y, col)
			} else {
				// transparent
				rgba.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
	return ebiten.NewImageFromImage(rgba)
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

// scaleImageToCanvas scales the decoded image to the requested target dimensions
// and returns an *ebiten.Image containing the scaled pixels.
func scaleImageToCanvas(img image.Image, targetW, targetH int) *ebiten.Image {
	if img == nil || targetW <= 0 || targetH <= 0 {
		return nil
	}
	src := ebiten.NewImageFromImage(img)
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	if sw == targetW && sh == targetH {
		return src
	}
	dst := ebiten.NewImage(targetW, targetH)
	op := &ebiten.DrawImageOptions{}
	sx := float64(targetW) / float64(sw)
	sy := float64(targetH) / float64(sh)
	op.GeoM.Scale(sx, sy)
	op.Filter = ebiten.FilterNearest
	dst.DrawImage(src, op)
	return dst
}

// screenToCanvasPoint converts screen coordinates (sx,sy) into canvas-local
// unzoomed coordinates (pixels relative to level origin). panelRight is the
// X coordinate on screen where the right UI panel begins (canvas clip on right).
func (g *Editor) screenToCanvasPoint(sx, sy int, panelRight int) (float64, float64, bool) {
	if sx < leftPanelWidth || sx >= panelRight {
		return 0, 0, false
	}
	lx := float64(sx - leftPanelWidth)
	ly := float64(sy)
	if g.canvasZoom == 0 {
		g.canvasZoom = 1.0
	}
	cx := (lx - g.canvasOffsetX) / g.canvasZoom
	cy := (ly - g.canvasOffsetY) / g.canvasZoom
	return cx, cy, true
}

// normalizeAssetPath converts an absolute or assets-prefixed path to an assets-relative path.
func normalizeAssetPath(path string) string {
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		p := filepath.ToSlash(path)
		if strings.HasPrefix(p, "assets/") {
			return strings.TrimPrefix(p, "assets/")
		}
		return p
	}

	absPath, err := filepath.Abs(path)
	if err == nil {
		assetsDir, err := filepath.Abs("assets")
		if err == nil {
			abs := filepath.ToSlash(absPath)
			assets := filepath.ToSlash(assetsDir)
			if strings.HasPrefix(abs, assets+"/") {
				return strings.TrimPrefix(abs, assets+"/")
			}
		}
	}

	return filepath.Base(path)
}

func normalizeBackgroundPaths(level *Level) {
	if level == nil || level.Backgrounds == nil {
		return
	}
	for i := range level.Backgrounds {
		level.Backgrounds[i].Path = normalizeAssetPath(level.Backgrounds[i].Path)
	}
}
