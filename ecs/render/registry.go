package render

import "github.com/hajimehoshi/ebiten/v2"

var images = map[string]*ebiten.Image{}

// RegisterImage stores an image by key.
func RegisterImage(key string, img *ebiten.Image) {
	if key == "" || img == nil {
		return
	}
	images[key] = img
}

// GetImage returns a cached image by key.
func GetImage(key string) *ebiten.Image {
	if key == "" {
		return nil
	}
	return images[key]
}
