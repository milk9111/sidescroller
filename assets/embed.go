package assets

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"
	"log"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed *
var assetsFS embed.FS

// PlayerTemplateSheet is the embedded player sprite sheet as an *ebiten.Image.
var PlayerTemplateSheet *ebiten.Image
var PlayerSheet *ebiten.Image

func init() {
	PlayerTemplateSheet = loadImageFromAssets("player_template-Sheet.png")
	PlayerSheet = loadImageFromAssets("player-Sheet.png")
}

// LoadImage loads an embedded asset by assets-relative path.
func LoadImage(path string) (*ebiten.Image, error) {
	clean := cleanAssetPath(path)
	b, err := assetsFS.ReadFile(clean)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

func loadImageFromAssets(path string) *ebiten.Image {
	img, err := LoadImage(path)
	if err != nil {
		log.Fatalf("embed: load %s: %v", path, err)
	}
	return img
}

func cleanAssetPath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		s := filepath.ToSlash(path)
		if idx := strings.LastIndex(s, "/assets/"); idx >= 0 {
			return s[idx+len("/assets/"):]
		}
		return filepath.Base(path)
	}
	s := filepath.ToSlash(path)
	if strings.HasPrefix(s, "assets/") {
		return strings.TrimPrefix(s, "assets/")
	}
	return s
}
