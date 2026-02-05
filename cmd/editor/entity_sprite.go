package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
)

// EntitySpriteSpec describes either a single image or a spritesheet frame.
// It supports JSON values that are either a string (file path) or an object
// with file/row/frame fields.
type EntitySpriteSpec struct {
	File   string `json:"file"`
	Row    int    `json:"row"`
	Frame  int    `json:"frame"`
	FrameW int    `json:"frame_w,omitempty"`
	FrameH int    `json:"frame_h,omitempty"`

	fromObject bool `json:"-"`
}

func (s *EntitySpriteSpec) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	if b[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		s.File = v
		s.Row = 0
		s.Frame = 0
		s.FrameW = 0
		s.FrameH = 0
		s.fromObject = false
		return nil
	}

	// object form
	type alias EntitySpriteSpec
	var tmp alias
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*s = EntitySpriteSpec(tmp)
	s.fromObject = true
	return nil
}

func (s EntitySpriteSpec) IsSheet() bool {
	return s.fromObject
}

type entityDef struct {
	Name   string           `json:"name"`
	Type   string           `json:"type"`
	Sprite EntitySpriteSpec `json:"sprite"`
}

func loadEntityDef(path string) (*entityDef, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var def entityDef
	if err := json.Unmarshal(b, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

func (g *Editor) getEntityIcon(entityFile string) *ebiten.Image {
	if g == nil || entityFile == "" {
		return nil
	}
	if g.entitySpriteCache == nil {
		g.entitySpriteCache = make(map[string]*ebiten.Image)
	}
	key := "entity:" + entityFile
	if img, ok := g.entitySpriteCache[key]; ok && img != nil {
		return img
	}
	def, err := loadEntityDef(filepath.Join("entities", entityFile))
	if err != nil {
		return nil
	}
	img, err := loadEntitySpriteImage(def.Sprite, g.entitySpriteCache)
	if err != nil {
		return nil
	}
	if img != nil {
		g.entitySpriteCache[key] = img
	}
	return img
}

func loadEntitySpriteImage(spec EntitySpriteSpec, cache map[string]*ebiten.Image) (*ebiten.Image, error) {
	if spec.File == "" {
		return nil, errors.New("empty sprite file")
	}
	if cache == nil {
		cache = make(map[string]*ebiten.Image)
	}

	key := spriteCacheKey(spec)
	if img, ok := cache[key]; ok && img != nil {
		return img, nil
	}

	sheetKey := "sheet:" + spec.File
	sheet := cache[sheetKey]
	if sheet == nil {
		img, err := loadImageFromAssetsOrFS(spec.File)
		if err != nil {
			return nil, err
		}
		sheet = img
		cache[sheetKey] = sheet
	}

	if !spec.IsSheet() {
		cache[key] = sheet
		return sheet, nil
	}

	frameW, frameH := spec.FrameW, spec.FrameH
	if frameW <= 0 || frameH <= 0 {
		frameW, frameH = inferSpriteSheetFrameSize(sheet, spec.Row, spec.Frame)
	}
	if frameW <= 0 || frameH <= 0 {
		cache[key] = sheet
		return sheet, nil
	}

	sx := spec.Frame * frameW
	sy := spec.Row * frameH
	if sx < 0 || sy < 0 || sx+frameW > sheet.Bounds().Dx() || sy+frameH > sheet.Bounds().Dy() {
		cache[key] = sheet
		return sheet, nil
	}
	if sub, ok := sheet.SubImage(image.Rect(sx, sy, sx+frameW, sy+frameH)).(*ebiten.Image); ok {
		cache[key] = sub
		return sub, nil
	}

	cache[key] = sheet
	return sheet, nil
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

func spriteCacheKey(spec EntitySpriteSpec) string {
	if !spec.IsSheet() {
		return "sprite:" + spec.File
	}
	return fmt.Sprintf("sprite:%s|r=%d|f=%d|w=%d|h=%d", spec.File, spec.Row, spec.Frame, spec.FrameW, spec.FrameH)
}

func inferSpriteSheetFrameSize(sheet *ebiten.Image, row, frame int) (int, int) {
	if sheet == nil {
		return 0, 0
	}
	w := sheet.Bounds().Dx()
	h := sheet.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	if row < 0 {
		row = 0
	}
	if frame < 0 {
		frame = 0
	}

	rows := pickRowCount(h, row)
	frameH := h / rows
	cols := pickColCount(w, frame, frameH)
	frameW := w / cols
	return frameW, frameH
}

func pickRowCount(height int, row int) int {
	if height <= 0 {
		return 1
	}
	bestRows := 1
	bestScore := -1
	for r := 1; r <= height; r++ {
		if height%r != 0 {
			continue
		}
		if r <= row {
			continue
		}
		frameH := height / r
		if frameH < 16 || frameH > 256 {
			continue
		}
		score := 0
		if frameH%16 == 0 {
			score += 4
		} else if frameH%8 == 0 {
			score += 2
		}
		score -= abs(frameH-64) / 8
		if r%2 == 0 {
			score += 1
		}
		if score > bestScore {
			bestScore = score
			bestRows = r
		}
	}
	if bestRows <= row {
		bestRows = row + 1
	}
	return bestRows
}

func pickColCount(width int, frame int, frameH int) int {
	if width <= 0 {
		return 1
	}
	// Prefer square frames when width divides evenly by frame height.
	if frameH > 0 && width%frameH == 0 {
		cols := width / frameH
		if cols > frame {
			return cols
		}
	}
	bestCols := 1
	bestScore := -1
	for c := 1; c <= width; c++ {
		if width%c != 0 {
			continue
		}
		if c <= frame {
			continue
		}
		frameW := width / c
		if frameW < 16 || frameW > 256 {
			continue
		}
		score := 0
		if frameW%16 == 0 {
			score += 2
		} else if frameW%8 == 0 {
			score += 1
		}
		if c%4 == 0 {
			score += 5
		} else if c%3 == 0 {
			score += 3
		} else if c%2 == 0 {
			score += 1
		}
		if frameH > 0 {
			score -= abs(frameW-frameH) / 8
		}
		if score > bestScore || (score == bestScore && c%4 == 0) {
			bestScore = score
			bestCols = c
		}
	}
	if bestCols <= frame {
		bestCols = frame + 1
	}
	return bestCols
}
