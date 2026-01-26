package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Background encapsulates background entries and their scaled images.
type Background struct {
	Entries []BackgroundEntry
	Images  []*ebiten.Image
}

// NewBackground constructs an empty Background manager.
func NewBackground() *Background {
	return &Background{}
}

// Add registers a background entry and stores a scaled image for drawing.
// `path` is the reference path/name recorded in the Level; `img` is the decoded image to use.
func (b *Background) Add(path string, img image.Image, lvl *Level, cellSize int) {
	if lvl == nil {
		return
	}
	be := BackgroundEntry{Path: path, Parallax: 0.5}
	b.Entries = append(b.Entries, be)
	if b.Images == nil {
		b.Images = make([]*ebiten.Image, 0, len(b.Entries))
	}
	// scale image to cover the logical canvas size
	tw := lvl.Width * cellSize
	ht := lvl.Height * cellSize
	b.Images = append(b.Images, scaleImageToCanvas(img, tw, ht))
}

// Draw renders any loaded background images onto the provided canvas image
// using the provided canvas transform (zoom/pan).
func (b *Background) Draw(canvas *ebiten.Image, zoom, offsetX, offsetY float64) {
	if b == nil || b.Images == nil || len(b.Images) == 0 {
		return
	}
	for _, img := range b.Images {
		if img == nil {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, 0)
		op.GeoM.Scale(zoom, zoom)
		op.GeoM.Translate(offsetX, offsetY)
		canvas.DrawImage(img, op)
	}
}

// Update is reserved for per-frame background updates (parallax/animation).
func (b *Background) Update() {
	// no-op for now
}
