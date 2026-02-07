package render

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
)

// LoadImage loads an image from assets or filesystem and caches it by key.
func LoadImage(key string) (*ebiten.Image, error) {
	if key == "" {
		return nil, fmt.Errorf("empty image key")
	}
	if img := GetImage(key); img != nil {
		return img, nil
	}
	img, err := loadImageFromAssetsOrFS(key)
	if err != nil {
		return nil, err
	}
	RegisterImage(key, img)
	return img, nil
}

func loadImageFromAssetsOrFS(path string) (*ebiten.Image, error) {
	if img, err := assets.LoadImage(path); err == nil {
		return img, nil
	}
	tried := []string{path, filepath.Join("assets", path), filepath.Base(path)}
	for _, p := range tried {
		if b, err := os.ReadFile(p); err == nil {
			if im, _, err := image.Decode(bytes.NewReader(b)); err == nil {
				return ebiten.NewImageFromImage(im), nil
			}
		}
	}
	return nil, fmt.Errorf("failed to load image %s", path)
}
